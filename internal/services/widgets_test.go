package services

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"testing"

	"github.com/Nezzy-joe/nez-widget-platform/internal/models"
	"github.com/Nezzy-joe/nez-widget-platform/internal/repository"
)

type fakeWidgetStore struct {
	createFn func(context.Context, string, *models.Widget) (*models.Widget, error)
	listFn   func(context.Context, string) ([]models.Widget, error)
	getFn    func(context.Context, string, string) (*models.Widget, error)
	updateFn func(context.Context, string, string, *models.Widget) (*models.Widget, error)
	deleteFn func(context.Context, string, string) error
}

func (f *fakeWidgetStore) Create(
	ctx context.Context,
	tenantID string,
	widget *models.Widget,
) (*models.Widget, error) {
	if f.createFn == nil {
		return nil, errors.New("unexpected Create call")
	}
	return f.createFn(ctx, tenantID, widget)
}

func (f *fakeWidgetStore) ListByTenant(
	ctx context.Context,
	tenantID string,
) ([]models.Widget, error) {
	if f.listFn == nil {
		return nil, errors.New("unexpected ListByTenant call")
	}
	return f.listFn(ctx, tenantID)
}

func (f *fakeWidgetStore) GetByID(
	ctx context.Context,
	tenantID string,
	widgetID string,
) (*models.Widget, error) {
	if f.getFn == nil {
		return nil, errors.New("unexpected GetByID call")
	}
	return f.getFn(ctx, tenantID, widgetID)
}

func (f *fakeWidgetStore) Update(
	ctx context.Context,
	tenantID string,
	widgetID string,
	widget *models.Widget,
) (*models.Widget, error) {
	if f.updateFn == nil {
		return nil, errors.New("unexpected Update call")
	}
	return f.updateFn(ctx, tenantID, widgetID, widget)
}

func (f *fakeWidgetStore) Delete(
	ctx context.Context,
	tenantID string,
	widgetID string,
) error {
	if f.deleteFn == nil {
		return errors.New("unexpected Delete call")
	}
	return f.deleteFn(ctx, tenantID, widgetID)
}

func TestWidgetServiceCreateNormalizesInputAndGeneratesPublicID(t *testing.T) {
	var createCalls int

	store := &fakeWidgetStore{
		createFn: func(
			_ context.Context,
			tenantID string,
			widget *models.Widget,
		) (*models.Widget, error) {
			createCalls++

			if tenantID != "verified-tenant" {
				t.Fatalf("unexpected tenant ID: %q", tenantID)
			}
			if widget.TenantID != tenantID {
				t.Fatalf("tenant was not set by the service: %q", widget.TenantID)
			}
			if widget.ID != "" {
				t.Fatal("client-provided internal ID was not cleared")
			}
			if len(widget.PublicID) != 36 {
				t.Fatalf("expected 36-character public ID, got %q", widget.PublicID)
			}
			if _, err := hex.DecodeString(widget.PublicID); err != nil {
				t.Fatalf("public ID must be hexadecimal: %v", err)
			}
			if widget.Title != "Newsletter Signup" {
				t.Fatalf("title wasn't normalized: %q", widget.Title)
			}
			if widget.ButtonText != "Submit" {
				t.Fatalf("expected default button text, got %q", widget.ButtonText)
			}
			if string(widget.DisplayOptions) != "{}" {
				t.Fatalf("expected empty display options, got %s", widget.DisplayOptions)
			}
			if len(widget.Fields) != 1 || widget.Fields[0].Name != "email" {
				t.Fatalf("field name wasn't normalized: %+v", widget.Fields)
			}
			if widget.Fields[0].Position != 0 {
				t.Fatalf("expected first field position 0, got %d", widget.Fields[0].Position)
			}

			result := *widget
			result.ID = "database-widget-id"
			result.TenantID = tenantID
			return &result, nil
		},
	}

	service := NewWidgetService(store)

	widget, err := service.Create(
		context.Background(),
		"verified-tenant",
		&models.Widget{
			ID:             "client-chosen-id",
			TenantID:       "other-tenant",
			PublicID:       "client-chosen-public-id",
			Type:           "SIGNUP",
			Title:          "  Newsletter Signup  ",
			ButtonText:     "  ",
			DisplayOptions: nil,
			Fields: []models.WidgetField{
				{
					Name:      " Email ",
					Label:     "Email Address",
					FieldType: "EMAIL",
					Required:  true,
				},
			},
		},
	)
	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	if createCalls != 1 {
		t.Fatalf("expected one repository call, got %d", createCalls)
	}
	if widget.ID != "database-widget-id" {
		t.Fatalf("unexpected returned widget ID: %q", widget.ID)
	}
}

func TestWidgetServiceRejectsInvalidWidgetsBeforePersistence(t *testing.T) {
	validField := models.WidgetField{
		Name:      "email",
		Label:     "Email Address",
		FieldType: "email",
		Required:  true,
	}

	tests := []struct {
		name   string
		widget *models.Widget
	}{
		{
			name: "unsupported widget type",
			widget: &models.Widget{
				Type:  "unknown",
				Title: "Example",
			},
		},
		{
			name: "empty title",
			widget: &models.Widget{
				Type:  "cta",
				Title: "   ",
			},
		},
		{
			name: "signup without fields",
			widget: &models.Widget{
				Type:  "signup",
				Title: "Newsletter",
			},
		},
		{
			name: "invalid field name",
			widget: &models.Widget{
				Type:  "signup",
				Title: "Newsletter",
				Fields: []models.WidgetField{
					{
						Name:      "Email Address",
						Label:     "Email",
						FieldType: "email",
					},
				},
			},
		},
		{
			name: "duplicate field names",
			widget: &models.Widget{
				Type:  "signup",
				Title: "Newsletter",
				Fields: []models.WidgetField{
					validField,
					validField,
				},
			},
		},
		{
			name: "invalid display options",
			widget: &models.Widget{
				Type:           "cta",
				Title:          "Get Started",
				DisplayOptions: json.RawMessage(`[]`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			store := &fakeWidgetStore{
				createFn: func(
					context.Context,
					string,
					*models.Widget,
				) (*models.Widget, error) {
					called = true
					return nil, nil
				},
			}

			service := NewWidgetService(store)

			_, err := service.Create(
				context.Background(),
				"verified-tenant",
				tt.widget,
			)

			if !errors.Is(err, ErrInvalidWidget) {
				t.Fatalf("expected ErrInvalidWidget, got %v", err)
			}
			if called {
				t.Fatal("repository was called for an invalid widget")
			}
		})
	}
}

func TestWidgetServiceRequiresTenantID(t *testing.T) {
	service := NewWidgetService(&fakeWidgetStore{})

	_, err := service.List(context.Background(), "  ")
	if !errors.Is(err, ErrTenantIDRequired) {
		t.Fatalf("expected ErrTenantIDRequired, got %v", err)
	}
}

func TestWidgetServiceTranslatesWidgetNotFound(t *testing.T) {
	store := &fakeWidgetStore{
		getFn: func(
			context.Context,
			string,
			string,
		) (*models.Widget, error) {
			return nil, repository.ErrWidgetNotFound
		},
	}

	service := NewWidgetService(store)

	_, err := service.Get(
		context.Background(),
		"verified-tenant",
		"missing-widget",
	)
	if !errors.Is(err, ErrWidgetNotFound) {
		t.Fatalf("expected service ErrWidgetNotFound, got %v", err)
	}
}
