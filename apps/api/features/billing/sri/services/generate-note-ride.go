package services

import (
	"encoding/base64"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"sort"
	"time"

	"pengi-med-saas/core/pdfrender"
	billing_models "pengi-med-saas/features/billing/models"
	billing_templates "pengi-med-saas/features/billing/templates"
	tenant "pengi-med-saas/features/tenants/models"
)

// paymentMethodLabels are the SRI formas de pago (ficha técnica, tabla 24).
var paymentMethodLabels = map[string]string{
	"01": "SIN UTILIZACIÓN DEL SISTEMA FINANCIERO",
	"15": "COMPENSACIÓN DE DEUDAS",
	"16": "TARJETA DE DÉBITO",
	"17": "DINERO ELECTRÓNICO",
	"18": "TARJETA PREPAGO",
	"19": "TARJETA DE CRÉDITO",
	"20": "OTROS CON UTILIZACIÓN DEL SISTEMA FINANCIERO",
	"21": "ENDOSO DE TÍTULOS",
}

func rideEnvironment(sriEnv string) string {
	if sriEnv == "2" {
		return "PRODUCCIÓN"
	}
	return "PRUEBAS"
}

// tenantLogoDataURI embeds the tenant's logo as a data: URI (Gotenberg loads
// nothing else); empty when the tenant has none.
func tenantLogoDataURI(tenantObj tenant.Tenant) template.URL {
	if tenantObj.LogoPath == nil {
		return ""
	}
	logoBytes, err := os.ReadFile(*tenantObj.LogoPath)
	if err != nil {
		return ""
	}
	return template.URL(fmt.Sprintf("data:%s;base64,%s", http.DetectContentType(logoBytes), base64.StdEncoding.EncodeToString(logoBytes)))
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// noteRideDocument is the part of a nota the shared RIDE header reads.
type noteRideDocument struct {
	kind          string
	id            uint
	accessKey     *string
	authorizedAt  *time.Time
	issueDate     time.Time
	establishment string
	emissionPoint string
	sequential    string
	invoice       billing_models.Invoice // the modified invoice, with its Patient
}

func buildNoteRideHeader(doc noteRideDocument, tenantObj tenant.Tenant, establishmentAddress string, sriEnv string) (billing_templates.NoteRideHeader, error) {
	if doc.accessKey == nil || *doc.accessKey == "" {
		return billing_templates.NoteRideHeader{}, fmt.Errorf("%s %d has no access key yet", doc.kind, doc.id)
	}
	barcode, err := GenerateBarcodeBase64(*doc.accessKey)
	if err != nil {
		return billing_templates.NoteRideHeader{}, err
	}
	authDate := ""
	if doc.authorizedAt != nil {
		authDate = doc.authorizedAt.Format("02/01/2006 15:04:05")
	}
	accountingObliged := "NO"
	if tenantObj.AccountingObliged {
		accountingObliged = "SI"
	}

	// Same buyer as the modified invoice.
	buyer := invoiceBuyer(doc.kind+"_ride", doc.id, doc.invoice)

	return billing_templates.NoteRideHeader{
		TradeName:            tenantObj.TradeName,
		CorporateName:        tenantObj.CorporateName,
		TaxID:                tenantObj.TaxID,
		MainAddress:          tenantObj.Address,
		EstablishmentAddress: establishmentAddress,
		SpecialContributor:   deref(tenantObj.SpecialContributorNumber),
		AccountingObliged:    accountingObliged,
		WithholdingAgent:     deref(tenantObj.WithholdingAgent),
		RimpeTaxpayer:        deref(tenantObj.RimpeTaxpayer),
		LogoDataURI:          tenantLogoDataURI(tenantObj),
		Establishment:        doc.establishment,
		EmissionPoint:        doc.emissionPoint,
		Sequential:           doc.sequential,
		AccessKey:            *doc.accessKey,
		AuthorizationDate:    authDate,
		Environment:          rideEnvironment(sriEnv),
		EmissionType:         "NORMAL",
		BarcodeBase64:        barcode,
		BuyerName:            buyer.Name,
		BuyerIdentification:  buyer.Identification,
		IssueDate:            doc.issueDate.Format("02/01/2006"),
		ModifiedDocType:      "FACTURA",
		ModifiedDocNumber:    fmt.Sprintf("%s-%s-%s", doc.invoice.EstablishmentCode, doc.invoice.EmissionPointCode, doc.invoice.Sequential),
		ModifiedDocIssueDate: doc.invoice.IssueDate.Format("02/01/2006"),
		AdditionalInfo:       rideAdditionalInfo(doc.invoice, buyer),
	}, nil
}

// taxSummary accumulates the tax rows of the totals box, one per tax and rate.
type taxSummary struct {
	order []string
	rows  map[string]*taxRow
}

type taxRow struct {
	label       string
	base, value float64
}

func (s *taxSummary) add(label string, base, value float64) {
	if s.rows == nil {
		s.rows = map[string]*taxRow{}
	}
	row, ok := s.rows[label]
	if !ok {
		row = &taxRow{label: label}
		s.rows[label] = row
		s.order = append(s.order, label)
	}
	row.base += base
	row.value += value
}

func (s *taxSummary) lines() []billing_templates.InvoiceRideTaxSummary {
	labels := append([]string(nil), s.order...)
	sort.Strings(labels)
	var out []billing_templates.InvoiceRideTaxSummary
	for _, label := range labels {
		row := s.rows[label]
		out = append(out, billing_templates.InvoiceRideTaxSummary{
			Label: row.label,
			Base:  fmt.Sprintf("%.2f", row.base),
			Value: fmt.Sprintf("%.2f", row.value),
		})
	}
	return out
}

func percentLabel(prefix string, rate float64) string {
	return fmt.Sprintf("%s %s%%", prefix, trimZeros(rate*100))
}

func trimZeros(v float64) string {
	s := fmt.Sprintf("%.2f", v)
	for len(s) > 0 && s[len(s)-1] == '0' {
		s = s[:len(s)-1]
	}
	if len(s) > 0 && s[len(s)-1] == '.' {
		s = s[:len(s)-1]
	}
	return s
}

// buildCreditNoteRideData builds the RIDE of a nota de crédito. creditNote
// must be loaded with its Items and Invoice (with Patient); codes maps each
// item's ProductID to its catalog code (SKU).
func buildCreditNoteRideData(creditNote billing_models.CreditNote, codes map[uint]string, tenantObj tenant.Tenant, establishmentAddress string, sriEnv string) (billing_templates.CreditNoteRideData, error) {
	header, err := buildNoteRideHeader(noteRideDocument{
		kind: "credit_note", id: creditNote.ID, accessKey: creditNote.AccessKey, authorizedAt: creditNote.AuthorizedAt,
		issueDate: creditNote.IssueDate, establishment: creditNote.EstablishmentCode, emissionPoint: creditNote.EmissionPointCode,
		sequential: creditNote.Sequential, invoice: creditNote.Invoice,
	}, tenantObj, establishmentAddress, sriEnv)
	if err != nil {
		return billing_templates.CreditNoteRideData{}, err
	}

	var items []billing_templates.CreditNoteRideItem
	var discount float64
	var taxes taxSummary
	for _, item := range creditNote.Items {
		items = append(items, billing_templates.CreditNoteRideItem{
			Code:        codes[item.ProductID],
			Quantity:    fmt.Sprintf("%.2f", item.Quantity),
			Description: item.Description,
			UnitPrice:   fmt.Sprintf("%.2f", item.UnitPrice),
			Discount:    fmt.Sprintf("%.2f", item.Discount),
			Total:       fmt.Sprintf("%.2f", item.Subtotal),
		})
		discount += item.Discount
		if item.TaxCode != "" && item.TaxPercentage != "" {
			taxes.add(percentLabel("IVA", item.TaxRate), item.Subtotal, item.Subtotal*item.TaxRate)
		}
		if item.IceTaxCode != "" && item.IceTaxCode != "3000" && item.IceTaxPercentage != "" {
			taxes.add("ICE", item.Subtotal, item.Subtotal*item.IceTax)
		}
	}

	return billing_templates.CreditNoteRideData{
		NoteRideHeader: header,
		Reason:         creditNote.Reason,
		Items:          items,
		Subtotal:       fmt.Sprintf("%.2f", creditNote.Subtotal),
		Discount:       fmt.Sprintf("%.2f", discount),
		TaxSummary:     taxes.lines(),
		Total:          fmt.Sprintf("%.2f", creditNote.Total),
	}, nil
}

// buildDebitNoteRideData builds the RIDE of a nota de débito. debitNote must be
// loaded with its Motives and Invoice (with Patient). The forma de pago is the
// modified invoice's, for the note's total.
func buildDebitNoteRideData(debitNote billing_models.DebitNote, tenantObj tenant.Tenant, establishmentAddress string, sriEnv string) (billing_templates.DebitNoteRideData, error) {
	header, err := buildNoteRideHeader(noteRideDocument{
		kind: "debit_note", id: debitNote.ID, accessKey: debitNote.AccessKey, authorizedAt: debitNote.AuthorizedAt,
		issueDate: debitNote.IssueDate, establishment: debitNote.EstablishmentCode, emissionPoint: debitNote.EmissionPointCode,
		sequential: debitNote.Sequential, invoice: debitNote.Invoice,
	}, tenantObj, establishmentAddress, sriEnv)
	if err != nil {
		return billing_templates.DebitNoteRideData{}, err
	}

	var motives []billing_templates.DebitNoteRideMotive
	var taxes taxSummary
	for _, motive := range debitNote.Motives {
		motives = append(motives, billing_templates.DebitNoteRideMotive{
			Reason: motive.Reason,
			Value:  fmt.Sprintf("%.2f", motive.Value),
		})
		if motive.TaxCode != "" && motive.TaxPercentageCode != "" {
			taxes.add(percentLabel("IVA", motive.TaxRate), motive.Value, motive.Value*motive.TaxRate)
		}
	}

	var payments []billing_templates.RidePayment
	if code := debitNote.Invoice.PaymentMethod; code != "" {
		method, ok := paymentMethodLabels[code]
		if !ok {
			method = code
		}
		payments = append(payments, billing_templates.RidePayment{Method: method, Value: fmt.Sprintf("%.2f", debitNote.Total)})
	}

	return billing_templates.DebitNoteRideData{
		NoteRideHeader: header,
		Motives:        motives,
		Payments:       payments,
		Subtotal:       fmt.Sprintf("%.2f", debitNote.Subtotal),
		TaxSummary:     taxes.lines(),
		Total:          fmt.Sprintf("%.2f", debitNote.Total),
	}, nil
}

// GenerateCreditNoteRide renders the RIDE of an authorized nota de crédito.
func GenerateCreditNoteRide(renderer *pdfrender.Renderer, creditNote billing_models.CreditNote, codes map[uint]string, tenantObj tenant.Tenant, establishmentAddress string, sriEnv string) ([]byte, error) {
	data, err := buildCreditNoteRideData(creditNote, codes, tenantObj, establishmentAddress, sriEnv)
	if err != nil {
		return nil, err
	}
	return renderer.Render(tenantObj.ID, billing_templates.CreditNoteRide, data)
}

// GenerateDebitNoteRide renders the RIDE of an authorized nota de débito.
func GenerateDebitNoteRide(renderer *pdfrender.Renderer, debitNote billing_models.DebitNote, tenantObj tenant.Tenant, establishmentAddress string, sriEnv string) ([]byte, error) {
	data, err := buildDebitNoteRideData(debitNote, tenantObj, establishmentAddress, sriEnv)
	if err != nil {
		return nil, err
	}
	return renderer.Render(tenantObj.ID, billing_templates.DebitNoteRide, data)
}
