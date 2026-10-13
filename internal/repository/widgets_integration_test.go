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

	if len(createdA.Fields) != 1 {
		t.Fatalf("expected one saved field, got %d", len(createdA.Fields))
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
