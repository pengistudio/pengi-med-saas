package services

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"pengi-med-saas/core/utils"

	"go.uber.org/zap"
)

// Shared pipeline states, reused by every SRI document type worker (factura, nota de
// crédito, nota de débito, ...).
const (
	StatusSigned     = "signed"
	StatusValidated  = "validated"
	StatusAuthorized = "authorized"
)

// SriAuthorization carries the SRI's authorization response for the "authorized" step.
// There is no separate "número de autorización" in the current SRI scheme — it is the
// same 49-digit AccessKey; AuthorizedAt (fechaAutorizacion) is the new information here.
type SriAuthorization struct {
	AuthorizedAt time.Time
}

// RunSriPipeline signs, stores, validates and authorizes an already-generated SRI XML
// document. It is agnostic to the document type — callers provide the raw XML, access
// key and a callback to persist status progress on their own model between steps.
// auth is non-nil only when status == StatusAuthorized.
func RunSriPipeline(
	logger *zap.Logger,
	tenantID uint,
	storageFolder string, // e.g. "invoices", "credit-notes", "debit-notes"
	p12Path string,
	p12Password string,
	rawXML string,
	accessKey string,
	sriEnv string,
	onStatus func(status string, auth *SriAuthorization) error,
) error {
	p12Buffer, err := os.ReadFile(p12Path)
	if err != nil {
		return fmt.Errorf("failed to read P12 file: %w", err)
	}

	sriClient := utils.NewSriSignerClient()

	signResp, err := sriClient.SignXML(p12Buffer, p12Password, []byte(rawXML))
	if err != nil {
		return fmt.Errorf("failed to sign XML: %w", err)
	}
	finalXML := signResp.XML

	destDir := filepath.Join("storage", "tenants", fmt.Sprint(tenantID), storageFolder)
	if err := os.MkdirAll(destDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create storage dir: %w", err)
	}
	xmlPath := filepath.Join(destDir, fmt.Sprintf("%s.xml", accessKey))
	if err := os.WriteFile(xmlPath, []byte(finalXML), 0644); err != nil {
		return fmt.Errorf("failed to save signed XML: %w", err)
	}

	if err := onStatus(StatusSigned, nil); err != nil {
		return err
	}

	signerEnv := toSignerEnv(sriEnv)
	if _, err := sriClient.ValidateXMLWithSRI([]byte(finalXML), signerEnv); err != nil {
		return fmt.Errorf("failed to validate XML with SRI: %w", err)
	}
	if err := onStatus(StatusValidated, nil); err != nil {
		return err
	}

	authResp, err := sriClient.AuthorizeXMLWithSRI(accessKey, signerEnv)
	if err != nil {
		// SRI may still be processing; leave the document as validated so it can be retried.
		if strings.Contains(err.Error(), "autorizacion") || strings.Contains(err.Error(), "EN PROCESAMIENTO") {
			logger.Warn("SRI authorization pending, document left as validated", zap.String("access_key", accessKey), zap.Error(err))
			return nil
		}
		return fmt.Errorf("failed to authorize XML with SRI: %w", err)
	}

	auth := &SriAuthorization{}
	if parsedDate, parseErr := time.Parse("2006-01-02T15:04:05.999-07:00", authResp.FechaAutorizacion); parseErr == nil {
		auth.AuthorizedAt = parsedDate
	} else {
		logger.Warn("Failed to parse fechaAutorizacion from SRI, using current time", zap.String("raw", authResp.FechaAutorizacion), zap.Error(parseErr))
		auth.AuthorizedAt = time.Now()
	}

	return onStatus(StatusAuthorized, auth)
}

// ResolveSriEnv reads SRI_ENV and normalizes it to the SRI ambiente code used in the
// XML and access key: "1" pruebas, "2" producción. Accepts either the numeric code or
// the human-readable "test"/"prod" (case-insensitive); anything else defaults to pruebas.
func ResolveSriEnv() string {
	sriEnv := os.Getenv("SRI_ENV")
	if sriEnv == "2" || strings.EqualFold(sriEnv, "prod") {
		return "2"
	}
	return "1"
}

// toSignerEnv translates the SRI ambiente code ("1"/"2") into the "test"/"prod" path
// segment expected by the sri-xml-signer microservice's routes.
func toSignerEnv(sriEnv string) string {
	if sriEnv == "2" {
		return "prod"
	}
	return "test"
}
