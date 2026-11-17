package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	backoffice_models "pengi-med-saas/features/backoffice/models"

	"gorm.io/gorm"
)

func auditEntries(t *testing.T, db *gorm.DB) []backoffice_models.BackofficeAuditLog {
	t.Helper()
	var entries []backoffice_models.BackofficeAuditLog
	if err := db.Order("id").Find(&entries).Error; err != nil {
		t.Fatal(err)
	}
	return entries
}

// The audit trail records the sign-in, every change a signed-in admin makes
// (also the ones the handler rejects) and the sign-out; it leaves out reads
// and requests nobody is signed in for.
func TestBackofficeAudit_RecordsSessionsAndChanges(t *testing.T) {
	router, db := backofficeRouter(t)
	createBackofficeAdmin(t, db)

	// Nobody signed in: a wrong password and a change without a token.
	bad := httptest.NewRequest(http.MethodPost, "/backoffice/auth/login", strings.NewReader(`{"user_name":"admin","password":"wrong"}`))
	bad.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(httptest.NewRecorder(), bad)
	call(router, http.MethodDelete, "/backoffice/plans/1", "")
	if entries := auditEntries(t, db); len(entries) != 0 {
		t.Fatalf("requests without a known admin left %d audit entries, want 0: %+v", len(entries), entries)
	}

	token, cookies := backofficeLogin(t, router)
	var admin backoffice_models.BackofficeUser
	if err := db.Where("user_name = ?", "admin").First(&admin).Error; err != nil {
		t.Fatal(err)
	}
	adminPath := "/backoffice/users/" + strconv.FormatUint(uint64(admin.ID), 10)

	if w := call(router, http.MethodGet, "/backoffice/users", token); w.Code != http.StatusOK {
		t.Fatalf("GET /backoffice/users = %d", w.Code)
	}
	rename := httptest.NewRequest(http.MethodPut, adminPath, strings.NewReader(`{"name":"Admin Renamed"}`))
	rename.Header.Set("Content-Type", "application/json")
	rename.Header.Set("Authorization", "Bearer "+token)
	renamed := httptest.NewRecorder()
	router.ServeHTTP(renamed, rename)
	if renamed.Code != http.StatusOK {
		t.Fatalf("rename = %d: %s", renamed.Code, renamed.Body.String())
	}
	if w := call(router, http.MethodDelete, "/backoffice/plans/999", token); w.Code != http.StatusNotFound {
		t.Fatalf("DELETE of a missing plan = %d, want 404", w.Code)
	}
	if w := call(router, http.MethodPost, "/backoffice/auth/logout", "", cookies...); w.Code != http.StatusOK {
		t.Fatalf("logout = %d", w.Code)
	}

	want := []struct {
		method, route, path string
		status              int
	}{
		{http.MethodPost, "/backoffice/auth/enroll/totp/finish", "/backoffice/auth/enroll/totp/finish", http.StatusOK},
		{http.MethodPut, "/backoffice/users/:id", adminPath, http.StatusOK},
		{http.MethodDelete, "/backoffice/plans/:id", "/backoffice/plans/999", http.StatusNotFound},
		{http.MethodPost, "/backoffice/auth/logout", "/backoffice/auth/logout", http.StatusOK},
	}
	entries := auditEntries(t, db)
	if len(entries) != len(want) {
		t.Fatalf("got %d audit entries, want %d: %+v", len(entries), len(want), entries)
	}
	for i, w := range want {
		e := entries[i]
		if e.Method != w.method || e.Route != w.route || e.Path != w.path || e.Status != w.status {
			t.Errorf("entry %d = %s %s (%s) %d, want %s %s (%s) %d", i, e.Method, e.Path, e.Route, e.Status, w.method, w.path, w.route, w.status)
		}
		if e.BackofficeUserID != admin.ID || e.UserName != "admin" {
			t.Errorf("entry %d attributed to #%d %q, want #%d \"admin\"", i, e.BackofficeUserID, e.UserName, admin.ID)
		}
		if e.CreatedAt.IsZero() {
			t.Errorf("entry %d has no date", i)
		}
	}
}

// GET /backoffice/audit-logs needs a session and returns the trail newest
// first, a page at a time (before_id continues it), optionally for one admin.
func TestBackofficeAudit_List(t *testing.T) {
	router, db := backofficeRouter(t)
	createBackofficeAdmin(t, db)

	if w := call(router, http.MethodGet, "/backoffice/audit-logs", ""); w.Code != http.StatusUnauthorized {
		t.Fatalf("GET /backoffice/audit-logs without a token = %d, want 401", w.Code)
	}

	token, _ := backofficeLogin(t, router)
	if err := db.Where("1 = 1").Delete(&backoffice_models.BackofficeAuditLog{}).Error; err != nil {
		t.Fatal(err)
	}
	// 100 entries is one page: 102 of admin #7 and one of admin #8.
	rows := make([]backoffice_models.BackofficeAuditLog, 0, 103)
	for i := 0; i < 102; i++ {
		rows = append(rows, backoffice_models.BackofficeAuditLog{BackofficeUserID: 7, UserName: "ana", Method: http.MethodPost, Path: "/backoffice/plans", Status: http.StatusOK})
	}
	rows = append(rows, backoffice_models.BackofficeAuditLog{BackofficeUserID: 8, UserName: "luis", Method: http.MethodDelete, Path: "/backoffice/plans/1", Status: http.StatusOK})
	if err := db.Create(&rows).Error; err != nil {
		t.Fatal(err)
	}

	list := func(query string) (ids []uint, hasMore bool) {
		t.Helper()
		w := call(router, http.MethodGet, "/backoffice/audit-logs"+query, token)
		if w.Code != http.StatusOK {
			t.Fatalf("GET /backoffice/audit-logs%s = %d: %s", query, w.Code, w.Body.String())
		}
		var body struct {
			Data struct {
				Items []struct {
					ID uint `json:"id"`
				} `json:"items"`
				HasMore bool `json:"has_more"`
			} `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		for _, item := range body.Data.Items {
			ids = append(ids, item.ID)
		}
		return ids, body.Data.HasMore
	}

	first, more := list("")
	if len(first) != 100 || !more {
		t.Fatalf("first page: %d entries, has_more=%v; want 100, true", len(first), more)
	}
	if first[0] != rows[102].ID || first[0] < first[99] {
		t.Fatalf("first page must start at the newest entry (#%d), got #%d … #%d", rows[102].ID, first[0], first[99])
	}
	rest, more := list("?before_id=" + strconv.FormatUint(uint64(first[99]), 10))
	if len(rest) != 3 || more {
		t.Fatalf("second page: %d entries, has_more=%v; want 3, false", len(rest), more)
	}
	if rest[2] != rows[0].ID {
		t.Fatalf("second page must end at the oldest entry (#%d), got #%d", rows[0].ID, rest[2])
	}

	if of8, more := list("?user_id=8"); len(of8) != 1 || of8[0] != rows[102].ID || more {
		t.Fatalf("?user_id=8: got %v (has_more=%v), want only #%d", of8, more, rows[102].ID)
	}
	if w := call(router, http.MethodGet, "/backoffice/audit-logs?before_id=abc", token); w.Code != http.StatusBadRequest {
		t.Fatalf("?before_id=abc = %d, want 400", w.Code)
	}
}
