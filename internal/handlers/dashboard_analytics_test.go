package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Nezzy-joe/nez-widget-platform/internal/middleware"
	"github.com/Nezzy-joe/nez-widget-platform/internal/models"
	"github.com/Nezzy-joe/nez-widget-platform/internal/services"
)

const dashboardAnalyticsTestSecret = "dashboard-analytics-test-secret-at-least-32-bytes"

type dashboardAnalyticsGetterStub struct {
	calls    int
	tenantID string
	widgetID string
	result   *models.SubmissionAnalytics
	err      error
}

func (s *dashboardAnalyticsGetterStub) GetByTenant(
	_ context.Context,
	tenantID string,
	widgetID string,
) (*models.SubmissionAnalytics, error) {
	s.calls++
	s.tenantID = tenantID
	s.widgetID = widgetID
	return s.result, s.err
}

func protectAnalyticsHandler(
	t *testing.T,
	handler *DashboardAnalyticsHandler,
) (http.Handler, string) {
	t.Helper()

	jwtService, err := services.NewJWTService(dashboardAnalyticsTestSecret)
	if err != nil {
		t.Fatalf("create JWT service: %v", err)
	}

	token, err := jwtService.Generate(&models.User{
		ID:       "analytics-test-user",
		TenantID: "analytics-trusted-tenant",
	})
	if err != nil {
		t.Fatalf("generate JWT: %v", err)
	}

	protected := middleware.RequireAuth(jwtService)(
		http.HandlerFunc(handler.Get),
	)

	return protected, token
}

func TestDashboardAnalyticsReturnsTenantMetrics(t *testing.T) {
	expected := &models.SubmissionAnalytics{
		TotalSubmissions: 12,
		Last7Days:        4,
		Last30Days:       9,
		Daily: []models.DailySubmissionCount{
			{Date: "2026-10-13", Count: 1},
			{Date: "2026-10-14", Count: 0},
			{Date: "2026-10-15", Count: 1},
			{Date: "2026-10-16", Count: 0},
			{Date: "2026-10-17", Count: 0},
			{Date: "2026-10-18", Count: 0},
			{Date: "2026-10-19", Count: 2},
		},
	}

	store := &dashboardAnalyticsGetterStub{result: expected}
	handler := NewDashboardAnalyticsHandler(store)
	protected, token := protectAnalyticsHandler(t, handler)

	widgetID := dashboardTestWidgetID
	req := httptest.NewRequest(
		http.MethodGet,
		"/dashboard/analytics?widget_id="+widgetID+
			"&tenant_id=attacker-tenant",
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
	if store.tenantID != "analytics-trusted-tenant" {
		t.Fatalf("untrusted tenant used: %q", store.tenantID)
	}
	if store.widgetID != widgetID {
		t.Fatalf("unexpected widget filter: %q", store.widgetID)
	}

	if rec.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("unexpected cache policy: %q", rec.Header().Get("Cache-Control"))
	}

	var actual models.SubmissionAnalytics
	if err := json.Unmarshal(rec.Body.Bytes(), &actual); err != nil {
		t.Fatalf("decode analytics response: %v", err)
	}
	if actual.TotalSubmissions != expected.TotalSubmissions ||
		actual.Last7Days != expected.Last7Days ||
		actual.Last30Days != expected.Last30Days ||
		len(actual.Daily) != 7 {
		t.Fatalf("unexpected analytics response: %+v", actual)
	}
}

func TestDashboardAnalyticsRequiresAuthentication(t *testing.T) {
	store := &dashboardAnalyticsGetterStub{}
	handler := NewDashboardAnalyticsHandler(store)
	protected, _ := protectAnalyticsHandler(t, handler)

	req := httptest.NewRequest(http.MethodGet, "/dashboard/analytics", nil)
	rec := httptest.NewRecorder()

	protected.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
	if store.calls != 0 {
		t.Fatal("unauthenticated request reached the service")
	}
}

func TestDashboardAnalyticsRejectsInvalidWidgetID(t *testing.T) {
	store := &dashboardAnalyticsGetterStub{}
	handler := NewDashboardAnalyticsHandler(store)
	protected, token := protectAnalyticsHandler(t, handler)

	req := httptest.NewRequest(
		http.MethodGet,
		"/dashboard/analytics?widget_id=not-a-uuid",
		nil,
	)
	req.Header.Set("Authorization", "Bearer "+token)

	rec := httptest.NewRecorder()
	protected.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	if store.calls != 0 {
		t.Fatal("invalid widget ID reached the service")
	}
}

func TestDashboardAnalyticsRejectsWrongMethod(t *testing.T) {
	store := &dashboardAnalyticsGetterStub{}
	handler := NewDashboardAnalyticsHandler(store)
	protected, token := protectAnalyticsHandler(t, handler)

	req := httptest.NewRequest(http.MethodPost, "/dashboard/analytics", nil)
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

func TestDashboardAnalyticsDoesNotExposeInternalErrors(t *testing.T) {
	store := &dashboardAnalyticsGetterStub{
		err: errors.New("private database details"),
	}
	handler := NewDashboardAnalyticsHandler(store)
	protected, token := protectAnalyticsHandler(t, handler)

	req := httptest.NewRequest(http.MethodGet, "/dashboard/analytics", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	rec := httptest.NewRecorder()
	protected.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "private database details") {
		t.Fatal("response exposed internal database details")
	}
}
