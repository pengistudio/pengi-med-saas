package billing_handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"pengi-med-saas/core/envelope"
	billing_dto "pengi-med-saas/features/billing/dto"
	billing_models "pengi-med-saas/features/billing/models"
	clinical_models "pengi-med-saas/features/clinical/models"
	tenant_models "pengi-med-saas/features/tenants/models"
	"pengi-med-saas/testutils"
)

type customerFixture struct {
	t       *testing.T
	db      *gorm.DB
	handler *CustomerHandler
	tenants [2]uint
}

func newCustomerFixture(t *testing.T) *customerFixture {
	t.Helper()
	db := testutils.SetupTestDB(t,
		&tenant_models.Tenant{}, &clinical_models.Patient{}, &billing_models.Customer{},
		&billing_models.Invoice{}, &billing_models.InvoiceItem{}, &billing_models.InvoiceCounter{},
		&billing_models.CatalogItem{},
	)
	f := &customerFixture{t: t, db: db, handler: NewCustomerHandler(db, zap.NewNop())}
	now := time.Now().UnixNano()
	for i := range f.tenants {
		tenant := tenant_models.Tenant{Name: "Customers", Slug: fmt.Sprintf("cust-%d-%d", i, now), DisplayToken: fmt.Sprintf("tok-cust-%d-%d", i, now)}
		if err := db.Create(&tenant).Error; err != nil {
			t.Fatalf("create tenant: %v", err)
		}
		f.tenants[i] = tenant.ID
	}
	return f
}

// call runs action for tenant (index) with an optional JSON body and :id.
func (f *customerFixture) call(tenant int, action envelope.Action, id uint, body any) envelope.Response {
	f.t.Helper()
	c, _ := testutils.NewGinContext(f.tenants[tenant], 1)
	var raw []byte
	if body != nil {
		raw, _ = json.Marshal(body)
	}
	c.Request = httptest.NewRequest("POST", "/billing/customers", bytes.NewReader(raw))
	c.Request.Header.Set("Content-Type", "application/json")
	if id != 0 {
		c.Params = gin.Params{{Key: "id", Value: fmt.Sprint(id)}}
	}
	return action(c)
}

func (f *customerFixture) list(tenant int, query string) envelope.PagedData {
	f.t.Helper()
	c, _ := testutils.NewGinContext(f.tenants[tenant], 1)
	c.Request = httptest.NewRequest("GET", "/billing/customers?"+query, nil)
	resp := f.handler.GetAllCustomers(c)
	if resp.Code != 200 {
		f.t.Fatalf("list: code=%d message=%q", resp.Code, resp.Message)
	}
	return resp.Data.(envelope.PagedData)
}

func (f *customerFixture) create(tenant int, dto billing_dto.CreateCustomerDTO) (envelope.Response, *billing_models.Customer) {
	f.t.Helper()
	resp := f.call(tenant, f.handler.CreateCustomer, 0, dto)
	customer, _ := resp.Data.(*billing_models.Customer)
	return resp, customer
}

func wantError(t *testing.T, resp envelope.Response, code int, message string) {
	t.Helper()
	if resp.Code != code || resp.Message != message {
		t.Fatalf("code=%d message=%q, want %d %q", resp.Code, resp.Message, code, message)
	}
}

func TestCustomers_CRUD(t *testing.T) {
	f := newCustomerFixture(t)

	resp, created := f.create(0, billing_dto.CreateCustomerDTO{Identification: " 1790012345001 ", Name: " Aseguradora del Sur S.A. ",
		Email: "facturas@aseguradora.ec", Address: "Av. Amazonas"})
	if resp.Code != 200 || created == nil {
		t.Fatalf("create: code=%d message=%q", resp.Code, resp.Message)
	}
	if created.Identification != "1790012345001" || created.Name != "Aseguradora del Sur S.A." || created.TenantID != f.tenants[0] {
		t.Fatalf("created = %+v", created)
	}
	f.create(0, billing_dto.CreateCustomerDTO{Identification: "1710034065", Name: "María Pérez"})
	f.create(1, billing_dto.CreateCustomerDTO{Identification: "AB123456", Name: "Other tenant"})

	if list := f.list(0, ""); list.Total != 2 {
		t.Fatalf("list total = %d, want 2", list.Total)
	}
	for query, want := range map[string]string{"search=aseguradora": "1790012345001", "search=171003": "1710034065"} {
		found := f.list(0, query).Items.([]billing_models.Customer)
		if len(found) != 1 || found[0].Identification != want {
			t.Fatalf("%s → %+v, want only %s", query, found, want)
		}
	}

	name := "Aseguradora del Norte"
	upd := f.call(0, f.handler.UpdateCustomer, created.ID, billing_dto.UpdateCustomerDTO{Name: &name})
	if upd.Code != 200 || upd.Data.(*billing_models.Customer).Name != name || upd.Data.(*billing_models.Customer).Email != "facturas@aseguradora.ec" {
		t.Fatalf("update: code=%d data=%+v", upd.Code, upd.Data)
	}
	if got := f.call(0, f.handler.GetCustomerByID, created.ID, nil); got.Code != 200 || got.Data.(*billing_models.Customer).Name != name {
		t.Fatalf("get: code=%d", got.Code)
	}

	if del := f.call(0, f.handler.DeleteCustomer, created.ID, nil); del.Code != 200 {
		t.Fatalf("delete: code=%d", del.Code)
	}
	wantError(t, f.call(0, f.handler.GetCustomerByID, created.ID, nil), 404, "billing.customer.error.not_found")

	// A deleted customer's identification can be registered again.
	if again, _ := f.create(0, billing_dto.CreateCustomerDTO{Identification: "1790012345001", Name: "Aseguradora"}); again.Code != 200 {
		t.Fatalf("re-create after delete: code=%d message=%q", again.Code, again.Message)
	}
}

func TestCustomers_Validation(t *testing.T) {
	f := newCustomerFixture(t)
	_, first := f.create(0, billing_dto.CreateCustomerDTO{Identification: "1710034065", Name: "Ana"})
	_, second := f.create(0, billing_dto.CreateCustomerDTO{Identification: "AB123456", Name: "Bob"})

	dup, _ := f.create(0, billing_dto.CreateCustomerDTO{Identification: "1710034065", Name: "Otra"})
	wantError(t, dup, 409, "billing.customer.error.duplicate_identification")
	// The same identification in another tenant is fine.
	if other, _ := f.create(1, billing_dto.CreateCustomerDTO{Identification: "1710034065", Name: "Ana"}); other.Code != 200 {
		t.Fatalf("other tenant: code=%d message=%q", other.Code, other.Message)
	}
	// Updating onto another customer's identification is a duplicate too.
	taken := "1710034065"
	wantError(t, f.call(0, f.handler.UpdateCustomer, second.ID, billing_dto.UpdateCustomerDTO{Identification: &taken}), 409, "billing.customer.error.duplicate_identification")
	// Keeping its own identification is not.
	same := "Ana María"
	if ok := f.call(0, f.handler.UpdateCustomer, first.ID, billing_dto.UpdateCustomerDTO{Identification: &taken, Name: &same}); ok.Code != 200 {
		t.Fatalf("update own identification: code=%d message=%q", ok.Code, ok.Message)
	}

	for _, id := range []string{"1712345678", "9999999999999", "1790012345002", "AB 123"} {
		resp, _ := f.create(0, billing_dto.CreateCustomerDTO{Identification: id, Name: "X"})
		wantError(t, resp, 400, "billing.customer.error.invalid_identification")
	}
	for _, dto := range []billing_dto.CreateCustomerDTO{
		{Identification: "AB123456"},
		{Identification: "AB123457", Name: "   "},
		{Identification: "AB123458", Name: "X", Email: "not-an-email"},
	} {
		resp, _ := f.create(0, dto)
		wantError(t, resp, 400, "billing.customer.error.invalid_payload")
	}
	blank := ""
	wantError(t, f.call(0, f.handler.UpdateCustomer, first.ID, billing_dto.UpdateCustomerDTO{Name: &blank}), 400, "billing.customer.error.invalid_payload")
}

func TestCustomers_OtherTenantsAreNotFound(t *testing.T) {
	f := newCustomerFixture(t)
	_, foreign := f.create(1, billing_dto.CreateCustomerDTO{Identification: "1710034065", Name: "Ana"})
	name := "Hijacked"

	wantError(t, f.call(0, f.handler.GetCustomerByID, foreign.ID, nil), 404, "billing.customer.error.not_found")
	wantError(t, f.call(0, f.handler.UpdateCustomer, foreign.ID, billing_dto.UpdateCustomerDTO{Name: &name}), 404, "billing.customer.error.not_found")
	wantError(t, f.call(0, f.handler.DeleteCustomer, foreign.ID, nil), 404, "billing.customer.error.not_found")
	if list := f.list(0, ""); list.Total != 0 {
		t.Fatalf("tenant 0 sees %d customers", list.Total)
	}
}

// createInvoice posts an invoice of one product worth price (no tax) for
// tenant 0.
func (f *customerFixture) createInvoice(price float64, patientID, customerID *uint) envelope.Response {
	f.t.Helper()
	product := billing_models.CatalogItem{TenantID: f.tenants[0], Name: "Consulta", SKU: fmt.Sprintf("C-%d", time.Now().UnixNano()), UnitPrice: price}
	if err := f.db.Create(&product).Error; err != nil {
		f.t.Fatalf("create product: %v", err)
	}
	return f.call(0, NewInvoiceHandler(f.db, zap.NewNop(), nil).CreateInvoice, 0, billing_dto.CreateInvoiceDTO{
		PatientID: patientID, CustomerID: customerID,
		PaymentMethod: "01", TimeUnit: "dias", EstablishmentCode: "001", EmissionPointCode: "001",
		Items: []billing_dto.CreateInvoiceItem{{ProductID: product.ID, Quantity: 1}},
	})
}

func TestCreateInvoice_WithCustomer(t *testing.T) {
	f := newCustomerFixture(t)
	patient := clinical_models.Patient{TenantID: f.tenants[0], FirstName: "Ana", LastName: "Lopez", Document: "1710034065"}
	noDocument := clinical_models.Patient{TenantID: f.tenants[0], FirstName: "Sin", LastName: "Cédula"}
	for _, p := range []*clinical_models.Patient{&patient, &noDocument} {
		if err := f.db.Create(p).Error; err != nil {
			t.Fatalf("create patient: %v", err)
		}
	}
	_, customer := f.create(0, billing_dto.CreateCustomerDTO{Identification: "1790012345001", Name: "ASEGURADORA", Email: "a@aseg.ec"})
	_, foreign := f.create(1, billing_dto.CreateCustomerDTO{Identification: "1790012345001", Name: "OTRA"})

	// Customer + patient: buyer is the customer, patient kept; over 50 is fine.
	resp := f.createInvoice(120, &patient.ID, &customer.ID)
	if resp.Code != 200 {
		t.Fatalf("create: code=%d message=%q", resp.Code, resp.Message)
	}
	inv := resp.Data.(*billing_models.Invoice)
	if *inv.CustomerID != customer.ID || *inv.PatientID != patient.ID ||
		inv.BuyerIdentification != "1790012345001" || inv.BuyerIdentificationType != "04" || inv.BuyerName != "ASEGURADORA" || inv.BuyerEmail != "a@aseg.ec" {
		t.Fatalf("invoice = %+v", inv)
	}

	// Patient only: as before, the patient is the buyer.
	resp = f.createInvoice(120, &patient.ID, nil)
	if resp.Code != 200 || resp.Data.(*billing_models.Invoice).BuyerIdentification != "1710034065" || resp.Data.(*billing_models.Invoice).BuyerName != "Ana Lopez" {
		t.Fatalf("patient invoice: code=%d data=%+v", resp.Code, resp.Data)
	}

	// The Consumidor Final cap applies to the resolved buyer: a customer lifts
	// it for a patient without document; that patient alone does not.
	if resp := f.createInvoice(120, &noDocument.ID, &customer.ID); resp.Code != 200 {
		t.Fatalf("customer for undocumented patient: code=%d message=%q", resp.Code, resp.Message)
	}
	wantError(t, f.createInvoice(120, &noDocument.ID, nil), 400, "billing.invoice.error.final_consumer_limit")
	resp = f.createInvoice(30, nil, nil)
	if resp.Code != 200 || resp.Data.(*billing_models.Invoice).BuyerIdentificationType != "07" {
		t.Fatalf("Consumidor Final invoice: code=%d", resp.Code)
	}

	// Another tenant's customer does not exist here.
	wantError(t, f.createInvoice(10, nil, &foreign.ID), 404, "billing.invoice.error.customer_not_found")
	foreignPatient := clinical_models.Patient{TenantID: f.tenants[1], FirstName: "Otro", LastName: "Tenant", Document: "1710034065"}
	if err := f.db.Create(&foreignPatient).Error; err != nil {
		t.Fatalf("create patient: %v", err)
	}
	wantError(t, f.createInvoice(10, &foreignPatient.ID, nil), 404, "billing.invoice.error.patient_not_found")

	// The list carries the customer and the snapshot.
	c, _ := testutils.NewGinContext(f.tenants[0], 1)
	c.Request = httptest.NewRequest("GET", "/billing/invoices", nil)
	list := NewInvoiceHandler(f.db, zap.NewNop(), nil).GetAllInvoices(c)
	raw, _ := json.Marshal(list.Data.(envelope.PagedData).Items)
	var rows []map[string]any
	_ = json.Unmarshal(raw, &rows)
	withCustomer := 0
	for _, row := range rows {
		if _, ok := row["buyer_name"]; !ok {
			t.Fatalf("list row lacks buyer_name: %v", row)
		}
		if cust, ok := row["customer"].(map[string]any); ok && cust != nil {
			if cust["name"] != "ASEGURADORA" || cust["identification"] != "1790012345001" {
				t.Fatalf("list customer = %v", cust)
			}
			withCustomer++
		}
	}
	if withCustomer != 2 {
		t.Fatalf("%d list rows carry the customer, want 2", withCustomer)
	}
}
