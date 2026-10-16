package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/Nezzy-joe/nez-widget-platform/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestWidgetRepositoryTenantIsolation(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is required for PostgreSQL integration tests")
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		15*time.Second,
	)
	defer cancel()

	pool, err := NewPostgresPool(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect to PostgreSQL: %v", err)
	}
	defer pool.Close()

	tenantA := createIntegrationTenant(t, ctx, pool, "tenant-a")
	tenantB := createIntegrationTenant(t, ctx, pool, "tenant-b")

	repo := NewWidgetRepository(pool)
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())

	widgetA := &models.Widget{
		PublicID:       "integration-a-" + suffix,
		Type:           "signup",
		Title:          "Tenant A Signup",
		ButtonText:     "Join",
		DisplayOptions: json.RawMessage(`{}`),
		Fields: []models.WidgetField{
			{
				Name:      "email",
				Label:     "Email address",
				FieldType: "email",
				Required:  true,
				Position:  0,
			},
		},
	}

	createdA, err := repo.Create(ctx, tenantA, widgetA)
	if err != nil {
		t.Fatalf("create tenant A widget: %v", err)
	}

	widgetB := &models.Widget{
		PublicID:       "integration-b-" + suffix,
		Type:           "signup",
		Title:          "Tenant B Signup",
		ButtonText:     "Register",
		DisplayOptions: json.RawMessage(`{}`),
	}

	createdB, err := repo.Create(ctx, tenantB, widgetB)
	if err != nil {
		t.Fatalf("create tenant B widget: %v", err)
	}
	// A valid submission must persist under the correct tenant and widget.
	submissionRepo := NewSubmissionRepository(pool)

	createdSubmission, err := submissionRepo.Create(
		ctx,
		tenantA,
		&models.Submission{
			WidgetID:  createdA.ID,
			Data:      json.RawMessage(`{"email":"test@example.com"}`),
			IPAddress: "203.0.113.10",
			Country:   "Nigeria",
			City:      "Lagos",
		},
	)
	if err != nil {
		t.Fatalf("create submission: %v", err)
	}

	if createdSubmission.TenantID != tenantA {
		t.Fatalf("wrong submission tenant: %q", createdSubmission.TenantID)
	}

	if createdSubmission.WidgetID != createdA.ID {
		t.Fatalf("wrong submission widget: %q", createdSubmission.WidgetID)
	}

	if string(createdSubmission.Data) != `{"email": "test@example.com"}` &&
		string(createdSubmission.Data) != `{"email":"test@example.com"}` {
		t.Fatalf("unexpected submission data: %s", createdSubmission.Data)
	}

	if createdSubmission.Country != "Nigeria" ||
		createdSubmission.City != "Lagos" {
		t.Fatalf(
			"unexpected location data: country=%q city=%q",
			createdSubmission.Country,
			createdSubmission.City,
		)
	}

	// Tenant B must not submit against Tenant A's widget.
	_, err = submissionRepo.Create(
		ctx,
		tenantB,
		&models.Submission{
			WidgetID: createdA.ID,
			Data:     json.RawMessage(`{"email":"intruder@example.com"}`),
		},
	)
	if !errors.Is(err, ErrWidgetNotFound) {
		t.Fatalf(
			"expected ErrWidgetNotFound for cross-tenant submission, got %v",
			err,
		)
	}

	if len(createdA.Fields) != 1 {
		t.Fatalf("expected one saved field, got %d", len(createdA.Fields))
	}

	// A public ID retrieves the correct widget and its configured fields.
	publicWidget, err := repo.GetByPublicID(ctx, createdA.PublicID)
	if err != nil {
		t.Fatalf("get widget by public ID: %v", err)
	}

	if publicWidget.ID != createdA.ID {
		t.Fatalf(
			"public ID returned the wrong widget: got %q, want %q",
			publicWidget.ID,
			createdA.ID,
		)
	}

	if publicWidget.Title != "Tenant A Signup" {
		t.Fatalf("unexpected public widget title: %q", publicWidget.Title)
	}

	if len(publicWidget.Fields) != 1 {
		t.Fatalf(
			"expected one field from public lookup, got %d",
			len(publicWidget.Fields),
		)
	}

	// An unknown public ID must not return a widget.
	_, err = repo.GetByPublicID(ctx, "missing-public-id-"+suffix)
	if !errors.Is(err, ErrWidgetNotFound) {
		t.Fatalf(
			"expected ErrWidgetNotFound for unknown public ID, got %v",
			err,
		)
	}

	// Tenant A must not see tenant B's widget in its list.
	listA, err := repo.ListByTenant(ctx, tenantA)
	if err != nil {
		t.Fatalf("list tenant A widgets: %v", err)
	}

	for _, widget := range listA {
		if widget.ID == createdB.ID {
			t.Fatal("tenant A's widget list exposed tenant B's widget")
		}
	}

	// Tenant A can retrieve its own widget.
	ownWidget, err := repo.GetByID(ctx, tenantA, createdA.ID)
	if err != nil {
		t.Fatalf("tenant A could not retrieve its own widget: %v", err)
	}

	if ownWidget.Title != "Tenant A Signup" {
		t.Fatalf("unexpected widget title: %q", ownWidget.Title)
	}

	// Tenant B cannot retrieve tenant A's widget.
	_, err = repo.GetByID(ctx, tenantB, createdA.ID)
	if !errors.Is(err, ErrWidgetNotFound) {
		t.Fatalf("expected ErrWidgetNotFound for cross-tenant get, got %v", err)
	}

	// Tenant B cannot update tenant A's widget.
	_, err = repo.Update(ctx, tenantB, createdA.ID, &models.Widget{
		Type:           "signup",
		Title:          "Unauthorized Change",
		ButtonText:     "Change",
		DisplayOptions: json.RawMessage(`{}`),
	})
	if !errors.Is(err, ErrWidgetNotFound) {
		t.Fatalf("expected ErrWidgetNotFound for cross-tenant update, got %v", err)
	}

	// Tenant B cannot delete tenant A's widget.
	err = repo.Delete(ctx, tenantB, createdA.ID)
	if !errors.Is(err, ErrWidgetNotFound) {
		t.Fatalf("expected ErrWidgetNotFound for cross-tenant delete, got %v", err)
	}

	// Confirm the failed operations did not alter tenant A's widget.
	stillThere, err := repo.GetByID(ctx, tenantA, createdA.ID)
	if err != nil {
		t.Fatalf("tenant A's widget disappeared: %v", err)
	}

	if stillThere.Title != "Tenant A Signup" {
		t.Fatalf("widget changed after unauthorized update: %q", stillThere.Title)
	}

	t.Log("tenant isolation verified: cross-tenant get, update and delete rejected")
}

func createIntegrationTenant(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	label string,
) string {
	t.Helper()

	var tenantID string
	name := fmt.Sprintf(
		"integration-%s-%d",
		label,
		time.Now().UnixNano(),
	)

	err := pool.QueryRow(
		ctx,
		`INSERT INTO tenants (name) VALUES ($1) RETURNING id`,
		name,
	).Scan(&tenantID)
	if err != nil {
		t.Fatalf("create integration tenant: %v", err)
	}

	t.Cleanup(func() {
		// Cascading foreign keys remove the temporary test data.
		_, _ = pool.Exec(
			context.Background(),
			`DELETE FROM tenants WHERE id = $1`,
			tenantID,
		)
	})

	return tenantID
}
