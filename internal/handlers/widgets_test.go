package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Nezzy-joe/nez-widget-platform/internal/models"
	"github.com/Nezzy-joe/nez-widget-platform/internal/services"
)

const testPublicID = "0123456789abcdef0123456789abcdef0123"

type widgetOperationsStub struct {
	publicWidget *models.Widget
	publicErr    error
	publicCalls  int
	lastPublicID string
}

func (s *widgetOperationsStub) Create(
	context.Context,
	string,
	*models.Widget,
) (*models.Widget, error) {
	return nil, errors.New("unexpected Create call")
}

func (s *widgetOperationsStub) List(
	context.Context,
	string,
) ([]models.Widget, error) {
	return nil, errors.New("unexpected List call")
}

func (s *widgetOperationsStub) Get(
	context.Context,
	string,
	string,
) (*models.Widget, error) {
	return nil, errors.New("unexpected Get call")
}

func (s *widgetOperationsStub) GetPublic(
	_ context.Context,
	publicID string,
) (*models.Widget, error) {
	s.publicCalls++
	s.lastPublicID = publicID
	return s.publicWidget, s.publicErr
}

func (s *widgetOperationsStub) Update(
	context.Context,
	string,
	string,
	*models.Widget,
) (*models.Widget, error) {
	return nil, errors.New("unexpected Update call")
}

func (s *widgetOperationsStub) Delete(
	context.Context,
	string,
	string,
) error {
	return errors.New("unexpected Delete call")
}

func TestPublicConfigReturnsSafeConfiguration(t *testing.T) {
	description := "Register for our technology programs."

	store := &widgetOperationsStub{
		publicWidget: &models.Widget{
			ID:             "internal-widget-id",
			TenantID:       "internal-tenant-id",
			PublicID:       testPublicID,
			Type:           "signup",
			Title:          "Engreg Registration",
			Description:    &description,
			ButtonText:     "Register Now",
			DisplayOptions: json.RawMessage(`{"theme":"light"}`),
			Fields: []models.WidgetField{
				{
					ID:        "internal-field-id",
					Name:      "email",
					Label:     "Email Address",
					FieldType: "email",
					Required:  true,
					Position:  0,
				},
			},
		},
	}

	handler := NewWidgetHandler(store)
	request := httptest.NewRequest(
		http.MethodGet,
		"/public/widgets/"+testPublicID,
		nil,
	)
	response := httptest.NewRecorder()

	handler.PublicConfig(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}

	if store.lastPublicID != testPublicID {
		t.Fatalf("unexpected public ID lookup: %q", store.lastPublicID)
	}

	if response.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatal("expected public configuration CORS header")
	}

	if response.Header().Get("Cache-Control") != "public, max-age=60" {
		t.Fatalf("unexpected Cache-Control header: %q",
			response.Header().Get("Cache-Control"))
	}

	var payload map[string]json.RawMessage
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	var widget map[string]json.RawMessage
	if err := json.Unmarshal(payload["widget"], &widget); err != nil {
		t.Fatalf("decode widget: %v", err)
	}

	for _, key := range []string{"id", "tenant_id", "created_at", "updated_at"} {
		if _, exists := widget[key]; exists {
			t.Errorf("public response exposed internal property %q", key)
		}
	}

	for _, key := range []string{"public_id", "type", "title", "button_text", "fields"} {
		if _, exists := widget[key]; !exists {
			t.Errorf("public response is missing %q", key)
		}
	}

	var fields []map[string]json.RawMessage
	if err := json.Unmarshal(widget["fields"], &fields); err != nil {
		t.Fatalf("decode fields: %v", err)
	}

	if len(fields) != 1 {
		t.Fatalf("expected one public field, got %d", len(fields))
	}

	if _, exists := fields[0]["id"]; exists {
		t.Error("public field response exposed its internal ID")
	}
}

func TestPublicConfigRejectsInvalidID(t *testing.T) {
	store := &widgetOperationsStub{}
	handler := NewWidgetHandler(store)

	request := httptest.NewRequest(
		http.MethodGet,
		"/public/widgets/not-a-valid-public-id",
		nil,
	)
	response := httptest.NewRecorder()

	handler.PublicConfig(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", response.Code)
	}

	if store.publicCalls != 0 {
		t.Fatal("invalid ID should not reach the service")
	}
}

func TestPublicConfigMapsMissingWidgetToNotFound(t *testing.T) {
	store := &widgetOperationsStub{
		publicErr: services.ErrWidgetNotFound,
	}
	handler := NewWidgetHandler(store)

	request := httptest.NewRequest(
		http.MethodGet,
		"/public/widgets/"+testPublicID,
		nil,
	)
	response := httptest.NewRecorder()

	handler.PublicConfig(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", response.Code, response.Body.String())
	}
}

func TestPublicConfigHandlesCORSPreflight(t *testing.T) {
	store := &widgetOperationsStub{}
	handler := NewWidgetHandler(store)

	request := httptest.NewRequest(
		http.MethodOptions,
		"/public/widgets/"+testPublicID,
		nil,
	)
	response := httptest.NewRecorder()

	handler.PublicConfig(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", response.Code)
	}

	if store.publicCalls != 0 {
		t.Fatal("preflight request should not retrieve a widget")
	}

	if response.Header().Get("Access-Control-Allow-Methods") != "GET, OPTIONS" {
		t.Fatal("unexpected CORS methods header")
	}
}
