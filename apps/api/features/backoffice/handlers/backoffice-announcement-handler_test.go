package backoffice_handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"pengi-med-saas/core/envelope"
	company_models "pengi-med-saas/features/companies/models"
	notifications_models "pengi-med-saas/features/notifications/models"
	notifications_workers "pengi-med-saas/features/notifications/workers"
	permission_models "pengi-med-saas/features/permissions/models"
	user_models "pengi-med-saas/features/users/models"
	"pengi-med-saas/testutils"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type announcementFixture struct {
	db       *gorm.DB
	router   *gin.Engine
	clinicA  company_models.Company
	clinicB  company_models.Company
	doctorA1 user_models.User
	doctorA2 user_models.User
	doctorB  user_models.User
}

// Two clinics on different tenants: A with two users, B with one.
func setupAnnouncements(t *testing.T) *announcementFixture {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db := testutils.SetupTestDB(t,
		&permission_models.Permission{}, &user_models.Role{}, &user_models.User{}, &user_models.Environment{},
		&company_models.Company{}, &notifications_models.Notification{}, &notifications_models.Announcement{},
	)

	f := &announcementFixture{db: db}
	f.clinicA = company_models.Company{TradeName: "Clínica A", TenantID: 9101}
	f.clinicB = company_models.Company{TradeName: "Clínica B", TenantID: 9102}
	for _, company := range []*company_models.Company{&f.clinicA, &f.clinicB} {
		if err := db.Create(company).Error; err != nil {
			t.Fatal(err)
		}
	}
	f.doctorA1 = user_models.User{UserName: "doctor-a1", Email: "a1@test.local"}
	f.doctorA2 = user_models.User{UserName: "doctor-a2", Email: "a2@test.local"}
	f.doctorB = user_models.User{UserName: "doctor-b", Email: "b@test.local"}
	for _, u := range []*user_models.User{&f.doctorA1, &f.doctorA2, &f.doctorB} {
		if err := db.Create(u).Error; err != nil {
			t.Fatal(err)
		}
	}
	for _, env := range []user_models.Environment{
		{UserID: f.doctorA1.ID, CompanyID: f.clinicA.ID},
		{UserID: f.doctorA2.ID, CompanyID: f.clinicA.ID},
		{UserID: f.doctorB.ID, CompanyID: f.clinicB.ID},
	} {
		if err := db.Create(&env).Error; err != nil {
			t.Fatal(err)
		}
	}

	h := NewBackofficeAnnouncementHandler(db, zap.NewNop())
	f.router = gin.New()
	f.router.POST("/backoffice/announcements", envelope.Handle(h.CreateAnnouncement))
	f.router.POST("/backoffice/announcements/:id/cancel", envelope.Handle(h.CancelAnnouncement))
	return f
}

func (f *announcementFixture) create(t *testing.T, body map[string]any) (*httptest.ResponseRecorder, notifications_models.Announcement) {
	t.Helper()
	payload, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/backoffice/announcements", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	f.router.ServeHTTP(w, req)

	var res struct {
		Data notifications_models.Announcement `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &res)
	return w, res.Data
}

// recipients maps each user notified about the announcement to its tenant.
func (f *announcementFixture) recipients(t *testing.T, announcementID uint) map[uint]uint {
	t.Helper()
	var notifications []notifications_models.Notification
	if err := f.db.Where("type = ? AND resource_id = ?", "announcement", announcementID).Find(&notifications).Error; err != nil {
		t.Fatal(err)
	}
	got := map[uint]uint{}
	for _, n := range notifications {
		got[n.UserID] = n.TenantID
	}
	return got
}

func TestCreateAnnouncement_GlobalReachesEveryUserOnTheirTenant(t *testing.T) {
	f := setupAnnouncements(t)

	w, a := f.create(t, map[string]any{"scope": "global", "title": "Mantenimiento", "body": "Esta noche", "level": "warning"})

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
	}
	got := f.recipients(t, a.ID)
	if got[f.doctorA1.ID] != f.clinicA.TenantID || got[f.doctorA2.ID] != f.clinicA.TenantID || got[f.doctorB.ID] != f.clinicB.TenantID {
		t.Fatalf("recipients = %v", got)
	}
	var stored notifications_models.Announcement
	f.db.First(&stored, a.ID)
	if stored.Status != notifications_models.AnnouncementStatusSent || stored.SentAt == nil || stored.RecipientCount != len(got) {
		t.Fatalf("announcement = %+v, want sent to %d", stored, len(got))
	}
}

func TestCreateAnnouncement_CompanyReachesOnlyItsUsers(t *testing.T) {
	f := setupAnnouncements(t)

	_, a := f.create(t, map[string]any{"scope": "company", "company_id": f.clinicA.ID, "title": "Hola", "body": "Clínica A", "level": "info"})

	got := f.recipients(t, a.ID)
	if len(got) != 2 || got[f.doctorA1.ID] == 0 || got[f.doctorA2.ID] == 0 {
		t.Fatalf("recipients = %v, want only clinic A users", got)
	}
}

func TestCreateAnnouncement_UserReachesOnlyThatUser(t *testing.T) {
	f := setupAnnouncements(t)

	_, a := f.create(t, map[string]any{
		"scope": "user", "company_id": f.clinicA.ID, "user_id": f.doctorA2.ID,
		"title": "Hola", "body": "Solo tú", "level": "success", "action_url": "/billing",
	})

	got := f.recipients(t, a.ID)
	if len(got) != 1 || got[f.doctorA2.ID] != f.clinicA.TenantID {
		t.Fatalf("recipients = %v, want only doctor A2", got)
	}
}

func TestCreateAnnouncement_UserOutsideCompanyIsRejected(t *testing.T) {
	f := setupAnnouncements(t)

	w, _ := f.create(t, map[string]any{"scope": "user", "company_id": f.clinicA.ID, "user_id": f.doctorB.ID, "title": "x", "body": "y", "level": "info"})

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestCreateAnnouncement_RejectsExternalNonHTTPSLink(t *testing.T) {
	f := setupAnnouncements(t)

	for _, url := range []string{"http://evil.test", "//evil.test", "javascript:alert(1)"} {
		w, _ := f.create(t, map[string]any{"scope": "global", "title": "x", "body": "y", "level": "info", "action_url": url})
		if w.Code != http.StatusBadRequest {
			t.Fatalf("action_url %q: status = %d, want 400", url, w.Code)
		}
	}
}

func TestScheduledAnnouncement_SentOnlyWhenDue(t *testing.T) {
	f := setupAnnouncements(t)
	scheduler := notifications_workers.NewAnnouncementScheduler(f.db, zap.NewNop())

	_, a := f.create(t, map[string]any{
		"scope": "company", "company_id": f.clinicB.ID, "title": "Pronto", "body": "Mañana",
		"level": "info", "scheduled_at": time.Now().Add(time.Hour),
	})
	if a.Status != notifications_models.AnnouncementStatusScheduled {
		t.Fatalf("status = %q, want scheduled", a.Status)
	}

	scheduler.SendDue()
	if got := f.recipients(t, a.ID); len(got) != 0 {
		t.Fatalf("sent before due: %v", got)
	}

	f.db.Model(&notifications_models.Announcement{}).Where("id = ?", a.ID).Update("scheduled_at", time.Now().Add(-time.Minute))
	scheduler.SendDue()
	if got := f.recipients(t, a.ID); len(got) != 1 || got[f.doctorB.ID] != f.clinicB.TenantID {
		t.Fatalf("recipients = %v, want doctor B", got)
	}
}

func TestCancelAnnouncement_OnlyWhileScheduled(t *testing.T) {
	f := setupAnnouncements(t)
	_, scheduled := f.create(t, map[string]any{
		"scope": "global", "title": "x", "body": "y", "level": "info", "scheduled_at": time.Now().Add(time.Hour),
	})
	_, sent := f.create(t, map[string]any{"scope": "global", "title": "x", "body": "y", "level": "info"})

	cancel := func(id uint) int {
		w := httptest.NewRecorder()
		f.router.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/backoffice/announcements/"+strconv.FormatUint(uint64(id), 10)+"/cancel", nil))
		return w.Code
	}

	if code := cancel(scheduled.ID); code != http.StatusOK {
		t.Fatalf("cancel scheduled: status = %d, want 200", code)
	}
	if code := cancel(sent.ID); code != http.StatusConflict {
		t.Fatalf("cancel sent: status = %d, want 409", code)
	}
}
