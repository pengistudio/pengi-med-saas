package portal_workers_test

import (
	"sync"
	"testing"
	"time"

	"pengi-med-saas/core/audit"
	"pengi-med-saas/core/tenantdb"
	portal_models "pengi-med-saas/features/portal/models"
	portal_workers "pengi-med-saas/features/portal/workers"
	"pengi-med-saas/testutils"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type fakeNotifier struct {
	mu   sync.Mutex
	sent [][2]uint // tenant, request
}

func (n *fakeNotifier) SendRequestNotice(tenantID, requestID uint) string {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.sent = append(n.sent, [2]uint{tenantID, requestID})
	return "email"
}

var clock = time.Date(2026, 11, 14, 15, 0, 0, 0, time.UTC)

func setup(t *testing.T) *gorm.DB {
	t.Helper()
	return testutils.SetupTestDB(t, &portal_models.AppointmentRequest{}, &portal_models.PortalRefreshToken{}, &audit.AuditLog{})
}

// request stores a request of tenantID; its slot date is in the past, so
// releasing it never reaches the waitlist.
func request(t *testing.T, db *gorm.DB, tenantID uint, status string, expiresAt time.Time) portal_models.AppointmentRequest {
	t.Helper()
	r := portal_models.AppointmentRequest{
		TenantID: tenantID, PatientID: 1, DoctorID: 1, AppointmentTypeID: 1,
		Date: "2020-01-01", StartTime: "09:00", EndTime: "09:30", Status: status, ExpiresAt: expiresAt,
	}
	if err := tenantdb.System(db).Create(&r).Error; err != nil {
		t.Fatal(err)
	}
	return r
}

func status(t *testing.T, db *gorm.DB, id uint) string {
	t.Helper()
	var r portal_models.AppointmentRequest
	if err := tenantdb.System(db).First(&r, id).Error; err != nil {
		t.Fatal(err)
	}
	return r.Status
}

func TestTick_ExpiresDuePendingRequestsAcrossTenants(t *testing.T) {
	db := setup(t)
	dueA := request(t, db, 1, portal_models.RequestPending, clock.Add(-time.Minute))
	dueB := request(t, db, 2, portal_models.RequestPending, clock.Add(-time.Hour))
	edge := request(t, db, 2, portal_models.RequestPending, clock) // expires_at == now: due
	future := request(t, db, 1, portal_models.RequestPending, clock.Add(time.Minute))
	approved := request(t, db, 1, portal_models.RequestApproved, clock.Add(-time.Hour))
	rejected := request(t, db, 2, portal_models.RequestRejected, clock.Add(-time.Hour))

	n := &fakeNotifier{}
	worker := portal_workers.NewRequestExpiryScheduler(db, zap.NewNop(), n).WithClock(func() time.Time { return clock })
	if got := worker.Tick(); got != 3 {
		t.Fatalf("expired = %d, want 3", got)
	}
	for _, r := range []portal_models.AppointmentRequest{dueA, dueB, edge} {
		if s := status(t, db, r.ID); s != portal_models.RequestExpired {
			t.Fatalf("request %d status = %s, want expired", r.ID, s)
		}
	}
	for r, want := range map[uint]string{future.ID: portal_models.RequestPending, approved.ID: portal_models.RequestApproved, rejected.ID: portal_models.RequestRejected} {
		if s := status(t, db, r); s != want {
			t.Fatalf("request %d status = %s, want %s", r, s, want)
		}
	}
	// Each expired request is notified once, with its own tenant.
	want := map[uint]uint{dueA.ID: 1, dueB.ID: 2, edge.ID: 2}
	if len(n.sent) != 3 {
		t.Fatalf("notices = %v", n.sent)
	}
	for _, s := range n.sent {
		if want[s[1]] != s[0] {
			t.Fatalf("notice %v sent with the wrong tenant", s)
		}
	}
	// A second tick finds nothing and notifies nothing.
	if got := worker.Tick(); got != 0 || len(n.sent) != 3 {
		t.Fatalf("second tick expired %d, notices %d", got, len(n.sent))
	}
}

func TestTick_NilNotifier(t *testing.T) {
	db := setup(t)
	r := request(t, db, 1, portal_models.RequestPending, clock.Add(-time.Minute))
	worker := portal_workers.NewRequestExpiryScheduler(db, zap.NewNop(), nil).WithClock(func() time.Time { return clock })
	if got := worker.Tick(); got != 1 || status(t, db, r.ID) != portal_models.RequestExpired {
		t.Fatalf("expired = %d", got)
	}
}

func TestPurgeSessions_DeletesOnlyExpiredTokens(t *testing.T) {
	db := setup(t)
	sys := tenantdb.System(db)
	for i, exp := range []time.Time{clock.Add(-time.Hour), clock.Add(time.Hour)} {
		row := portal_models.PortalRefreshToken{TenantID: uint(i + 1), PatientID: 1, FamilyID: "fam", TokenHash: string(rune('a' + i)), ExpiresAt: exp}
		if err := sys.Create(&row).Error; err != nil {
			t.Fatal(err)
		}
	}
	portal_workers.NewRequestExpiryScheduler(db, zap.NewNop(), nil).WithClock(func() time.Time { return clock }).PurgeSessions()
	var rows []portal_models.PortalRefreshToken
	sys.Find(&rows)
	if len(rows) != 1 || rows[0].TenantID != 2 {
		t.Fatalf("remaining tokens = %+v", rows)
	}
}
