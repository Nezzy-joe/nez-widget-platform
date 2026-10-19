package services

import (
	"context"
	"errors"
	"testing"

	"github.com/Nezzy-joe/nez-widget-platform/internal/models"
)

type submissionAnalyticsReaderStub struct {
	calls    int
	tenantID string
	widgetID string
	result   *models.SubmissionAnalytics
	err      error
}

func (s *submissionAnalyticsReaderStub) GetAnalyticsByTenant(
	_ context.Context,
	tenantID string,
	widgetID string,
) (*models.SubmissionAnalytics, error) {
	s.calls++
	s.tenantID = tenantID
	s.widgetID = widgetID
	return s.result, s.err
}

func TestDashboardAnalyticsServicePassesTenantAndWidget(t *testing.T) {
	expected := &models.SubmissionAnalytics{
		TotalSubmissions: 8,
		Last7Days:        2,
		Last30Days:       5,
		Daily: []models.DailySubmissionCount{
			{Date: "2026-10-19", Count: 2},
		},
	}

	reader := &submissionAnalyticsReaderStub{result: expected}
	service := NewDashboardAnalyticsService(reader)

	actual, err := service.GetByTenant(
		context.Background(),
		" trusted-tenant ",
		" widget-id ",
	)
	if err != nil {
		t.Fatalf("GetByTenant returned an error: %v", err)
	}

	if actual != expected {
		t.Fatal("service did not return the repository result")
	}
	if reader.calls != 1 {
		t.Fatalf("expected one repository call, got %d", reader.calls)
	}
	if reader.tenantID != "trusted-tenant" {
		t.Fatalf("unexpected tenant ID: %q", reader.tenantID)
	}
	if reader.widgetID != "widget-id" {
		t.Fatalf("unexpected widget ID: %q", reader.widgetID)
	}
}

func TestDashboardAnalyticsServiceRejectsBlankTenant(t *testing.T) {
	reader := &submissionAnalyticsReaderStub{}
	service := NewDashboardAnalyticsService(reader)

	_, err := service.GetByTenant(
		context.Background(),
		"   ",
		"",
	)
	if !errors.Is(err, ErrTenantIDRequired) {
		t.Fatalf("expected ErrTenantIDRequired, got %v", err)
	}
	if reader.calls != 0 {
		t.Fatal("blank tenant ID reached the repository")
	}
}

func TestDashboardAnalyticsServicePropagatesRepositoryError(t *testing.T) {
	expectedErr := errors.New("database unavailable")
	reader := &submissionAnalyticsReaderStub{err: expectedErr}
	service := NewDashboardAnalyticsService(reader)

	_, err := service.GetByTenant(
		context.Background(),
		"trusted-tenant",
		"",
	)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected repository error to propagate, got %v", err)
	}
}
