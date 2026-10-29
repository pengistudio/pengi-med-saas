package waitlist_services

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"pengi-med-saas/core/tenantdb"
	"pengi-med-saas/core/whatsapp"
	clinical_models "pengi-med-saas/features/clinical/models"
	company_models "pengi-med-saas/features/companies/models"
	company_services "pengi-med-saas/features/companies/services"
	waitlist_models "pengi-med-saas/features/waitlist/models"
	whatsapp_models "pengi-med-saas/features/whatsapp/models"
	whatsapp_services "pengi-med-saas/features/whatsapp/services"

	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Why a slot can't be offered by WhatsApp.
var (
	ErrWhatsAppNotInPlan   = errors.New("the plan does not include WhatsApp")
	ErrNotConnected        = errors.New("whatsapp not connected")
	ErrTemplateUnavailable = errors.New("slot offer template not approved")
	ErrNoOptIn             = errors.New("patient has not opted in")
	ErrInvalidPhone        = errors.New("patient has no valid phone")
	ErrMonthlyLimit        = errors.New("monthly whatsapp limit reached")
	ErrSendFailed          = errors.New("whatsapp message not sent")
	// ErrAlreadyOffered: this slot was already offered to the entry.
	ErrAlreadyOffered = errors.New("slot already offered to this entry")
	// ErrOwnSlot: the slot was freed by the entry's own patient.
	ErrOwnSlot = errors.New("slot freed by the same patient")
	// ErrNotActive: the entry left the list.
	ErrNotActive = errors.New("waitlist entry not active")
)

// PlanIncludesWhatsApp reports whether the tenant's active subscription has
// a plan with the WHATSAPP feature (no company or subscription: false).
func PlanIncludesWhatsApp(db *gorm.DB, tenantID uint) (bool, error) {
	var company company_models.Company
	if err := tenantdb.ForTenant(db, tenantID).Select("id").Limit(1).Find(&company).Error; err != nil || company.ID == 0 {
		return false, err
	}
	var sub company_models.Subscription
	err := company_services.ActiveSubscriptionQuery(tenantdb.System(db), company.ID).
		Preload("Plan.Features.Permissions").Order("expires_at DESC").Limit(1).Find(&sub).Error
	if err != nil || sub.ID == 0 {
		return false, err
	}
	return company_services.CalculateEnabledFeaturesFromFeatures(sub.Plan.Features).WhatsApp, nil
}

// whatsAppAccount returns the clinic's connected account and its token when
// the plan includes WhatsApp and the slot offer template is approved.
func whatsAppAccount(db *gorm.DB, tenantID uint) (whatsapp_models.WhatsAppAccount, string, error) {
	var account whatsapp_models.WhatsAppAccount
	inPlan, err := PlanIncludesWhatsApp(db, tenantID)
	if err != nil {
		return account, "", err
	}
	if !inPlan {
		return account, "", ErrWhatsAppNotInPlan
	}
	scoped := tenantdb.ForTenant(db, tenantID)
	if err := scoped.Limit(1).Find(&account).Error; err != nil {
		return account, "", err
	}
	if account.ID == 0 || account.Status != whatsapp_models.AccountStatusConnected {
		return account, "", ErrNotConnected
	}
	rows, err := whatsapp_services.TemplateRows(scoped)
	if err != nil {
		return account, "", err
	}
	if rows[whatsapp_models.TemplateSlotOffer].Status != whatsapp_models.TemplateStatusApproved {
		return account, "", ErrTemplateUnavailable
	}
	token, err := account.OpenAccessToken()
	if err != nil {
		return account, "", ErrNotConnected
	}
	return account, token, nil
}

// WhatsAppReady reports whether the clinic can offer slots by WhatsApp: the
// plan includes it, the account is connected and the template approved. The
// patient's consent, phone and the monthly cap decide each offer.
func WhatsAppReady(db *gorm.DB, tenantID uint) bool {
	_, _, err := whatsAppAccount(db, tenantID)
	return err == nil
}

// CanReceiveWhatsApp reports whether patient can be offered a slot by WhatsApp.
func CanReceiveWhatsApp(patient *clinical_models.Patient) bool {
	if patient == nil || !patient.WhatsAppOptIn {
		return false
	}
	_, err := whatsapp_services.NormalizePhone(patient.Phone)
	return err == nil
}

// Offers offers freed slots to waitlist entries. Offering marks the entry
// offered with the slot; offering the same slot to the same entry again is
// refused (ErrAlreadyOffered), so one click never sends two messages.
type Offers struct {
	db     *gorm.DB
	logger *zap.Logger
	client *whatsapp.Client
	now    func() time.Time
	loc    *time.Location
}

func NewOffers(db *gorm.DB, logger *zap.Logger, client *whatsapp.Client) *Offers {
	return &Offers{db: db, logger: logger, client: client, now: time.Now, loc: Location()}
}

// WithClock overrides the clock and time zone (tests).
func (o *Offers) WithClock(now func() time.Time, loc *time.Location) *Offers {
	o.now, o.loc = now, loc
	return o
}

// Offer offers slot to entry (with its Patient loaded) by channel. Phone
// only records it (reception calls). WhatsApp also sends the slot offer
// template (kind slot_offer, counted toward the monthly cap, in the
// patient's thread) whose "Lo quiero" button carries "slot_offer:<message
// id>"; when it can't be sent the entry is left as it was. userID is the
// staff member offering (nil: unknown).
func (o *Offers) Offer(tenantID uint, entry waitlist_models.WaitlistEntry, slot Slot, channel string, userID *uint) (waitlist_models.WaitlistEntry, error) {
	db := tenantdb.ForTenant(o.db, tenantID)
	now := o.now()
	if !entry.Active() {
		return entry, ErrNotActive
	}
	if slot.PatientID != 0 && entry.PatientID == slot.PatientID {
		return entry, ErrOwnSlot
	}
	// Reported before any send check: the history says it was offered.
	if done, err := OfferedSlots(db, []uint{entry.ID}, slot); err != nil {
		return entry, err
	} else if done[entry.ID] {
		return entry, ErrAlreadyOffered
	}
	var (
		account whatsapp_models.WhatsAppAccount
		token   string
		to      string
	)
	if channel == waitlist_models.ChannelWhatsApp {
		var err error
		if account, token, err = whatsAppAccount(o.db, tenantID); err != nil {
			return entry, err
		}
		if o.client == nil {
			return entry, ErrNotConnected
		}
		if entry.Patient == nil || !entry.Patient.WhatsAppOptIn {
			return entry, ErrNoOptIn
		}
		if to, err = whatsapp_services.NormalizePhone(entry.Patient.Phone); err != nil {
			return entry, ErrInvalidPhone
		}
		allowed, _, err := whatsapp_services.CanSendTemplate(o.db, tenantID, now)
		if err != nil {
			return entry, err
		}
		if !allowed {
			return entry, ErrMonthlyLimit
		}
	}

	// Claim, in one transaction: the offer history row (its unique index
	// refuses a slot already offered to the entry, even by a concurrent
	// click) and the entry, still active, marked offered.
	at := now.UTC()
	offer := waitlist_models.WaitlistOffer{TenantID: tenantID, EntryID: entry.ID, Date: slot.Date,
		StartTime: slot.StartTime, DoctorID: uintOr0(slot.DoctorID), Channel: channel}
	offer.CreatedAt = at
	err := db.Transaction(func(tx *gorm.DB) error {
		ins := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&offer)
		if ins.Error != nil {
			return ins.Error
		}
		if ins.RowsAffected == 0 {
			return ErrAlreadyOffered
		}
		claim := tx.Model(&waitlist_models.WaitlistEntry{}).
			Where("id = ? AND status IN ?", entry.ID, waitlist_models.ActiveStatuses).
			Updates(map[string]any{
				"status": waitlist_models.StatusOffered, "offered_at": at, "offered_doctor_id": slot.DoctorID,
				"offered_date": slot.Date, "offered_start_time": slot.StartTime, "offered_end_time": slot.EndTime,
				"offer_channel": channel, "offer_message_id": nil, "offer_accepted_at": nil, "updated_at": at,
			})
		if claim.Error != nil {
			return claim.Error
		}
		if claim.RowsAffected == 0 {
			return ErrNotActive
		}
		return nil
	})
	if err != nil {
		return entry, err
	}

	if channel == waitlist_models.ChannelWhatsApp {
		msgID, err := o.send(db, tenantID, entry, slot, account, token, to, userID)
		if err != nil {
			o.restore(db, entry, offer.ID)
			return entry, err
		}
		if err := db.Model(&waitlist_models.WaitlistOffer{}).Where("id = ?", offer.ID).Update("message_id", msgID).Error; err != nil {
			o.logger.Error("waitlist: offer sent but not linked to its history row", zap.Uint("offer_id", offer.ID), zap.Error(err))
		}
		if err := db.Model(&waitlist_models.WaitlistEntry{}).Where("id = ?", entry.ID).Update("offer_message_id", msgID).Error; err != nil {
			o.logger.Error("waitlist: offer sent but not linked to its entry", zap.Uint("entry_id", entry.ID), zap.Uint("message_id", msgID), zap.Error(err))
		}
	}
	var updated waitlist_models.WaitlistEntry
	if err := db.Preload("Patient").Preload("Doctor").Preload("AppointmentType").Where("id = ?", entry.ID).First(&updated).Error; err != nil {
		return entry, err
	}
	return updated, nil
}

// restore puts entry's offer back as it was before a failed WhatsApp offer
// and drops its history row offerID (nothing was offered).
func (o *Offers) restore(db *gorm.DB, entry waitlist_models.WaitlistEntry, offerID uint) {
	if err := db.Where("id = ?", offerID).Delete(&waitlist_models.WaitlistOffer{}).Error; err != nil {
		o.logger.Error("waitlist: failed to drop the history of a failed offer", zap.Uint("offer_id", offerID), zap.Error(err))
	}
	err := db.Model(&waitlist_models.WaitlistEntry{}).Where("id = ?", entry.ID).Updates(map[string]any{
		"status": entry.Status, "offered_at": entry.OfferedAt, "offered_doctor_id": entry.OfferedDoctorID,
		"offered_date": entry.OfferedDate, "offered_start_time": entry.OfferedStartTime, "offered_end_time": entry.OfferedEndTime,
		"offer_channel": entry.OfferChannel, "offer_message_id": entry.OfferMessageID, "offer_accepted_at": entry.OfferAcceptedAt,
	}).Error
	if err != nil {
		o.logger.Error("waitlist: failed to restore entry after a failed offer", zap.Uint("entry_id", entry.ID), zap.Error(err))
	}
}

// send records the slot offer in the patient's thread and sends it.
func (o *Offers) send(db *gorm.DB, tenantID uint, entry waitlist_models.WaitlistEntry, slot Slot, account whatsapp_models.WhatsAppAccount, token, to string, userID *uint) (uint, error) {
	start, err := slot.Start(o.loc)
	if err != nil {
		return 0, err
	}
	spec := whatsapp_services.SlotOfferTemplate
	params := whatsapp_services.TemplateParams(spec, whatsapp_services.TemplateData{
		PatientName: strings.TrimSpace(entry.Patient.FirstName + " " + entry.Patient.LastName),
		ClinicName:  whatsapp_services.ClinicName(o.db, tenantID),
		Start:       start,
		DoctorName:  DoctorName(db, slot.DoctorID),
	})
	patientID := entry.PatientID
	msg := whatsapp_models.WhatsAppMessage{
		TenantID:     tenantID,
		Kind:         whatsapp_models.KindSlotOffer,
		PatientID:    &patientID,
		ToPhone:      to,
		Template:     spec.Definition.Name,
		Status:       whatsapp_models.MessageStatusSending,
		Body:         whatsapp_services.RenderTemplateBody(spec, params),
		ContentType:  whatsapp_models.ContentText,
		SentByUserID: userID,
	}
	conv, err := whatsapp_services.EnsureConversation(db, tenantID, to, &patientID)
	if err != nil {
		return 0, err
	}
	if err := whatsapp_services.RecordOutbound(db, &conv, &msg, o.now()); err != nil {
		return 0, err
	}
	tpl := SlotOfferMessage(to, msg.ID, params)
	sender := whatsapp_services.NewSender(o.db, o.logger, o.client).WithClock(o.now, o.loc)
	if err := sender.DeliverTemplate(db, &msg, account, token, tpl); err != nil {
		o.logger.Warn("waitlist: slot offer not sent", zap.Uint("entry_id", entry.ID), zap.Uint("message_id", msg.ID), zap.Error(err))
		return 0, fmt.Errorf("%w: %v", ErrSendFailed, err)
	}
	return msg.ID, nil
}

// SlotOfferMessage builds the slot offer for message msgID; its button
// carries "slot_offer:<id>" so the webhook knows what was answered.
func SlotOfferMessage(to string, msgID uint, params []string) whatsapp.TemplateMessage {
	spec := whatsapp_services.SlotOfferTemplate
	return whatsapp.TemplateMessage{
		To:         to,
		Name:       spec.Definition.Name,
		Language:   spec.Definition.Language,
		BodyParams: params,
		Buttons:    []whatsapp.QuickReplyButton{{Payload: fmt.Sprintf("%s:%d", whatsapp_models.ReplySlotOffer, msgID)}},
	}
}

func uintOr0(p *uint) uint {
	if p == nil {
		return 0
	}
	return *p
}
