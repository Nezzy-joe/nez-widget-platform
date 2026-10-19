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

func TestSubmissionRepositoryListByTenantIsolationAndPagination(
	t *testing.T,
) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is required for PostgreSQL integration tests")
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		20*time.Second,
	)
	defer cancel()

	pool, err := NewPostgresPool(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect to PostgreSQL: %v", err)
	}
	defer pool.Close()

	tenantA := createIntegrationTenant(t, ctx, pool, "submission-list-a")
	tenantB := createIntegrationTenant(t, ctx, pool, "submission-list-b")

	widgetRepo := NewWidgetRepository(pool)
	submissionRepo := NewSubmissionRepository(pool)
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())

	createWidget := func(
		tenantID string,
		publicID string,
		title string,
	) *models.Widget {
		t.Helper()

		widget, err := widgetRepo.Create(ctx, tenantID, &models.Widget{
			PublicID:       publicID,
			Type:           "signup",
			Title:          title,
			ButtonText:     "Submit",
			DisplayOptions: json.RawMessage(`{}`),
		})
		if err != nil {
			t.Fatalf("create integration widget %q: %v", title, err)
		}
		return widget
	}

	widgetA1 := createWidget(
		tenantA, "list-a1-"+suffix, "Tenant A Widget One",
	)
	widgetA2 := createWidget(
		tenantA, "list-a2-"+suffix, "Tenant A Widget Two",
	)
	widgetB := createWidget(
		tenantB, "list-b-"+suffix, "Tenant B Widget",
	)

	createSubmission := func(
		tenantID string,
		widget *models.Widget,
		email string,
	) *models.Submission {
		t.Helper()

		submission, err := submissionRepo.Create(
			ctx,
			tenantID,
			&models.Submission{
				WidgetID: widget.ID,
				Data: json.RawMessage(
					fmt.Sprintf(`{"email":%q}`, email),
				),
			},
		)
		if err != nil {
			t.Fatalf("create integration submission: %v", err)
		}
		return submission
	}

	a1 := createSubmission(tenantA, widgetA1, "a1@example.com")
	a2 := createSubmission(tenantA, widgetA1, "a2@example.com")
	a3 := createSubmission(tenantA, widgetA2, "a3@example.com")
	createSubmission(tenantB, widgetB, "b1@example.com")

	// The first page must contain only Tenant A's records.
	pageOne, total, err := submissionRepo.ListByTenant(
		ctx, tenantA, "", 2, 0,
	)
	if err != nil {
		t.Fatalf("list first page: %v", err)
	}
	if total != 3 {
		t.Fatalf("expected 3 Tenant A submissions, got total %d", total)
	}
	if len(pageOne) != 2 {
		t.Fatalf("expected 2 submissions on first page, got %d", len(pageOne))
	}

	seen := make(map[string]bool)
	for _, submission := range pageOne {
		if submission.TenantID != tenantA {
			t.Fatalf("Tenant A received another tenant's submission: %q",
				submission.TenantID)
		}
		seen[submission.ID] = true
	}

	// The second page must contain the remaining record with no overlap.
	pageTwo, secondTotal, err := submissionRepo.ListByTenant(
		ctx, tenantA, "", 2, 2,
	)
	if err != nil {
		t.Fatalf("list second page: %v", err)
	}
	if secondTotal != 3 {
		t.Fatalf("expected total 3 on second page, got %d", secondTotal)
	}
	if len(pageTwo) != 1 {
		t.Fatalf("expected 1 submission on second page, got %d", len(pageTwo))
	}
	if pageTwo[0].TenantID != tenantA {
		t.Fatalf("second page exposed another tenant's record")
	}
	if seen[pageTwo[0].ID] {
		t.Fatal("the same submission appeared on both pages")
	}

	// Filtering by Tenant A's first widget must return only its records.
	filtered, filteredTotal, err := submissionRepo.ListByTenant(
		ctx, tenantA, widgetA1.ID, 100, 0,
	)
	if err != nil {
		t.Fatalf("list submissions for Tenant A widget: %v", err)
	}
	if filteredTotal != 2 || len(filtered) != 2 {
		t.Fatalf(
			"expected 2 submissions for Tenant A widget, got %d records and total %d",
			len(filtered), filteredTotal,
		)
	}
	for _, submission := range filtered {
		if submission.WidgetID != widgetA1.ID ||
			submission.TenantID != tenantA {
			t.Fatalf("widget filter returned an unexpected submission: %+v",
				submission)
		}
	}

	// Supplying Tenant B's widget ID must not expose Tenant B's data.
	crossTenant, crossTenantTotal, err := submissionRepo.ListByTenant(
		ctx, tenantA, widgetB.ID, 100, 0,
	)
	if err != nil {
		t.Fatalf("query with cross-tenant widget filter: %v", err)
	}
	if len(crossTenant) != 0 || crossTenantTotal != 0 {
		t.Fatalf(
			"cross-tenant widget filter exposed records: %d records, total %d",
			len(crossTenant), crossTenantTotal,
		)
	}

	// Confirm the test records we created were the expected ones.
	if a1.TenantID != tenantA ||
		a2.TenantID != tenantA ||
		a3.TenantID != tenantA {
		t.Fatal("test setup created submissions under an unexpected tenant")
	}

	t.Log("verified tenant isolation, pagination, and widget filtering")
}
