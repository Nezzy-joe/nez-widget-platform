package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/Nezzy-joe/nez-widget-platform/internal/middleware"
	"github.com/Nezzy-joe/nez-widget-platform/internal/models"
	"github.com/Nezzy-joe/nez-widget-platform/internal/services"
)

var widgetIDPattern = regexp.MustCompile(
	`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`,
)

type widgetOperations interface {
	Create(context.Context, string, *models.Widget) (*models.Widget, error)
	List(context.Context, string) ([]models.Widget, error)
	Get(context.Context, string, string) (*models.Widget, error)
	Update(context.Context, string, string, *models.Widget) (*models.Widget, error)
	Delete(context.Context, string, string) error
}

type WidgetHandler struct {
	widgets widgetOperations
}

func NewWidgetHandler(widgets widgetOperations) *WidgetHandler {
	return &WidgetHandler{widgets: widgets}
}

type widgetRequest struct {
	Type           string               `json:"type"`
	Title          string               `json:"title"`
	Description    *string              `json:"description"`
	ButtonText     string               `json:"button_text"`
	DisplayOptions json.RawMessage      `json:"display_options"`
	Fields         []widgetFieldRequest `json:"fields"`
}

type widgetFieldRequest struct {
	Name      string `json:"name"`
	Label     string `json:"label"`
	FieldType string `json:"field_type"`
	Required  bool   `json:"required"`
}

func (p widgetRequest) toModel() *models.Widget {
	fields := make([]models.WidgetField, 0, len(p.Fields))

	for _, field := range p.Fields {
		fields = append(fields, models.WidgetField{
			Name:      field.Name,
			Label:     field.Label,
			FieldType: field.FieldType,
			Required:  field.Required,
		})
	}

	return &models.Widget{
		Type:           p.Type,
		Title:          p.Title,
		Description:    p.Description,
		ButtonText:     p.ButtonText,
		DisplayOptions: p.DisplayOptions,
		Fields:         fields,
	}
}

func (h *WidgetHandler) Collection(w http.ResponseWriter, r *http.Request) {
	identity, ok := middleware.IdentityFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	switch r.Method {
	case http.MethodGet:
		widgets, err := h.widgets.List(r.Context(), identity.TenantID)
		if err != nil {
			writeWidgetError(w, "list", err)
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"widgets": widgets})

	case http.MethodPost:
		var input widgetRequest
		if err := decodeJSON(w, r, &input); err != nil {
			writeDecodeError(w, err)
			return
		}

		widget, err := h.widgets.Create(
			r.Context(),
			identity.TenantID,
			input.toModel(),
		)
		if err != nil {
			writeWidgetError(w, "create", err)
			return
		}

		writeJSON(w, http.StatusCreated, map[string]any{"widget": widget})

	default:
		w.Header().Set("Allow", "GET, POST")
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *WidgetHandler) Item(w http.ResponseWriter, r *http.Request) {
	identity, ok := middleware.IdentityFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	widgetID := strings.TrimPrefix(r.URL.Path, "/widgets/")
	if !widgetIDPattern.MatchString(widgetID) {
		writeError(w, http.StatusBadRequest, "invalid widget ID")
		return
	}

	switch r.Method {
	case http.MethodGet:
		widget, err := h.widgets.Get(r.Context(), identity.TenantID, widgetID)
		if err != nil {
			writeWidgetError(w, "get", err)
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"widget": widget})

	case http.MethodPut:
		var input widgetRequest
		if err := decodeJSON(w, r, &input); err != nil {
			writeDecodeError(w, err)
			return
		}

		widget, err := h.widgets.Update(
			r.Context(),
			identity.TenantID,
			widgetID,
			input.toModel(),
		)
		if err != nil {
			writeWidgetError(w, "update", err)
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"widget": widget})

	case http.MethodDelete:
		err := h.widgets.Delete(r.Context(), identity.TenantID, widgetID)
		if err != nil {
			writeWidgetError(w, "delete", err)
			return
		}

		w.WriteHeader(http.StatusNoContent)

	default:
		w.Header().Set("Allow", "GET, PUT, DELETE")
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func writeWidgetError(w http.ResponseWriter, operation string, err error) {
	switch {
	case errors.Is(err, services.ErrInvalidWidget):
		writeError(w, http.StatusBadRequest, "invalid widget details")
	case errors.Is(err, services.ErrWidgetNotFound):
		// Also avoids revealing whether another tenant owns the ID.
		writeError(w, http.StatusNotFound, "widget not found")
	case errors.Is(err, services.ErrTenantIDRequired):
		writeError(w, http.StatusUnauthorized, "authentication required")
	default:
		log.Printf("widget %s failed: %v", operation, err)
		writeError(w, http.StatusInternalServerError, "widget operation failed")
	}
}
