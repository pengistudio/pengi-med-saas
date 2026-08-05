# Consumidor Final Invoices Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let a tenant create an invoice with no patient attached ("Consumidor Final") without crashing, and have that invoice's SRI XML/RIDE correctly declare the SRI-mandated Consumidor Final buyer identity instead of an empty one.

**Architecture:** `Invoice.PatientID`/`Invoice.Patient` become nullable (the `patients.patient_id` DB column is already nullable — no migration needed, this is a Go-model-only change). A new `ResolveBuyerInfo(patient *clinical_models.Patient)` helper centralizes the "no patient → SRI Consumidor Final constants" fallback and is wired into the three SRI document generators (invoice, credit note, debit note) plus the RIDE (printable PDF) builder. The frontend already has a "Consumidor Final" toggle in the invoice form; it currently sends `patient_id: 0` instead of omitting the field, which is what causes today's FK-violation crash — it's fixed to send `null`.

**Tech Stack:** Go + GORM + PostgreSQL (`apps/api`), React + TypeScript + Zod (`apps/web`).

## Global Constraints

- Never write directly to `gin.Context` — handlers return `envelope.Response` (existing pattern, unaffected by this plan).
- No hardcoded user-facing strings — reuse the existing i18n key `billing.invoice.final_consumer` (already present in both `messages_es.json` and `messages_en.json`, value "Consumidor Final" / "Final Consumer") for the frontend fallback display text introduced in Task 3. No new i18n keys are needed.
- Multi-tenant scoping (`tenantScope`) is unaffected by this plan — no query in scope here bypasses it.
- Do not add a code-migration: `docker exec pengi-db-dev psql -U postgres -d pengi_gentoo -c "\d invoices"` was checked directly — `patient_id` has **no** `NOT NULL` in the live schema already, only the `fk_invoices_patient` FK constraint (which Postgres never enforces against `NULL`). The whole bug is that Go's `uint` (not `*uint`) forces a `0` to be written instead of `NULL`.

---

### Task 1: Nullable `Invoice.PatientID` + `CreateInvoice` handler

**Files:**
- Modify: `apps/api/testutils/helpers.go`
- Modify: `apps/api/features/billing/models/invoice.go`
- Modify: `apps/api/features/billing/handlers/invoice-handler.go:49-63`
- Modify: `apps/api/features/billing/handlers/invoice_handler_test.go:65,72,160` (compile fix, existing tests)
- Modify: `apps/api/features/billing/handlers/credit_note_handler_test.go:68` (compile fix)
- Modify: `apps/api/features/billing/handlers/debit_note_handler_test.go:56` (compile fix)

**Interfaces:**
- Produces: `testutils.Ptr[T any](v T) *T` — generic pointer-of helper, used by every test in this repo that needs to populate an optional `*uint`/`*string`/etc. field from a value.
- Produces: `billing_models.Invoice.PatientID *uint`, `billing_models.Invoice.Patient *clinical_models.Patient` — consumed by Task 2's SRI generators via `ResolveBuyerInfo(invoice.Patient)`.

- [ ] **Step 1: Write the failing test**

Add to `apps/api/features/billing/handlers/invoice_handler_test.go` (same package, reuse existing imports — no new imports needed):

```go
func TestCreateInvoice_FinalConsumer_NoPatient(t *testing.T) {
	db := testutils.SetupTestDB(t,
		&tenant_models.Tenant{},
		&clinical_models.Patient{},
		&billing_models.Invoice{},
		&billing_models.InvoiceItem{},
		&billing_models.InvoiceCounter{},
		&billing_models.CatalogItem{},
	)
	logger := zap.NewNop()

	now := time.Now().UnixNano() % 1000000
	tenant := &tenant_models.Tenant{
		Name:         "Final Consumer Tenant",
		Slug:         fmt.Sprintf("inv-fc-%d", now),
		DisplayToken: fmt.Sprintf("tok-inv-fc-%d", now),
	}
	if err := db.Create(tenant).Error; err != nil {
		t.Fatalf("failed to create test tenant: %v", err)
	}

	product := &billing_models.CatalogItem{
		TenantID:  tenant.ID,
		Name:      "Consulta",
		SKU:       fmt.Sprintf("CONS-%d", now),
		UnitPrice: 50.0,
		Tax:       0.12,
	}
	if err := db.Create(product).Error; err != nil {
		t.Fatalf("failed to create test catalog item: %v", err)
	}

	handler := NewInvoiceHandler(db, logger)

	payload := billing_dto.CreateInvoiceDTO{
		// PatientID intentionally omitted — "Consumidor Final" invoice
		PaymentMethod:     "01",
		Term:              0,
		TimeUnit:          "dias",
		EstablishmentCode: "001",
		EmissionPointCode: "001",
		Items: []billing_dto.CreateInvoiceItem{
			{ProductID: product.ID, Quantity: 1, Discount: 0},
		},
	}
	body, _ := json.Marshal(payload)

	c, _ := testutils.NewGinContext(tenant.ID, 1)
	c.Request = httptest.NewRequest("POST", "/invoices", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	response := handler.CreateInvoice(c)

	if response.Code != 200 {
		t.Fatalf("expected status 200, got %d (data: %+v)", response.Code, response.Data)
	}

	var stored billing_models.Invoice
	if err := db.Where("tenant_id = ?", tenant.ID).First(&stored).Error; err != nil {
		t.Fatalf("failed to load created invoice: %v", err)
	}
	if stored.PatientID != nil {
		t.Errorf("expected PatientID to be nil for a final-consumer invoice, got %v", *stored.PatientID)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd apps/api && go test ./features/billing/handlers/... -run TestCreateInvoice_FinalConsumer_NoPatient -v`
Expected: FAIL — `response.Code` is 500 (insert violates `fk_invoices_patient`, mirrors the bug report: today `dto.PatientID == nil` still leaves `invoice.PatientID` at Go's zero value `0`, which is what gets inserted).

- [ ] **Step 3: Add the `Ptr` test helper**

In `apps/api/testutils/helpers.go`, add at the end of the file:

```go
// Ptr returns a pointer to the given value. Useful for populating optional
// (*T) struct fields in tests without an intermediate variable.
func Ptr[T any](v T) *T {
	return &v
}
```

- [ ] **Step 4: Make `PatientID`/`Patient` nullable on the model**

In `apps/api/features/billing/models/invoice.go`, change:

```go
	PatientID         uint                    `json:"patient_id"` // FK a Patient
	Patient           clinical_models.Patient `gorm:"foreignKey:PatientID" json:"patient"`
```

to:

```go
	PatientID         *uint                    `json:"patient_id"` // FK a Patient — nil = Consumidor Final (sin paciente)
	Patient           *clinical_models.Patient `gorm:"foreignKey:PatientID" json:"patient"`
```

- [ ] **Step 5: Simplify the `CreateInvoice` assignment**

In `apps/api/features/billing/handlers/invoice-handler.go:61-63`, replace:

```go
	if dto.PatientID != nil {
		invoice.PatientID = *dto.PatientID
	}
```

with:

```go
	invoice.PatientID = dto.PatientID
```

(`dto.PatientID` is already `*uint` — see `apps/api/features/billing/dto/invoice-dto.go:10` — so this is now a direct assignment, no dereference.)

- [ ] **Step 6: Fix existing tests to compile against the pointer field**

In `apps/api/features/billing/handlers/invoice_handler_test.go`:
- Line 65: `PatientID:  pat1.ID,` → `PatientID:  testutils.Ptr(pat1.ID),`
- Line 72: `PatientID:  pat2.ID,` → `PatientID:  testutils.Ptr(pat2.ID),`
- Line 160: `PatientID:  patient.ID,` → `PatientID:  testutils.Ptr(patient.ID),`

In `apps/api/features/billing/handlers/credit_note_handler_test.go:68`:
- `PatientID:         patient.ID,` → `PatientID:         testutils.Ptr(patient.ID),`

In `apps/api/features/billing/handlers/debit_note_handler_test.go:56`:
- `PatientID:         patient.ID,` → `PatientID:         testutils.Ptr(patient.ID),`

- [ ] **Step 7: Run tests to verify everything passes**

Run: `cd apps/api && go build ./... && go test ./features/billing/... ./testutils/... -v`
Expected: PASS — all billing handler tests green, including the new `TestCreateInvoice_FinalConsumer_NoPatient`.

- [ ] **Step 8: Commit**

```bash
git add apps/api/testutils/helpers.go \
  apps/api/features/billing/models/invoice.go \
  apps/api/features/billing/handlers/invoice-handler.go \
  apps/api/features/billing/handlers/invoice_handler_test.go \
  apps/api/features/billing/handlers/credit_note_handler_test.go \
  apps/api/features/billing/handlers/debit_note_handler_test.go
git commit -m "fix(billing): permitir facturas sin paciente (Consumidor Final)"
```

---

### Task 2: SRI "Consumidor Final" buyer resolution

**Files:**
- Modify: `apps/api/features/billing/sri/services/buyer-identification.go`
- Create: `apps/api/features/billing/sri/services/buyer-identification_test.go`
- Modify: `apps/api/features/billing/sri/services/generate-invoice.go:260-277`
- Modify: `apps/api/features/billing/sri/services/generate-credit-note.go:144-160`
- Modify: `apps/api/features/billing/sri/services/generate-debit-note.go:85-103`
- Modify: `apps/api/features/billing/sri/services/generate-invoice-ride.go:123-157`
- Modify: `apps/api/features/billing/sri/services/generate_test.go` (4 occurrences of `Patient: patient`)
- Modify: `apps/api/features/billing/sri/services/generate-invoice-ride_test.go` (1 occurrence of `Patient: patient`)

**Interfaces:**
- Consumes: `billing_models.Invoice.Patient *clinical_models.Patient` (Task 1).
- Produces: `services.ResolveBuyerInfo(patient *clinical_models.Patient) (identification, identificationType, socialReason string)` and constants `services.FinalConsumerIdentification`, `services.FinalConsumerIdentificationType`, `services.FinalConsumerSocialReason` — all in package `services` (`apps/api/features/billing/sri/services`), used directly (same package, no import needed) by every generator in this task.

- [ ] **Step 1: Write the failing test**

Create `apps/api/features/billing/sri/services/buyer-identification_test.go`:

```go
package services

import (
	"testing"

	clinical_models "pengi-med-saas/features/clinical/models"
)

func TestResolveBuyerInfo_NilPatient_ReturnsFinalConsumer(t *testing.T) {
	identification, identificationType, socialReason := ResolveBuyerInfo(nil)

	if identification != FinalConsumerIdentification {
		t.Errorf("expected identification %q, got %q", FinalConsumerIdentification, identification)
	}
	if identificationType != FinalConsumerIdentificationType {
		t.Errorf("expected identificationType %q, got %q", FinalConsumerIdentificationType, identificationType)
	}
	if socialReason != FinalConsumerSocialReason {
		t.Errorf("expected socialReason %q, got %q", FinalConsumerSocialReason, socialReason)
	}
}

func TestResolveBuyerInfo_WithPatient_ReturnsPatientData(t *testing.T) {
	patient := &clinical_models.Patient{
		FirstName: "Juan",
		LastName:  "Perez",
		Document:  "1710000000",
	}

	identification, identificationType, socialReason := ResolveBuyerInfo(patient)

	if identification != "1710000000" {
		t.Errorf("expected identification %q, got %q", "1710000000", identification)
	}
	if identificationType != "05" {
		t.Errorf("expected identificationType %q, got %q", "05", identificationType)
	}
	if socialReason != "Juan Perez" {
		t.Errorf("expected socialReason %q, got %q", "Juan Perez", socialReason)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd apps/api && go test ./features/billing/sri/services/... -run TestResolveBuyerInfo -v`
Expected: FAIL with a compile error — `ResolveBuyerInfo`/`FinalConsumerIdentification`/etc. undefined.

- [ ] **Step 3: Implement `ResolveBuyerInfo`**

Replace the full contents of `apps/api/features/billing/sri/services/buyer-identification.go` with:

```go
package services

import (
	"strings"

	clinical_models "pengi-med-saas/features/clinical/models"
)

// SRI-mandated identification for a document (factura/nota) issued without a
// buyer on file — i.e. a "Consumidor Final" sale.
const (
	FinalConsumerIdentification     = "9999999999999"
	FinalConsumerIdentificationType = "07"
	FinalConsumerSocialReason       = "CONSUMIDOR FINAL"
)

// ResolveBuyerIdentificationType returns the SRI buyer identification type code:
//
//	04 RUC (13 digits, ends in 001)
//	05 Cédula (10 digits)
//	06 Pasaporte / otro (anything else non-empty)
//	07 Consumidor Final (no document)
func ResolveBuyerIdentificationType(document string) string {
	if document == "" {
		return FinalConsumerIdentificationType
	}

	digitsOnly := true
	for _, r := range document {
		if r < '0' || r > '9' {
			digitsOnly = false
			break
		}
	}

	switch {
	case digitsOnly && len(document) == 13 && strings.HasSuffix(document, "001"):
		return "04"
	case digitsOnly && len(document) == 10:
		return "05"
	default:
		return "06"
	}
}

// ResolveBuyerInfo returns the SRI buyer identification, identification type and
// social reason for a document. When patient is nil (no patient attached — a
// "Consumidor Final" sale), it returns the SRI-mandated Consumidor Final triple
// instead of an empty/zero buyer.
func ResolveBuyerInfo(patient *clinical_models.Patient) (identification, identificationType, socialReason string) {
	if patient == nil {
		return FinalConsumerIdentification, FinalConsumerIdentificationType, FinalConsumerSocialReason
	}
	return patient.Document, ResolveBuyerIdentificationType(patient.Document), patient.FirstName + " " + patient.LastName
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd apps/api && go test ./features/billing/sri/services/... -run TestResolveBuyerInfo -v`
Expected: PASS.

- [ ] **Step 5: Wire `ResolveBuyerInfo` into the invoice XML generator**

In `apps/api/features/billing/sri/services/generate-invoice.go:260-277`, replace:

```go
	return invoiceSRI.InvoiceInfo{
		IssueDate:               invoice.IssueDate.Format("02/01/2006"),
		BuyerIdentification:     invoice.Patient.Document,
		BuyerAddress:            "S/N", // As Pengi's Patient model does not have an address yet
		BuyerIdentificationType: ResolveBuyerIdentificationType(invoice.Patient.Document),
		BuyerSocialReason:       invoice.Patient.FirstName + " " + invoice.Patient.LastName,
```

with:

```go
	buyerIdentification, buyerIdentificationType, buyerSocialReason := ResolveBuyerInfo(invoice.Patient)

	return invoiceSRI.InvoiceInfo{
		IssueDate:               invoice.IssueDate.Format("02/01/2006"),
		BuyerIdentification:     buyerIdentification,
		BuyerAddress:            "S/N", // As Pengi's Patient model does not have an address yet
		BuyerIdentificationType: buyerIdentificationType,
		BuyerSocialReason:       buyerSocialReason,
```

(leave the rest of the struct literal — `EstablishmentAddress`, `SpecialContributor`, etc. — untouched.)

- [ ] **Step 6: Wire `ResolveBuyerInfo` into the credit note XML generator**

In `apps/api/features/billing/sri/services/generate-credit-note.go:144-160`, replace:

```go
	modifiedDocNumber := fmt.Sprintf("%s-%s-%s", creditNote.Invoice.EstablishmentCode, creditNote.Invoice.EmissionPointCode, creditNote.Invoice.Sequential)

	currency := "DOLAR"
	info := invoiceSRI.CreditNoteInfo{
		IssueDate:               creditNote.IssueDate.Format("02/01/2006"),
		EstablishmentAddress:    establishmentAddress,
		SpecialContributor:      tenantObj.SpecialContributorNumber,
		AccountingObliged:       accountingObliged,
		BuyerIdentificationType: ResolveBuyerIdentificationType(creditNote.Invoice.Patient.Document),
		BuyerSocialReason:       creditNote.Invoice.Patient.FirstName + " " + creditNote.Invoice.Patient.LastName,
		BuyerIdentification:     creditNote.Invoice.Patient.Document,
```

with:

```go
	buyerIdentification, buyerIdentificationType, buyerSocialReason := ResolveBuyerInfo(creditNote.Invoice.Patient)

	modifiedDocNumber := fmt.Sprintf("%s-%s-%s", creditNote.Invoice.EstablishmentCode, creditNote.Invoice.EmissionPointCode, creditNote.Invoice.Sequential)

	currency := "DOLAR"
	info := invoiceSRI.CreditNoteInfo{
		IssueDate:               creditNote.IssueDate.Format("02/01/2006"),
		EstablishmentAddress:    establishmentAddress,
		SpecialContributor:      tenantObj.SpecialContributorNumber,
		AccountingObliged:       accountingObliged,
		BuyerIdentificationType: buyerIdentificationType,
		BuyerSocialReason:       buyerSocialReason,
		BuyerIdentification:     buyerIdentification,
```

(leave `ModifiedDocCode` onward untouched.)

- [ ] **Step 7: Wire `ResolveBuyerInfo` into the debit note XML generator**

In `apps/api/features/billing/sri/services/generate-debit-note.go:85-103`, replace:

```go
	modifiedDocNumber := fmt.Sprintf("%s-%s-%s", debitNote.Invoice.EstablishmentCode, debitNote.Invoice.EmissionPointCode, debitNote.Invoice.Sequential)
	currency := "DOLAR"

	return invoiceSRI.DebitNoteInfo{
		IssueDate:               debitNote.IssueDate.Format("02/01/2006"),
		EstablishmentAddress:    establishmentAddress,
		SpecialContributor:      tenantObj.SpecialContributorNumber,
		AccountingObliged:       accountingObliged,
		BuyerIdentificationType: ResolveBuyerIdentificationType(debitNote.Invoice.Patient.Document),
		BuyerSocialReason:       debitNote.Invoice.Patient.FirstName + " " + debitNote.Invoice.Patient.LastName,
		BuyerIdentification:     debitNote.Invoice.Patient.Document,
```

with:

```go
	buyerIdentification, buyerIdentificationType, buyerSocialReason := ResolveBuyerInfo(debitNote.Invoice.Patient)

	modifiedDocNumber := fmt.Sprintf("%s-%s-%s", debitNote.Invoice.EstablishmentCode, debitNote.Invoice.EmissionPointCode, debitNote.Invoice.Sequential)
	currency := "DOLAR"

	return invoiceSRI.DebitNoteInfo{
		IssueDate:               debitNote.IssueDate.Format("02/01/2006"),
		EstablishmentAddress:    establishmentAddress,
		SpecialContributor:      tenantObj.SpecialContributorNumber,
		AccountingObliged:       accountingObliged,
		BuyerIdentificationType: buyerIdentificationType,
		BuyerSocialReason:       buyerSocialReason,
		BuyerIdentification:     buyerIdentification,
```

(leave `ModifiedDocCode` onward untouched.)

- [ ] **Step 8: Wire `ResolveBuyerInfo` into the RIDE (printable PDF) builder**

In `apps/api/features/billing/sri/services/generate-invoice-ride.go`, replace lines 123-126:

```go
	buyerName := ""
	if invoice.Patient.ID != 0 {
		buyerName = invoice.Patient.FirstName + " " + invoice.Patient.LastName
	}
```

with:

```go
	buyerIdentification, _, buyerName := ResolveBuyerInfo(invoice.Patient)
```

Then in the `InvoiceRideTemplateData{...}` struct literal, replace line 157:

```go
		BuyerIdentification:  invoice.Patient.Document,
```

with:

```go
		BuyerIdentification:  buyerIdentification,
```

- [ ] **Step 9: Fix existing tests to compile against the pointer field**

In `apps/api/features/billing/sri/services/generate_test.go`, change all 4 occurrences of `Patient:      patient,` / `Patient:           patient,` to `Patient: &patient,` (keep each line's existing alignment/whitespace — just prefix `patient` with `&`).

In `apps/api/features/billing/sri/services/generate-invoice-ride_test.go`, change `Patient:           patient,` to `Patient:           &patient,`. The following line `invoice.Patient.ID = 1` needs no change — field access through a pointer works the same way in Go.

- [ ] **Step 10: Run all SRI service tests**

Run: `cd apps/api && go build ./... && go test ./features/billing/sri/services/... -v`
Expected: PASS — all existing tests (`TestGenerateInvoice_*`, `TestBuildInvoiceRideData_AndRenderTemplate`, `TestResolveBuyerInfo_*`) green.

- [ ] **Step 11: Run the full backend test suite**

Run: `cd apps/api && go build ./... && go vet ./... && go test ./...`
Expected: PASS.

- [ ] **Step 12: Commit**

```bash
git add apps/api/features/billing/sri/services/buyer-identification.go \
  apps/api/features/billing/sri/services/buyer-identification_test.go \
  apps/api/features/billing/sri/services/generate-invoice.go \
  apps/api/features/billing/sri/services/generate-credit-note.go \
  apps/api/features/billing/sri/services/generate-debit-note.go \
  apps/api/features/billing/sri/services/generate-invoice-ride.go \
  apps/api/features/billing/sri/services/generate_test.go \
  apps/api/features/billing/sri/services/generate-invoice-ride_test.go
git commit -m "feat(billing): declarar Consumidor Final en XML/RIDE cuando la factura no tiene paciente"
```

---

### Task 3: Frontend — stop sending `patient_id: 0`, display "Consumidor Final"

**Files:**
- Modify: `apps/web/src/api/billing-service.ts:36-55,129-141`
- Modify: `apps/web/src/sections/forms/billing/invoice-form.tsx:93-101`
- Modify: `apps/web/src/sections/columns/billing/invoice-columns.tsx:232-237`
- Modify: `apps/web/src/pages/home/home.tsx:292-294`

**Interfaces:**
- Consumes: backend now accepts a missing/`null` `patient_id` on `POST /billing/invoices` and returns `patient_id: null` / `patient: null` on invoices without a patient (Tasks 1-2).

- [ ] **Step 1: Widen the `Invoice`/`CreateInvoicePayload` types**

In `apps/web/src/api/billing-service.ts`, in the `Invoice` interface (lines 36-55), change:

```ts
	patient_id: number;
	patient?: Patient;
```

to:

```ts
	patient_id: number | null;
	patient?: Patient | null;
```

In the `CreateInvoicePayload` type (lines 129-141), change:

```ts
export type CreateInvoicePayload = {
	patient_id: number;
```

to:

```ts
export type CreateInvoicePayload = {
	patient_id?: number | null;
```

- [ ] **Step 2: Stop sending `patient_id: 0`**

In `apps/web/src/sections/forms/billing/invoice-form.tsx:95`, change:

```ts
			patient_id: values.patient_id ? parseInt(values.patient_id, 10) : 0,
```

to:

```ts
			patient_id: values.patient_id ? parseInt(values.patient_id, 10) : null,
```

- [ ] **Step 3: Show "Consumidor Final" instead of a generic fallback**

In `apps/web/src/sections/columns/billing/invoice-columns.tsx:232-237`, change:

```tsx
								{invoice.patient ? (
									`${invoice.patient.first_name} ${invoice.patient.last_name}`
								) : (
									<Text uuid="common.unknown" />
								)}
```

to:

```tsx
								{invoice.patient ? (
									`${invoice.patient.first_name} ${invoice.patient.last_name}`
								) : (
									<Text uuid="billing.invoice.final_consumer" />
								)}
```

In `apps/web/src/pages/home/home.tsx:292-294` (inside `RecentInvoicesCard`, which already has `const { textGet } = useText();` at line 268), change:

```ts
							const patientName = invoice.patient
								? `${invoice.patient.first_name} ${invoice.patient.last_name}`
								: `#${invoice.sequential}`;
```

to:

```ts
							const patientName = invoice.patient
								? `${invoice.patient.first_name} ${invoice.patient.last_name}`
								: textGet("billing.invoice.final_consumer");
```

- [ ] **Step 4: Typecheck and lint**

Run: `cd apps/web && pnpm run build && pnpm run lint`
Expected: both pass with no new errors.

- [ ] **Step 5: Commit**

```bash
git add apps/web/src/api/billing-service.ts \
  apps/web/src/sections/forms/billing/invoice-form.tsx \
  apps/web/src/sections/columns/billing/invoice-columns.tsx \
  apps/web/src/pages/home/home.tsx
git commit -m "fix(billing): no enviar patient_id:0 y mostrar Consumidor Final en la UI"
```

---

## Manual End-to-End Verification (after all 3 tasks)

1. `just dev` (or ensure `pengi-api`/`pengi-web` containers are running with the new code).
2. In the web app, go to Billing → New Invoice, toggle "Consumidor Final" **on**, add at least one item, submit.
3. Confirm: no 500 error, invoice is created, redirected to `/billing`.
4. In the invoice list, confirm the row shows "Consumidor Final" (not blank/"Desconocido") where the patient column would be.
5. Trigger SRI processing for that invoice (`POST /billing/invoices/:id/sri/process` or the equivalent UI action) and inspect the generated XML/RIDE: `identificacionComprador` should be `9999999999999`, `tipoIdentificacionComprador` should be `07`, `razonSocialComprador` should be `CONSUMIDOR FINAL`.
6. Repeat steps 2-3 with a real patient selected (toggle off) to confirm the non-final-consumer path is unaffected — patient name should appear normally everywhere, and the XML should carry the patient's real document/name.

## Self-Review

- **Spec coverage:** frontend `patient_id: 0` bug → Task 3 Step 2. Model/FK crash → Task 1. SRI XML blank/wrong buyer for Consumidor Final → Task 2 Steps 5-8. RIDE PDF blank buyer → Task 2 Step 8. Existing tests broken by the pointer change → Task 1 Step 6, Task 2 Step 9. All three billing document types (invoice, credit note, debit note) covered → Task 2 Steps 5-7.
- **Placeholder scan:** none — every step has literal, complete code.
- **Type consistency:** `ResolveBuyerInfo(patient *clinical_models.Patient) (identification, identificationType, socialReason string)` — the same three-return order (`identification, identificationType, socialReason`) is used consistently at every call site in Task 2 (Steps 5, 6, 7 all destructure as `buyerIdentification, buyerIdentificationType, buyerSocialReason`; Step 8 destructures as `buyerIdentification, _, buyerName` since RIDE doesn't use the type). `testutils.Ptr[T any](v T) *T` matches its 3 call sites in Task 1 Step 6. `Invoice.PatientID *uint` / `Invoice.Patient *clinical_models.Patient` match `dto.PatientID *uint` (unchanged) and every preload/read site already found to only ever read through `.Patient` (never write it directly outside the handler touched in Task 1).
