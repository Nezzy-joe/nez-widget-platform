package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/Nezzy-joe/nez-widget-platform/internal/models"
)

func TestSubmissionAnalyticsTenantIsolationAndWindows(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is required for PostgreSQL integration tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	pool, err := NewPostgresPool(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect to PostgreSQL: %v", err)
	}
	defer pool.Close()

	tenantA := createIntegrationTenant(t, ctx, pool, "analytics-a")
	tenantB := createIntegrationTenant(t, ctx, pool, "analytics-b")
	widgetRepo := NewWidgetRepository(pool)
	submissionRepo := NewSubmissionRepository(pool)
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())

	createWidget := func(tenantID, label string) *models.Widget {
		t.Helper()

		widget, err := widgetRepo.Create(ctx, tenantID, &models.Widget{
			PublicID:       "analytics-" + label + "-" + suffix,
			Type:           "signup",
			Title:          "Analytics " + label,
			ButtonText:     "Submit",
			DisplayOptions: json.RawMessage(`{}`),
		})
		if err != nil {
			t.Fatalf("create widget %s: %v", label, err)
		}
		return widget
	}

	widgetA1 := createWidget(tenantA, "a1")
	widgetA2 := createWidget(tenantA, "a2")
	widgetB := createWidget(tenantB, "b")

	createSubmission := func(
		tenantID string,
		widget *models.Widget,
		email string,
	) *models.Submission {
		t.Helper()

		submission, err := submissionRepo.Create(ctx, tenantID, &models.Submission{
			WidgetID: widget.ID,
			Data:     json.RawMessage(fmt.Sprintf(`{"email":%q}`, email)),
		})
		if err != nil {
			t.Fatalf("create submission: %v", err)
		}
		return submission
	}

	createSubmission(tenantA, widgetA1, "a1@example.com")
	createSubmission(tenantA, widgetA1, "a2@example.com")
	future := createSubmission(tenantA, widgetA2, "future@example.com")
	createSubmission(tenantB, widgetB, "b1@example.com")

	// Keep this record in the lifetime total, but outside today's calendar window.
	_, err = pool.Exec(
		ctx,
		`UPDATE submissions
		 SET created_at = CURRENT_DATE + INTERVAL '2 days'
		 WHERE id = $1`,
		future.ID,
	)
	if err != nil {
		t.Fatalf("make test record future-dated: %v", err)
	}

	analytics, err := submissionRepo.GetAnalyticsByTenant(ctx, tenantA, "")
	if err != nil {
		t.Fatalf("get tenant analytics: %v", err)
	}

	if analytics.TotalSubmissions != 3 {
		t.Errorf("total: got %d, want 3", analytics.TotalSubmissions)
	}
	if analytics.Last7Days != 2 {
		t.Errorf("last 7 days: got %d, want 2", analytics.Last7Days)
	}
	if analytics.Last30Days != 2 {
		t.Errorf("last 30 days: got %d, want 2", analytics.Last30Days)
	}
	if len(analytics.Daily) != 7 {
		t.Fatalf("daily buckets: got %d, want 7", len(analytics.Daily))
	}

	dailySum := 0
	var previousDate time.Time
	for i, day := range analytics.Daily {
		parsed, err := time.Parse("2006-01-02", day.Date)
		if err != nil {
			t.Fatalf("invalid daily date %q: %v", day.Date, err)
		}
		if i > 0 && !parsed.Equal(previousDate.AddDate(0, 0, 1)) {
			t.Fatalf("daily dates are not consecutive: %q", day.Date)
		}
		previousDate = parsed
		dailySum += day.Count
	}
	if dailySum != 2 {
		t.Errorf("sum of daily buckets: got %d, want 2", dailySum)
	}
	if analytics.Daily[6].Count != 2 {
		t.Errorf("today's count: got %d, want 2", analytics.Daily[6].Count)
	}

	// A widget filter narrows the results without removing tenant scoping.
	filtered, err := submissionRepo.GetAnalyticsByTenant(ctx, tenantA, widgetA1.ID)
	if err != nil {
		t.Fatalf("get filtered analytics: %v", err)
	}
	if filtered.TotalSubmissions != 2 ||
		filtered.Last7Days != 2 ||
		filtered.Last30Days != 2 {
		t.Errorf("unexpected analytics for Tenant A's first widget: %+v", filtered)
	}

	futureOnly, err := submissionRepo.GetAnalyticsByTenant(ctx, tenantA, widgetA2.ID)
	if err != nil {
		t.Fatalf("get future-only widget analytics: %v", err)
	}
	if futureOnly.TotalSubmissions != 1 ||
		futureOnly.Last7Days != 0 ||
		futureOnly.Last30Days != 0 {
		t.Errorf("unexpected future-only analytics: %+v", futureOnly)
	}

	crossTenant, err := submissionRepo.GetAnalyticsByTenant(ctx, tenantA, widgetB.ID)
	if err != nil {
		t.Fatalf("query with other tenant's widget: %v", err)
	}
	if crossTenant.TotalSubmissions != 0 ||
		crossTenant.Last7Days != 0 ||
		crossTenant.Last30Days != 0 {
		t.Errorf("cross-tenant widget filter returned data: %+v", crossTenant)
	}

	if len(crossTenant.Daily) != 7 {
		t.Fatalf("cross-tenant daily buckets: got %d, want 7", len(crossTenant.Daily))
	}
	for _, day := range crossTenant.Daily {
		if day.Count != 0 {
			t.Errorf("cross-tenant activity leaked on %s: %d", day.Date, day.Count)
		}
	}

	t.Log("verified analytics totals, date windows, daily buckets, and tenant isolation")
}
