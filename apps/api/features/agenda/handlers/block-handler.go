package agenda_handlers

import (
	"net/http"
	"strings"
	"time"

	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	"pengi-med-saas/core/tenantdb"
	agenda_dto "pengi-med-saas/features/agenda/dto"
	agenda_models "pengi-med-saas/features/agenda/models"
	agenda_services "pengi-med-saas/features/agenda/services"
	company_models "pengi-med-saas/features/companies/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ─── Doctor :id (admin) ──────────────────────────────────────────────────────

func (h *AgendaHandler) ListDoctorBlocks(c *gin.Context) envelope.Response {
	db := tenantdb.For(c, h.db)
	doctor, errResp := h.doctorParam(c, db)
	if errResp != nil {
		return *errResp
	}
	return h.listBlocks(c, db, &doctor.ID)
}

func (h *AgendaHandler) CreateDoctorBlock(c *gin.Context) envelope.Response {
	db := tenantdb.For(c, h.db)
	doctor, errResp := h.doctorParam(c, db)
	if errResp != nil {
		return *errResp
	}
	return h.saveBlock(c, db, &doctor.ID, 0)
}

func (h *AgendaHandler) UpdateDoctorBlock(c *gin.Context) envelope.Response {
	db := tenantdb.For(c, h.db)
	doctor, errResp := h.doctorParam(c, db)
	if errResp != nil {
		return *errResp
	}
	blockID, ok := uintParam(c, "blockId")
	if !ok {
		return blockNotFound()
	}
	return h.saveBlock(c, db, &doctor.ID, blockID)
}

func (h *AgendaHandler) DeleteDoctorBlock(c *gin.Context) envelope.Response {
	db := tenantdb.For(c, h.db)
	doctor, errResp := h.doctorParam(c, db)
	if errResp != nil {
		return *errResp
	}
	return h.deleteBlock(c, db, &doctor.ID, "blockId")
}

// ─── Own doctor (/doctors/me) ────────────────────────────────────────────────

func (h *AgendaHandler) ListMyBlocks(c *gin.Context) envelope.Response {
	db := tenantdb.For(c, h.db)
	doctor, errResp := h.ownDoctor(c, db)
	if errResp != nil {
		return *errResp
	}
	return h.listBlocks(c, db, &doctor.ID)
}

func (h *AgendaHandler) CreateMyBlock(c *gin.Context) envelope.Response {
	db := tenantdb.For(c, h.db)
	doctor, errResp := h.ownDoctor(c, db)
	if errResp != nil {
		return *errResp
	}
	return h.saveBlock(c, db, &doctor.ID, 0)
}

func (h *AgendaHandler) UpdateMyBlock(c *gin.Context) envelope.Response {
	db := tenantdb.For(c, h.db)
	doctor, errResp := h.ownDoctor(c, db)
	if errResp != nil {
		return *errResp
	}
	blockID, ok := uintParam(c, "blockId")
	if !ok {
		return blockNotFound()
	}
	return h.saveBlock(c, db, &doctor.ID, blockID)
}

func (h *AgendaHandler) DeleteMyBlock(c *gin.Context) envelope.Response {
	db := tenantdb.For(c, h.db)
	doctor, errResp := h.ownDoctor(c, db)
	if errResp != nil {
		return *errResp
	}
	return h.deleteBlock(c, db, &doctor.ID, "blockId")
}

// ─── Clinic-wide (/agenda/blocks) ────────────────────────────────────────────

func (h *AgendaHandler) ListClinicBlocks(c *gin.Context) envelope.Response {
	return h.listBlocks(c, tenantdb.For(c, h.db), nil)
}

func (h *AgendaHandler) CreateClinicBlock(c *gin.Context) envelope.Response {
	return h.saveBlock(c, tenantdb.For(c, h.db), nil, 0)
}

func (h *AgendaHandler) UpdateClinicBlock(c *gin.Context) envelope.Response {
	blockID, ok := uintParam(c, "id")
	if !ok {
		return blockNotFound()
	}
	return h.saveBlock(c, tenantdb.For(c, h.db), nil, blockID)
}

func (h *AgendaHandler) DeleteClinicBlock(c *gin.Context) envelope.Response {
	return h.deleteBlock(c, tenantdb.For(c, h.db), nil, "id")
}

// ─── Shared ──────────────────────────────────────────────────────────────────

func blockNotFound() envelope.Response {
	return envelope.ErrorResponse(http.StatusNotFound, "agenda.error.block_not_found", core_errors.ErrAgendaBlockNotFound)
}

// ownerScope limits a block query to doctorID's blocks, or to clinic-wide
// blocks when doctorID is nil.
func ownerScope(db *gorm.DB, doctorID *uint) *gorm.DB {
	if doctorID == nil {
		return db.Where("doctor_id IS NULL")
	}
	return db.Where("doctor_id = ?", *doctorID)
}

// listBlocks lists the blocks of one owner that end on or after ?from
// (default today) and, with ?to, start on or before it.
func (h *AgendaHandler) listBlocks(c *gin.Context, db *gorm.DB, doctorID *uint) envelope.Response {
	from := c.DefaultQuery("from", time.Now().In(company_models.SubscriptionLocation).Format(agenda_services.DateLayout))
	if _, ok := agenda_services.ParseDate(from); !ok {
		return badRequest("agenda.error.invalid_date_range", core_errors.ErrAgendaInvalidDateRange)
	}
	q := ownerScope(db, doctorID).Where("end_date >= ?", from)
	if to := c.Query("to"); to != "" {
		if _, ok := agenda_services.ParseDate(to); !ok || to < from {
			return badRequest("agenda.error.invalid_date_range", core_errors.ErrAgendaInvalidDateRange)
		}
		q = q.Where("start_date <= ?", to)
	}
	blocks := []agenda_models.ScheduleBlock{}
	if err := q.Order("start_date, start_time, id").Find(&blocks).Error; err != nil {
		return h.internal("failed to list schedule blocks", err)
	}
	return envelope.SuccessResponse(blocks, "agenda.blocks.list.success")
}

// validateBlock normalizes and checks a block request.
func validateBlock(req *agenda_dto.BlockRequest) *envelope.Response {
	req.StartDate = strings.TrimSpace(req.StartDate)
	req.EndDate = strings.TrimSpace(req.EndDate)
	req.StartTime = strings.TrimSpace(req.StartTime)
	req.EndTime = strings.TrimSpace(req.EndTime)
	req.Reason = strings.TrimSpace(req.Reason)
	_, ok1 := agenda_services.ParseDate(req.StartDate)
	_, ok2 := agenda_services.ParseDate(req.EndDate)
	if !ok1 || !ok2 || req.StartDate > req.EndDate {
		resp := badRequest("agenda.error.invalid_date_range", core_errors.ErrAgendaInvalidDateRange)
		return &resp
	}
	if req.StartTime == "" && req.EndTime == "" {
		return nil
	}
	if _, _, ok := agenda_services.ValidRange(req.StartTime, req.EndTime, stepMinutes); !ok {
		resp := badRequest("agenda.error.invalid_time", core_errors.ErrAgendaInvalidTime)
		return &resp
	}
	return nil
}

// saveBlock creates (blockID 0) or replaces a block of one owner and answers
// with the appointments that fall inside it.
func (h *AgendaHandler) saveBlock(c *gin.Context, db *gorm.DB, doctorID *uint, blockID uint) envelope.Response {
	var req agenda_dto.BlockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return invalidRequest()
	}
	if errResp := validateBlock(&req); errResp != nil {
		return *errResp
	}

	block := agenda_models.ScheduleBlock{TenantID: tenantdb.TenantID(c), DoctorID: doctorID}
	if blockID != 0 {
		if err := ownerScope(db, doctorID).Limit(1).Find(&block, blockID).Error; err != nil {
			return h.internal("failed to fetch schedule block", err)
		}
		if block.ID == 0 {
			return blockNotFound()
		}
	}
	block.StartDate, block.EndDate = req.StartDate, req.EndDate
	block.StartTime, block.EndTime = req.StartTime, req.EndTime
	block.Reason = req.Reason

	var err error
	if blockID == 0 {
		err = db.Create(&block).Error
	} else {
		err = db.Model(&block).Select("start_date", "end_date", "start_time", "end_time", "reason").Updates(&block).Error
	}
	if err != nil {
		return h.internal("failed to save schedule block", err)
	}

	affected, err := agenda_services.AffectedAppointments(db, block)
	if err != nil {
		return h.internal("failed to list appointments affected by a block", err)
	}
	data := agenda_dto.BlockResponse{Block: block, AffectedAppointments: affected}
	if blockID == 0 {
		return envelope.SuccessResponse(data, "agenda.blocks.create.success")
	}
	return envelope.SuccessResponse(data, "agenda.blocks.update.success")
}

func (h *AgendaHandler) deleteBlock(c *gin.Context, db *gorm.DB, doctorID *uint, param string) envelope.Response {
	blockID, ok := uintParam(c, param)
	if !ok {
		return blockNotFound()
	}
	var block agenda_models.ScheduleBlock
	if err := ownerScope(db, doctorID).Limit(1).Find(&block, blockID).Error; err != nil {
		return h.internal("failed to fetch schedule block", err)
	}
	if block.ID == 0 {
		return blockNotFound()
	}
	if err := db.Delete(&block).Error; err != nil {
		return h.internal("failed to delete schedule block", err)
	}
	return envelope.SuccessResponse(nil, "agenda.blocks.delete.success")
}
