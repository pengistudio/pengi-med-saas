package sri_document

import (
	"errors"
	"fmt"
	"pengi-med-saas/core/tenantdb"
	"pengi-med-saas/core/tenantfiles"
	"strings"
	"testing"
	"time"

	billing_models "pengi-med-saas/features/billing/models"
	tenant_models "pengi-med-saas/features/tenants/models"
	"pengi-med-saas/testutils"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ─── Fakes ───────────────────────────────────────────────────────────────────

type fakeGateway struct {
	signErr      error
	receptionErr error
	auth         Authorization
	authErr      error

	signCalls      int
	receptionCalls int
	authCalls      int
	signedInputs   []string
}

func (g *fakeGateway) Sign(p12 []byte, password string, xml string) (string, error) {
	g.signCalls++
	g.signedInputs = append(g.signedInputs, xml)
	if g.signErr != nil {
		return "", g.signErr
	}
	return "<signed>" + xml + "</signed>", nil
}

func (g *fakeGateway) SubmitForReception(signedXML string, sriEnv string) error {
	g.receptionCalls++
	return g.receptionErr
}

func (g *fakeGateway) QueryAuthorization(accessKey string, sriEnv string) (Authorization, error) {
	g.authCalls++
	return g.auth, g.authErr
}

type published struct {
	queue string
	body  string
}

type fakePublisher struct {
	messages []published
}

func (p *fakePublisher) Publish(queue string, body []byte) error {
	p.messages = append(p.messages, published{queue, string(body)})
	return nil
}

// ─── Fixture ─────────────────────────────────────────────────────────────────

type fixture struct {
	t          *testing.T
	db         *gorm.DB
	gateway    *fakeGateway
	files      *tenantfiles.MemoryStore
	publisher  *fakePublisher
	docs       *Lifecycle
	kind       Kind
	tenant     tenant_models.Tenant
	authorized []uint
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	db := testutils.SetupTestDB(t, &tenant_models.Tenant{}, &billing_models.Invoice{})

	now := time.Now().UnixNano()
	tenant := tenant_models.Tenant{
		Name:         "SRI Tenant",
		Slug:         fmt.Sprintf("sri-%d", now),
		DisplayToken: fmt.Sprintf("tok-sri-%d", now),
		TaxID:        "1790011223001",
		SriP12Path:   "certs/tenant.p12",
		SriPassword:  "secret",
	}
	if err := db.Create(&tenant).Error; err != nil {
		t.Fatalf("create tenant: %v", err)
	}

	f := &fixture{
		t:         t,
		db:        db,
		gateway:   &fakeGateway{auth: Authorization{Status: Authorized, AuthorizedAt: time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)}},
		files:     tenantfiles.Memory(),
		publisher: &fakePublisher{},
		tenant:    tenant,
	}
	// A document kind over the invoices table whose XML is just its access key,
	// so tests can see exactly which key each attempt signed.
	f.kind = Kind{
		Name:           "invoice",
		Queue:          "invoice_tasks",
		Folder:         "invoices",
		Model:          &billing_models.Invoice{},
		MessageIDField: "invoice_id",
		BuildXML: func(db *gorm.DB, id uint, tenant tenant_models.Tenant, accessKey string, sriEnv string) (string, error) {
			return "<factura>" + accessKey + "</factura>", nil
		},
		OnAuthorized: func(db *gorm.DB, docs Documents, id uint, tenant tenant_models.Tenant, sriEnv string) error {
			f.authorized = append(f.authorized, id)
			return nil
		},
	}
	_ = f.files.Write(tenant.ID, "signature.p12", []byte("p12"))
	f.docs = New(db, zap.NewNop(), f.gateway, f.publisher, Documents{Files: f.files}, "1")
	return f
}

func (f *fixture) invoice(status string) billing_models.Invoice {
	f.t.Helper()
	inv := billing_models.Invoice{
		TenantID:     f.tenant.ID,
		DocumentCode: "01",
		EmissionType: "1",
		Sequential:   "000000123",
		// Noon, not midnight: Postgres returns timestamptz in the process's
		// local zone, and the access key formats the date in that zone (as
		// production does, with TZ=America/Guayaquil). Midnight UTC would be
		// 31/08 anywhere west of UTC; noon UTC is 01/09 in every zone ±11h.
		IssueDate:         time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC),
		EstablishmentCode: "001",
		EmissionPointCode: "001",
		Status:            status,
	}
	if err := f.db.Create(&inv).Error; err != nil {
		f.t.Fatalf("create invoice: %v", err)
	}
	return inv
}

func (f *fixture) reload(id uint) billing_models.Invoice {
	f.t.Helper()
	var inv billing_models.Invoice
	if err := f.db.Unscoped().First(&inv, id).Error; err != nil {
		f.t.Fatalf("reload invoice: %v", err)
	}
	return inv
}

func (f *fixture) process(id uint) error {
	return f.docs.Process(f.kind, uint64(id))
}

// ─── Process ─────────────────────────────────────────────────────────────────

func TestProcess_AuthorizesPendingDocument(t *testing.T) {
	f := newFixture(t)
	inv := f.invoice(billing_models.InvoiceStatusPending)

	if err := f.process(inv.ID); err != nil {
		t.Fatalf("process: %v", err)
	}

	got := f.reload(inv.ID)
	if got.Status != billing_models.InvoiceStatusAuthorized {
		t.Fatalf("status = %q, want authorized", got.Status)
	}
	if got.AccessKey == nil || len(*got.AccessKey) != 49 {
		t.Fatalf("access key = %v, want a 49-digit key", got.AccessKey)
	}
	if !strings.HasPrefix(*got.AccessKey, "01092026"+"01"+"1790011223001"+"1"+"001001"+"000000123") {
		t.Fatalf("access key %q does not encode the document", *got.AccessKey)
	}
	if got.AuthorizedAt == nil || !got.AuthorizedAt.Equal(time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)) {
		t.Fatalf("authorized_at = %v, want SRI's fechaAutorizacion", got.AuthorizedAt)
	}
	if xml := f.signedXML(got); xml != "<signed><factura>"+*got.AccessKey+"</factura></signed>" {
		t.Fatalf("stored signed XML = %q", xml)
	}
	if len(f.authorized) != 1 || f.authorized[0] != inv.ID {
		t.Fatalf("OnAuthorized calls = %v, want [%d]", f.authorized, inv.ID)
	}
}

func TestProcess_ReusesExistingAccessKey(t *testing.T) {
	f := newFixture(t)
	inv := f.invoice(billing_models.InvoiceStatusFailed)
	key := "0109202601179001122300110010010000001231234567810"
	f.db.Model(&inv).Update("access_key", key)

	if err := f.process(inv.ID); err != nil {
		t.Fatalf("process: %v", err)
	}

	if got := f.reload(inv.ID); got.AccessKey == nil || *got.AccessKey != key {
		t.Fatalf("access key = %v, want the original %s", got.AccessKey, key)
	}
	if len(f.gateway.signedInputs) != 1 || f.gateway.signedInputs[0] != "<factura>"+key+"</factura>" {
		t.Fatalf("signed XML = %v, want it built with the original key", f.gateway.signedInputs)
	}
}

func TestProcess_SignerUnreachable_IsConnectionErrorAndRetryable(t *testing.T) {
	f := newFixture(t)
	inv := f.invoice(billing_models.InvoiceStatusPending)
	f.gateway.signErr = fmt.Errorf("sign: %w", ErrUnreachable)

	err := f.process(inv.ID)
	if err == nil || !IsRetryable(err) {
		t.Fatalf("err = %v, want a retryable error", err)
	}

	got := f.reload(inv.ID)
	if got.Status != billing_models.InvoiceStatusConnectionError {
		t.Fatalf("status = %q, want connection_error", got.Status)
	}
	if got.ErrorCode == nil || *got.ErrorCode != ErrorCodeConnection {
		t.Fatalf("error_code = %v, want %s", got.ErrorCode, ErrorCodeConnection)
	}
	if got.ErrorMessage == nil || !strings.Contains(*got.ErrorMessage, "unreachable") {
		t.Fatalf("error_message = %v, want the raw cause", got.ErrorMessage)
	}
	if xml := f.signedXML(got); xml != "" {
		t.Fatalf("stored signed XML = %q, want none", xml)
	}
}

func TestProcess_ReturnedBySri_FailsKeepingKeyAndRemovingXml(t *testing.T) {
	f := newFixture(t)
	inv := f.invoice(billing_models.InvoiceStatusPending)
	f.gateway.receptionErr = &Rejection{Message: "[SRI-ERROR] ARCHIVO NO CUMPLE ESTRUCTURA XML (ID 35)"}

	err := f.process(inv.ID)
	if err == nil || IsRetryable(err) {
		t.Fatalf("err = %v, want a non-retryable error", err)
	}

	got := f.reload(inv.ID)
	if got.Status != billing_models.InvoiceStatusFailed {
		t.Fatalf("status = %q, want failed", got.Status)
	}
	if got.ErrorCode == nil || *got.ErrorCode != ErrorCodeReturned {
		t.Fatalf("error_code = %v, want %s", got.ErrorCode, ErrorCodeReturned)
	}
	if got.ErrorMessage == nil || !strings.Contains(*got.ErrorMessage, "(ID 35)") {
		t.Fatalf("error_message = %v, want SRI's reason", got.ErrorMessage)
	}
	if got.AccessKey == nil {
		t.Fatalf("access key was cleared; it must survive failures")
	}
	if xml := f.signedXML(got); xml != "" {
		t.Fatalf("stored signed XML = %q, want it removed", xml)
	}
}

func TestProcess_ReceptionUnreachable_RetryResendsSameKey(t *testing.T) {
	f := newFixture(t)
	inv := f.invoice(billing_models.InvoiceStatusPending)
	f.gateway.receptionErr = fmt.Errorf("submit: %w", ErrUnreachable)

	if err := f.process(inv.ID); !IsRetryable(err) {
		t.Fatalf("err = %v, want retryable", err)
	}
	if got := f.reload(inv.ID); got.Status != billing_models.InvoiceStatusConnectionError {
		t.Fatalf("status = %q, want connection_error", got.Status)
	}

	f.gateway.receptionErr = nil
	if err := f.process(inv.ID); err != nil {
		t.Fatalf("retry: %v", err)
	}

	got := f.reload(inv.ID)
	if got.Status != billing_models.InvoiceStatusAuthorized {
		t.Fatalf("status = %q, want authorized", got.Status)
	}
	if len(f.gateway.signedInputs) != 2 || f.gateway.signedInputs[0] != f.gateway.signedInputs[1] {
		t.Fatalf("signed XML per attempt = %v, want the same key both times", f.gateway.signedInputs)
	}
}

func TestProcess_AuthorizationPending_LaterAttemptOnlyQueriesAuthorization(t *testing.T) {
	f := newFixture(t)
	inv := f.invoice(billing_models.InvoiceStatusPending)
	f.gateway.auth = Authorization{Status: AuthorizationPending}

	if err := f.process(inv.ID); err != nil {
		t.Fatalf("process: %v", err)
	}
	received := f.reload(inv.ID)
	if received.Status != billing_models.InvoiceStatusValidated {
		t.Fatalf("status = %q, want validated", received.Status)
	}

	f.gateway.auth = Authorization{Status: Authorized, AuthorizedAt: time.Date(2026, 9, 1, 11, 0, 0, 0, time.UTC)}
	if err := f.process(inv.ID); err != nil {
		t.Fatalf("second process: %v", err)
	}

	got := f.reload(inv.ID)
	if got.Status != billing_models.InvoiceStatusAuthorized {
		t.Fatalf("status = %q, want authorized", got.Status)
	}
	if f.gateway.signCalls != 1 || f.gateway.receptionCalls != 1 {
		t.Fatalf("sign=%d reception=%d, want the received comprobante never re-signed or resent", f.gateway.signCalls, f.gateway.receptionCalls)
	}
	if *got.AccessKey != *received.AccessKey {
		t.Fatalf("access key changed from %s to %s", *received.AccessKey, *got.AccessKey)
	}
	if len(f.authorized) != 1 {
		t.Fatalf("OnAuthorized calls = %v, want one", f.authorized)
	}
}

func TestProcess_AuthorizationUnreachable_StaysReceived(t *testing.T) {
	f := newFixture(t)
	inv := f.invoice(billing_models.InvoiceStatusPending)
	f.gateway.authErr = fmt.Errorf("authorize: %w", ErrUnreachable)

	if err := f.process(inv.ID); err != nil {
		t.Fatalf("process: %v, want nil (the next sweep re-queries)", err)
	}
	if got := f.reload(inv.ID); got.Status != billing_models.InvoiceStatusValidated {
		t.Fatalf("status = %q, want validated", got.Status)
	}
}

func TestProcess_NotAuthorized_IsRejected(t *testing.T) {
	f := newFixture(t)
	inv := f.invoice(billing_models.InvoiceStatusPending)
	f.gateway.auth = Authorization{Status: NotAuthorized, Message: "[SRI-ERROR] FIRMA INVALIDA (ID 39) - La firma es invalida"}

	if err := f.process(inv.ID); err != nil {
		t.Fatalf("process: %v", err)
	}

	got := f.reload(inv.ID)
	if got.Status != billing_models.InvoiceStatusRejected {
		t.Fatalf("status = %q, want rejected", got.Status)
	}
	if got.ErrorCode == nil || *got.ErrorCode != ErrorCodeNotAuthorized {
		t.Fatalf("error_code = %v, want %s", got.ErrorCode, ErrorCodeNotAuthorized)
	}
	if got.ErrorMessage == nil || !strings.Contains(*got.ErrorMessage, "(ID 39)") {
		t.Fatalf("error_message = %v, want SRI's reason", got.ErrorMessage)
	}
	if len(f.authorized) != 0 {
		t.Fatalf("OnAuthorized ran for a rejected document")
	}

	if err := f.process(inv.ID); err != nil {
		t.Fatalf("process rejected: %v", err)
	}
	if f.gateway.signCalls != 1 || f.gateway.authCalls != 1 {
		t.Fatalf("a rejected document was processed again (sign=%d auth=%d)", f.gateway.signCalls, f.gateway.authCalls)
	}
}

func TestProcess_OnlyClaimableDocumentsStartAnAttempt(t *testing.T) {
	stale := time.Now().Add(-StuckThreshold - time.Minute)
	fresh := time.Now()

	cases := []struct {
		status    string
		updatedAt time.Time
		attempted bool
	}{
		{billing_models.InvoiceStatusPending, fresh, true},
		{billing_models.InvoiceStatusFailed, fresh, true},
		{billing_models.InvoiceStatusConnectionError, fresh, true},
		{billing_models.InvoiceStatusProcessing, stale, true},
		{billing_models.InvoiceStatusSigned, stale, true},
		{billing_models.InvoiceStatusProcessing, fresh, false},
		{billing_models.InvoiceStatusSigned, fresh, false},
		{billing_models.InvoiceStatusAuthorized, stale, false},
		{billing_models.InvoiceStatusRejected, stale, false},
	}

	for _, tc := range cases {
		t.Run(fmt.Sprintf("%s_stale=%v", tc.status, tc.updatedAt.Equal(stale)), func(t *testing.T) {
			f := newFixture(t)
			inv := f.invoice(tc.status)
			f.db.Model(&inv).UpdateColumn("updated_at", tc.updatedAt)

			if err := f.process(inv.ID); err != nil {
				t.Fatalf("process: %v", err)
			}
			if attempted := f.gateway.signCalls == 1; attempted != tc.attempted {
				t.Fatalf("attempted = %v, want %v", attempted, tc.attempted)
			}
			if got := f.reload(inv.ID).Status; !tc.attempted && got != tc.status {
				t.Fatalf("status changed to %q", got)
			}
		})
	}
}

func TestProcess_MissingDocumentIsSkipped(t *testing.T) {
	f := newFixture(t)
	if err := f.process(999); err != nil {
		t.Fatalf("process: %v", err)
	}
}

func TestProcess_ReceivedDocumentWithoutAccessKey_FailsInsteadOfPanicking(t *testing.T) {
	f := newFixture(t)
	inv := f.invoice(billing_models.InvoiceStatusValidated)

	if err := f.process(inv.ID); err != nil {
		t.Fatalf("process: %v, want nil (nothing to retry)", err)
	}
	got := f.reload(inv.ID)
	if got.Status != billing_models.InvoiceStatusFailed || got.ErrorCode == nil || *got.ErrorCode != ErrorCodeInternal {
		t.Fatalf("status=%q error_code=%v, want failed/%s", got.Status, got.ErrorCode, ErrorCodeInternal)
	}
	if f.gateway.authCalls != 0 {
		t.Fatalf("queried the SRI without an access key")
	}
}

func TestProcess_TenantWithoutSignature_Fails(t *testing.T) {
	f := newFixture(t)
	f.db.Model(&f.tenant).Update("sri_p12_path", "")
	inv := f.invoice(billing_models.InvoiceStatusPending)

	if err := f.process(inv.ID); err == nil || IsRetryable(err) {
		t.Fatalf("err = %v, want a non-retryable error", err)
	}
	got := f.reload(inv.ID)
	if got.Status != billing_models.InvoiceStatusFailed || got.ErrorCode == nil || *got.ErrorCode != ErrorCodeMissingSignature {
		t.Fatalf("status=%q error_code=%v, want failed/%s", got.Status, got.ErrorCode, ErrorCodeMissingSignature)
	}
	if f.gateway.signCalls != 0 {
		t.Fatalf("signed without a certificate")
	}
}

func TestProcess_PostAuthorizationFailureDoesNotFailTheDocument(t *testing.T) {
	f := newFixture(t)
	f.kind.OnAuthorized = func(db *gorm.DB, docs Documents, id uint, tenant tenant_models.Tenant, sriEnv string) error {
		return fmt.Errorf("gotenberg down")
	}
	inv := f.invoice(billing_models.InvoiceStatusPending)

	if err := f.process(inv.ID); err != nil {
		t.Fatalf("process: %v", err)
	}
	if got := f.reload(inv.ID); got.Status != billing_models.InvoiceStatusAuthorized {
		t.Fatalf("status = %q, want authorized", got.Status)
	}
}

func TestProcess_ClearsPreviousErrorOnceReceived(t *testing.T) {
	f := newFixture(t)
	inv := f.invoice(billing_models.InvoiceStatusConnectionError)
	f.db.Model(&inv).Updates(map[string]any{"error_code": ErrorCodeConnection, "error_message": "timeout"})

	if err := f.process(inv.ID); err != nil {
		t.Fatalf("process: %v", err)
	}
	if got := f.reload(inv.ID); got.ErrorCode != nil || got.ErrorMessage != nil {
		t.Fatalf("error_code=%v error_message=%v, want cleared", got.ErrorCode, got.ErrorMessage)
	}
}

// ─── Enqueue ─────────────────────────────────────────────────────────────────

// tenantDB is what handlers pass to Enqueue: a handle bound to the caller's tenant.
func (f *fixture) tenantDB(tenantID uint) *gorm.DB {
	return tenantdb.ForTenant(f.db, tenantID)
}

func TestEnqueue_FailedDocumentGoesBackToPendingAndIsQueued(t *testing.T) {
	f := newFixture(t)
	inv := f.invoice(billing_models.InvoiceStatusFailed)
	f.db.Model(&inv).Updates(map[string]any{"error_code": ErrorCodeReturned, "error_message": "DEVUELTA"})

	if err := f.docs.Enqueue(f.tenantDB(f.tenant.ID), f.kind, uint64(inv.ID)); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	got := f.reload(inv.ID)
	if got.Status != billing_models.InvoiceStatusPending || got.ErrorCode != nil || got.ErrorMessage != nil {
		t.Fatalf("status=%q error_code=%v error_message=%v, want pending with errors cleared", got.Status, got.ErrorCode, got.ErrorMessage)
	}
	want := published{"invoice_tasks", fmt.Sprintf(`{"invoice_id":%d}`, inv.ID)}
	if len(f.publisher.messages) != 1 || f.publisher.messages[0] != want {
		t.Fatalf("published = %v, want %v", f.publisher.messages, want)
	}
}

func TestEnqueue_ReceivedDocumentIsQueuedWithoutChangingStatus(t *testing.T) {
	f := newFixture(t)
	inv := f.invoice(billing_models.InvoiceStatusValidated)

	if err := f.docs.Enqueue(f.tenantDB(f.tenant.ID), f.kind, uint64(inv.ID)); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	if got := f.reload(inv.ID).Status; got != billing_models.InvoiceStatusValidated {
		t.Fatalf("status = %q, want validated (a received comprobante is never reset)", got)
	}
	if len(f.publisher.messages) != 1 {
		t.Fatalf("published = %v, want one task", f.publisher.messages)
	}
}

func TestEnqueue_RefusesAuthorizedDocument(t *testing.T) {
	f := newFixture(t)
	inv := f.invoice(billing_models.InvoiceStatusAuthorized)

	err := f.docs.Enqueue(f.tenantDB(f.tenant.ID), f.kind, uint64(inv.ID))
	if !errors.Is(err, ErrAlreadyAuthorized) {
		t.Fatalf("err = %v, want ErrAlreadyAuthorized", err)
	}
	if len(f.publisher.messages) != 0 || f.reload(inv.ID).Status != billing_models.InvoiceStatusAuthorized {
		t.Fatalf("authorized document was touched")
	}
}

// Ficha técnica SRI offline v2.26 §5.10 and §5.12: a NO AUTORIZADO comprobante
// must be corrected and resent with the same access key and sequential.
func TestEnqueue_NotAuthorizedDocumentIsResentWithTheSameKeyOnceCorrected(t *testing.T) {
	f := newFixture(t)
	inv := f.invoice(billing_models.InvoiceStatusPending)
	f.gateway.auth = Authorization{Status: NotAuthorized, Message: "[SRI-ERROR] FIRMA INVALIDA (ID 39) - La firma es invalida"}
	if err := f.process(inv.ID); err != nil {
		t.Fatalf("process: %v", err)
	}
	rejected := f.reload(inv.ID)

	// The tenant uploads a valid certificate and retries from the UI.
	f.gateway.auth = Authorization{Status: Authorized, AuthorizedAt: time.Now()}
	if err := f.docs.Enqueue(f.tenantDB(f.tenant.ID), f.kind, uint64(inv.ID)); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	if got := f.reload(inv.ID); got.Status != billing_models.InvoiceStatusPending || got.ErrorCode != nil {
		t.Fatalf("status=%q error_code=%v, want pending with the rejection cleared", got.Status, got.ErrorCode)
	}
	if err := f.process(inv.ID); err != nil {
		t.Fatalf("process retry: %v", err)
	}

	got := f.reload(inv.ID)
	if got.Status != billing_models.InvoiceStatusAuthorized {
		t.Fatalf("status = %q, want authorized", got.Status)
	}
	if *got.AccessKey != *rejected.AccessKey {
		t.Fatalf("access key changed from %s to %s", *rejected.AccessKey, *got.AccessKey)
	}
	if f.gateway.signCalls != 2 || f.gateway.receptionCalls != 2 {
		t.Fatalf("sign=%d reception=%d, want the corrected comprobante re-signed and resent", f.gateway.signCalls, f.gateway.receptionCalls)
	}
}

func TestEnqueue_OtherTenantsDocumentIsNotFound(t *testing.T) {
	f := newFixture(t)
	inv := f.invoice(billing_models.InvoiceStatusFailed)

	err := f.docs.Enqueue(f.tenantDB(f.tenant.ID+1), f.kind, uint64(inv.ID))
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
	if len(f.publisher.messages) != 0 || f.reload(inv.ID).Status != billing_models.InvoiceStatusFailed {
		t.Fatalf("another tenant's document was touched")
	}
}

// ─── Sweep ───────────────────────────────────────────────────────────────────

func TestSweep_RequeuesStuckAndAwaitingAuthorizationDocuments(t *testing.T) {
	f := newFixture(t)
	stuck := time.Now().Add(-StuckThreshold - time.Minute)
	received := time.Now().Add(-AuthorizationRecheckAfter - time.Minute)
	recent := time.Now()

	docs := []struct {
		status    string
		updatedAt time.Time
		requeued  bool
	}{
		{billing_models.InvoiceStatusProcessing, stuck, true},
		{billing_models.InvoiceStatusSigned, stuck, true},
		{billing_models.InvoiceStatusValidated, received, true},
		{billing_models.InvoiceStatusProcessing, recent, false},
		{billing_models.InvoiceStatusValidated, recent, false},
		{billing_models.InvoiceStatusPending, stuck, false},
		{billing_models.InvoiceStatusFailed, stuck, false},
		{billing_models.InvoiceStatusAuthorized, stuck, false},
		{billing_models.InvoiceStatusRejected, stuck, false},
	}
	var want []published
	for _, d := range docs {
		inv := f.invoice(d.status)
		f.db.Model(&inv).UpdateColumn("updated_at", d.updatedAt)
		if d.requeued {
			want = append(want, published{"invoice_tasks", fmt.Sprintf(`{"invoice_id":%d}`, inv.ID)})
		}
	}

	f.docs.Sweep(f.kind)

	if fmt.Sprint(f.publisher.messages) != fmt.Sprint(want) {
		t.Fatalf("published = %v, want %v", f.publisher.messages, want)
	}
}

// ─── Consumer ────────────────────────────────────────────────────────────────

func TestHandleTask_ProcessesTheDocumentInTheMessage(t *testing.T) {
	f := newFixture(t)
	inv := f.invoice(billing_models.InvoiceStatusPending)

	if err := f.docs.HandleTask(f.kind)([]byte(fmt.Sprintf(`{"invoice_id":%d}`, inv.ID))); err != nil {
		t.Fatalf("handle: %v", err)
	}
	if got := f.reload(inv.ID).Status; got != billing_models.InvoiceStatusAuthorized {
		t.Fatalf("status = %q, want authorized", got)
	}
}

func TestHandleTask_MalformedMessageIsNotRetried(t *testing.T) {
	f := newFixture(t)
	for _, body := range []string{`not json`, `{"other_id":1}`} {
		if err := f.docs.HandleTask(f.kind)([]byte(body)); err == nil || IsRetryable(err) {
			t.Fatalf("body %s: err = %v, want a non-retryable error", body, err)
		}
	}
}

func TestEnqueue_DraftInvoiceIsQueuedAndThenProcessed(t *testing.T) {
	f := newFixture(t)
	inv := f.invoice("draft") // invoices are created with the column default

	if err := f.docs.Enqueue(f.tenantDB(f.tenant.ID), f.kind, uint64(inv.ID)); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	if err := f.process(inv.ID); err != nil {
		t.Fatalf("process: %v", err)
	}
	if got := f.reload(inv.ID).Status; got != billing_models.InvoiceStatusAuthorized {
		t.Fatalf("status = %q, want authorized", got)
	}
}

// signedXML is the signed XML stored for an invoice, or "" if there is none.
func (f *fixture) signedXML(inv billing_models.Invoice) string {
	if inv.AccessKey == nil {
		return ""
	}
	data, err := f.files.Read(f.tenant.ID, "invoices/"+*inv.AccessKey+".xml")
	if err != nil {
		return ""
	}
	return string(data)
}
