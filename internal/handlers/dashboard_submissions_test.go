package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Nezzy-joe/nez-widget-platform/internal/middleware"
	"github.com/Nezzy-joe/nez-widget-platform/internal/models"
	"github.com/Nezzy-joe/nez-widget-platform/internal/services"
)

const (
	dashboardTestSecret   = "dashboard-handler-test-secret-at-least-32-bytes"
	dashboardTestTenantID = "trusted-tenant-id"
	dashboardTestWidgetID = "22222222-2222-4222-8222-222222222222"
)

type dashboardSubmissionListerStub struct {
	calls       int
	tenantID    string
	widgetID    string
	limit       int
	offset      int
	submissions []models.Submission
	total       int
	err         error
}

func (s *dashboardSubmissionListerStub) ListByTenant(
	_ context.Context,
	tenantID string,
	widgetID string,
	limit int,
	offset int,
) ([]models.Submission, int, error) {
	s.calls++
	s.tenantID = tenantID
	s.widgetID = widgetID
	s.limit = limit
	s.offset = offset
	return s.submissions, s.total, s.err
}

func newDashboardTestAuth(
	t *testing.T,
	handler *DashboardSubmissionHandler,
) (http.Handler, string) {
	t.Helper()

	jwtService, err := services.NewJWTService(dashboardTestSecret)
	if err != nil {
		t.Fatalf("create JWT service: %v", err)
	}

	token, err := jwtService.Generate(&models.User{
		ID:       "dashboard-test-user",
		TenantID: dashboardTestTenantID,
	})
	if err != nil {
		t.Fatalf("generate JWT: %v", err)
	}

	protected := middleware.RequireAuth(jwtService)(
		http.HandlerFunc(handler.List),
	)

	return protected, token
}

func TestDashboardSubmissionListUsesAuthenticatedTenant(t *testing.T) {
	store := &dashboardSubmissionListerStub{
		submissions: []models.Submission{
			{
				ID:       "submission-1",
				WidgetID: dashboardTestWidgetID,
				Data:     json.RawMessage(`{"email":"visitor@example.com"}`),
			},
		},
		total: 1,
	}
	handler := NewDashboardSubmissionHandler(store)
	protected, token := newDashboardTestAuth(t, handler)

	req := httptest.NewRequest(
		http.MethodGet,
		"/dashboard/submissions?limit=10&offset=5&widget_id="+
			dashboardTestWidgetID+"&tenant_id=attacker-tenant",
		nil,
	)
	req.Header.Set("Authorization", "Bearer "+token)

	rec := httptest.NewRecorder()
	protected.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	if store.calls != 1 {
		t.Fatalf("expected one service call, got %d", store.calls)
	}
	if store.tenantID != dashboardTestTenantID {
		t.Fatalf("untrusted tenant used: %q", store.tenantID)
	}
	if store.widgetID != dashboardTestWidgetID {
		t.Fatalf("unexpected widget filter: %q", store.widgetID)
	}
	if store.limit != 10 || store.offset != 5 {
		t.Fatalf("unexpected pagination: limit=%d offset=%d",
			store.limit, store.offset)
	}

	var payload struct {
		Submissions []models.Submission  `json:"submissions"`
		Pagination  submissionPagination `json:"pagination"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(payload.Submissions) != 1 || payload.Pagination.Total != 1 {
		t.Fatalf("unexpected response: %+v", payload)
	}
}

func TestDashboardSubmissionListRequiresAuthentication(t *testing.T) {
	store := &dashboardSubmissionListerStub{}
	handler := NewDashboardSubmissionHandler(store)
	protected, _ := newDashboardTestAuth(t, handler)

	req := httptest.NewRequest(
		http.MethodGet,
		"/dashboard/submissions",
		nil,
	)
	rec := httptest.NewRecorder()

	protected.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
	if store.calls != 0 {
		t.Fatal("unauthenticated request reached the service")
	}
}

func TestDashboardSubmissionListRejectsInvalidQuery(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{"zero limit", "/dashboard/submissions?limit=0"},
		{"limit too large", "/dashboard/submissions?limit=101"},
		{"non-numeric limit", "/dashboard/submissions?limit=abc"},
		{"negative offset", "/dashboard/submissions?offset=-1"},
		{"non-numeric offset", "/dashboard/submissions?offset=abc"},
		{"offset too large", "/dashboard/submissions?offset=1000001"},
		{"invalid widget ID", "/dashboard/submissions?widget_id=not-a-uuid"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &dashboardSubmissionListerStub{}
			handler := NewDashboardSubmissionHandler(store)
			protected, token := newDashboardTestAuth(t, handler)

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			req.Header.Set("Authorization", "Bearer "+token)
			rec := httptest.NewRecorder()

			protected.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d: %s",
					rec.Code, rec.Body.String())
			}
			if store.calls != 0 {
				t.Fatal("invalid query reached the service")
			}
		})
	}
}

func TestDashboardSubmissionListRejectsWrongMethod(t *testing.T) {
	store := &dashboardSubmissionListerStub{}
	handler := NewDashboardSubmissionHandler(store)
	protected, token := newDashboardTestAuth(t, handler)

	req := httptest.NewRequest(
		http.MethodPost,
		"/dashboard/submissions",
		nil,
	)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	protected.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
	if rec.Header().Get("Allow") != http.MethodGet {
		t.Fatalf("unexpected Allow header: %q", rec.Header().Get("Allow"))
	}
	if store.calls != 0 {
		t.Fatal("wrong method reached the service")
	}
}

func TestDashboardSubmissionListHidesInternalErrors(t *testing.T) {
	store := &dashboardSubmissionListerStub{
		err: errors.New("internal database details"),
	}
	handler := NewDashboardSubmissionHandler(store)
	protected, token := newDashboardTestAuth(t, handler)

	req := httptest.NewRequest(
		http.MethodGet,
		"/dashboard/submissions",
		nil,
	)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	protected.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
	if rec.Body.String() == "" ||
		containsText(rec.Body.String(), "internal database details") {
		t.Fatal("response should not expose internal database errors")
	}
}

func containsText(value, part string) bool {
	for i := 0; i+len(part) <= len(value); i++ {
		if value[i:i+len(part)] == part {
			return true
		}
	}
	return false
}
