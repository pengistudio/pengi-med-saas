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
