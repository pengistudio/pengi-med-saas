// Package reports_handlers serves the reports: revenue, receivables,
// appointments and clinical statistics. Nothing is stored: every report is an
// aggregation (GROUP BY) over existing tenant data, as JSON, CSV or a PDF
// rendered through core/pdfrender.
package reports_handlers

import (
	"fmt"
	"math"
	"net/http"
	"strconv"
	"time"

	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	"pengi-med-saas/core/pdfrender"
	"pengi-med-saas/core/tenantdb"
	reports_dto "pengi-med-saas/features/reports/dto"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// MaxRangeDays is the longest range a report accepts.
const MaxRangeDays = 366

// monthlyAfterDays: ranges longer than this group the revenue series by month.
const monthlyAfterDays = 90

// timeZone is the tenants' clock (stored times are UTC); Ecuador has a single
// time zone and no daylight saving. Its SQL spelling is in dayExpr.
const timeZone = "America/Guayaquil"

var location = func() *time.Location {
	if loc, err := time.LoadLocation(timeZone); err == nil {
		return loc
	}
	return time.FixedZone("ECT", -5*60*60)
}()

type ReportsHandler struct {
	db       *gorm.DB
	logger   *zap.Logger
	renderer *pdfrender.Renderer
}

func NewReportsHandler(db *gorm.DB, logger *zap.Logger, renderer *pdfrender.Renderer) *ReportsHandler {
	return &ReportsHandler{db: db, logger: logger, renderer: renderer}
}

// filter is the parsed query: the range as local midnights, [start, end).
type filter struct {
	from, to time.Time // first and last day, local midnight
	start    time.Time // from
	end      time.Time // the day after to
	days     int
	doctorID *uint
}

func (f filter) dto() reports_dto.Filters {
	return reports_dto.Filters{From: f.from.Format("2006-01-02"), To: f.to.Format("2006-01-02"), DoctorID: f.doctorID}
}

// utcDay is a local calendar day as UTC midnight: appointments store a plain
// date, compared with DATE(date).
func utcDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func parseFilter(c *gin.Context, now time.Time) (filter, *envelope.Response) {
	bad := func(key string, code core_errors.AppError) (filter, *envelope.Response) {
		r := envelope.ErrorResponse(http.StatusBadRequest, key, code)
		return filter{}, &r
	}
	today := now.In(location)
	today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, location)
	f := filter{to: today, from: today.AddDate(0, 0, -29)}
	var err error
	if v := c.Query("from"); v != "" {
		if f.from, err = time.ParseInLocation("2006-01-02", v, location); err != nil {
			return bad("reports.error.invalid_filters", core_errors.ErrReportsInvalidFilters)
		}
		if c.Query("to") == "" {
			f.to = f.from.AddDate(0, 0, 29)
		}
	}
	if v := c.Query("to"); v != "" {
		if f.to, err = time.ParseInLocation("2006-01-02", v, location); err != nil {
			return bad("reports.error.invalid_filters", core_errors.ErrReportsInvalidFilters)
		}
		if c.Query("from") == "" {
			f.from = f.to.AddDate(0, 0, -29)
		}
	}
	if f.to.Before(f.from) {
		return bad("reports.error.invalid_filters", core_errors.ErrReportsInvalidFilters)
	}
	f.days = int(math.Round(f.to.Sub(f.from).Hours()/24)) + 1
	if f.days > MaxRangeDays {
		return bad("reports.error.range_too_long", core_errors.ErrReportsRangeTooLong)
	}
	if v := c.Query("doctor_id"); v != "" {
		id, err := strconv.ParseUint(v, 10, 32)
		if err != nil || id == 0 {
			return bad("reports.error.invalid_filters", core_errors.ErrReportsInvalidFilters)
		}
		d := uint(id)
		f.doctorID = &d
	}
	// UTC, like the stored timestamps: SQLite (tests) compares them as text.
	f.start = f.from.UTC()
	f.end = f.to.AddDate(0, 0, 1).UTC()
	return f, nil
}

// dayExpr is the SQL for the tenant-local day (YYYY-MM-DD) of a timestamp
// column. Production is PostgreSQL; SQLite only runs the tests.
func dayExpr(db *gorm.DB, col string) string {
	if db.Dialector.Name() == "sqlite" {
		return fmt.Sprintf("strftime('%%Y-%%m-%%d', %s, '-5 hours')", col)
	}
	return fmt.Sprintf("to_char(%s AT TIME ZONE '%s', 'YYYY-MM-DD')", col, timeZone)
}

func round2(v float64) float64 { return math.Round(v*100) / 100 }

func (h *ReportsHandler) internal(c *gin.Context, msg string, err error) {
	h.logger.Error(msg, zap.Error(err))
	envelope.Write(c, envelope.ErrorResponse(http.StatusInternalServerError, "reports.error.internal", core_errors.ErrReportsInternal))
}

// serve parses the filters and format, builds the report with build and writes
// it: JSON through the envelope, or a CSV / PDF file.
func serve[T any](h *ReportsHandler, c *gin.Context, successKey string, build func(db *gorm.DB, tenantID uint, f filter) (T, error), out exporter[T]) {
	f, errResp := parseFilter(c, time.Now())
	if errResp != nil {
		envelope.Write(c, *errResp)
		return
	}
	format := c.Query("format")
	if format != "" && format != "csv" && format != "pdf" {
		envelope.Write(c, envelope.ErrorResponse(http.StatusBadRequest, "reports.error.invalid_format", core_errors.ErrReportsInvalidFormat))
		return
	}
	tenantID := tenantdb.TenantID(c)
	report, err := build(tenantdb.For(c, h.db), tenantID, f)
	if err != nil {
		h.internal(c, "failed to build report "+out.name, err)
		return
	}
	switch format {
	case "csv":
		h.writeCSV(c, out.name+"_"+f.from.Format("20060102")+"_"+f.to.Format("20060102"), out.sections(translator(c), report))
	case "pdf":
		writePDF(h, c, out, tenantID, f, report)
	default:
		envelope.Write(c, envelope.SuccessResponse(report, successKey))
	}
}
