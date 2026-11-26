package whatsapp_handlers

import (
	"strings"
	"time"

	"pengi-med-saas/core/tenantdb"
	"pengi-med-saas/core/whatsapp"
	whatsapp_models "pengi-med-saas/features/whatsapp/models"

	"go.uber.org/zap"
)

// appDataSyncWait is how long a Coexistence connect waits for the app data
// sync requests before answering (a var so tests can shorten it).
var appDataSyncWait = 3 * time.Second

// requestAppDataSync asks Meta for the data of a newly connected Coexistence
// number's WhatsApp Business app: contacts, then chat history. Meta requires
// both within 24 h of the onboarding and accepts each only once, so nothing is
// retried: a failure is logged, stored on the account and doesn't fail the
// connect. It waits up to wait; after that the requests finish in background.
func (h *WhatsAppHandler) requestAppDataSync(tenantID uint, token, phoneNumberID string, wait time.Duration) {
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		h.runAppDataSync(tenantID, token, phoneNumberID)
	}()
	select {
	case <-finished:
	case <-time.After(wait):
		h.logger.Warn("whatsapp app data sync still running, finishing in background", zap.Uint("tenant_id", tenantID), zap.Duration("waited", wait))
	}
}

// runAppDataSync makes both requests, in the order Meta wants, and stores each
// outcome as it comes (a deploy may stop it halfway).
func (h *WhatsAppHandler) runAppDataSync(tenantID uint, token, phoneNumberID string) {
	// Only these columns, and only while the number is still the tenant's: the
	// webhook or a new connect may be changing the account meanwhile.
	save := func(values map[string]any) {
		if err := tenantdb.ForTenant(h.db, tenantID).Model(&whatsapp_models.WhatsAppAccount{}).
			Where("tenant_id = ? AND phone_number_id = ?", tenantID, phoneNumberID).Updates(values).Error; err != nil {
			h.logger.Error("failed to save whatsapp app data sync", zap.Uint("tenant_id", tenantID), zap.Error(err))
		}
	}
	save(map[string]any{"app_data_sync_at": time.Now()})

	var failures []string
	for _, step := range []struct{ syncType, column string }{
		{whatsapp.SyncContacts, "contacts_sync_request_id"},
		{whatsapp.SyncHistory, "history_sync_request_id"},
	} {
		log := h.logger.With(zap.Uint("tenant_id", tenantID), zap.String("phone_number_id", phoneNumberID), zap.String("sync_type", step.syncType))
		requestID, err := h.client.RequestAppDataSync(token, phoneNumberID, step.syncType)
		if err != nil {
			log.Warn("whatsapp app data sync failed", zap.Error(err))
			failures = append(failures, step.syncType+": "+err.Error())
			save(map[string]any{"app_data_sync_error": strings.Join(failures, "; ")})
			continue
		}
		log.Info("whatsapp app data sync requested", zap.String("request_id", requestID))
		save(map[string]any{step.column: requestID})
	}
}
