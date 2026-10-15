package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/Nezzy-joe/nez-widget-platform/internal/models"
	"github.com/Nezzy-joe/nez-widget-platform/internal/repository"
)

var (
	ErrInvalidWidget    = errors.New("invalid widget")
	ErrWidgetNotFound   = errors.New("widget not found")
	ErrTenantIDRequired = errors.New("tenant ID is required")
)

const (
	maxWidgetTitleLength       = 120
	maxWidgetDescriptionLength = 1000
	maxWidgetButtonLength      = 40
	maxWidgetFields            = 20
)

var fieldNamePattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,49}$`)

type WidgetStore interface {
	Create(
		ctx context.Context,
		tenantID string,
		widget *models.Widget,
	) (*models.Widget, error)

	ListByTenant(
		ctx context.Context,
		tenantID string,
	) ([]models.Widget, error)

	GetByID(
		ctx context.Context,
		tenantID string,
		widgetID string,
	) (*models.Widget, error)

	Update(
		ctx context.Context,
		tenantID string,
		widgetID string,
		widget *models.Widget,
	) (*models.Widget, error)

	Delete(
		ctx context.Context,
		tenantID string,
		widgetID string,
	) error
}

type WidgetService struct {
	widgets WidgetStore
}

func NewWidgetService(widgets WidgetStore) *WidgetService {
	return &WidgetService{widgets: widgets}
}

func (s *WidgetService) Create(
	ctx context.Context,
	tenantID string,
	widget *models.Widget,
) (*models.Widget, error) {
	if err := validateTenantID(tenantID); err != nil {
		return nil, err
	}

	if err := normalizeAndValidateWidget(widget); err != nil {
		return nil, err
	}

	publicID, err := generatePublicID()
	if err != nil {
		return nil, fmt.Errorf("generate widget public ID: %w", err)
	}

	// The client does not choose the internal ID, tenant, or public ID.
	widget.ID = ""
	widget.TenantID = tenantID
	widget.PublicID = publicID

	created, err := s.widgets.Create(ctx, tenantID, widget)
	if err != nil {
		return nil, fmt.Errorf("create widget: %w", err)
	}

	return created, nil
}

func (s *WidgetService) List(
	ctx context.Context,
	tenantID string,
) ([]models.Widget, error) {
	if err := validateTenantID(tenantID); err != nil {
		return nil, err
	}

	widgets, err := s.widgets.ListByTenant(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list widgets: %w", err)
	}

	if widgets == nil {
		widgets = []models.Widget{}
	}

	return widgets, nil
}

func (s *WidgetService) Get(
	ctx context.Context,
	tenantID string,
	widgetID string,
) (*models.Widget, error) {
	if err := validateTenantID(tenantID); err != nil {
		return nil, err
	}
	if strings.TrimSpace(widgetID) == "" {
		return nil, ErrInvalidWidget
	}

	widget, err := s.widgets.GetByID(ctx, tenantID, widgetID)
	if err != nil {
		return nil, translateWidgetError(err)
	}

	return widget, nil
}

func (s *WidgetService) Update(
	ctx context.Context,
	tenantID string,
	widgetID string,
	widget *models.Widget,
) (*models.Widget, error) {
	if err := validateTenantID(tenantID); err != nil {
		return nil, err
	}
	if strings.TrimSpace(widgetID) == "" {
		return nil, ErrInvalidWidget
	}
	if err := normalizeAndValidateWidget(widget); err != nil {
		return nil, err
	}

	// The repository keeps the original public ID immutable.
	widget.ID = ""
	widget.TenantID = tenantID
	widget.PublicID = ""

	updated, err := s.widgets.Update(ctx, tenantID, widgetID, widget)
	if err != nil {
		return nil, translateWidgetError(err)
	}

	return updated, nil
}

func (s *WidgetService) Delete(
	ctx context.Context,
	tenantID string,
	widgetID string,
) error {
	if err := validateTenantID(tenantID); err != nil {
		return err
	}
	if strings.TrimSpace(widgetID) == "" {
		return ErrInvalidWidget
	}

	err := s.widgets.Delete(ctx, tenantID, widgetID)
	if err != nil {
		return translateWidgetError(err)
	}

	return nil
}

func validateTenantID(tenantID string) error {
	if strings.TrimSpace(tenantID) == "" {
		return ErrTenantIDRequired
	}
	return nil
}

func translateWidgetError(err error) error {
	if errors.Is(err, repository.ErrWidgetNotFound) {
		return ErrWidgetNotFound
	}

	if errors.Is(err, repository.ErrTenantIDRequired) {
		return ErrTenantIDRequired
	}

	return err
}

func normalizeAndValidateWidget(widget *models.Widget) error {
	if widget == nil {
		return ErrInvalidWidget
	}

	widget.Type = strings.ToLower(strings.TrimSpace(widget.Type))
	widget.Title = strings.TrimSpace(widget.Title)
	widget.ButtonText = strings.TrimSpace(widget.ButtonText)

	switch widget.Type {
	case "signup", "contact", "cta", "popover":
	default:
		return ErrInvalidWidget
	}

	if widget.Title == "" ||
		utf8.RuneCountInString(widget.Title) > maxWidgetTitleLength {
		return ErrInvalidWidget
	}

	if widget.Description != nil {
		description := strings.TrimSpace(*widget.Description)

		if utf8.RuneCountInString(description) > maxWidgetDescriptionLength {
			return ErrInvalidWidget
		}

		if description == "" {
			widget.Description = nil
		} else {
			widget.Description = &description
		}
	}

	if widget.ButtonText == "" {
		widget.ButtonText = "Submit"
	}
	if utf8.RuneCountInString(widget.ButtonText) > maxWidgetButtonLength {
		return ErrInvalidWidget
	}

	// An omitted display_options value means an empty configuration.
	raw := strings.TrimSpace(string(widget.DisplayOptions))
	if raw == "" {
		raw = "{}"
	}

	var options map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &options); err != nil || options == nil {
		return ErrInvalidWidget
	}
	widget.DisplayOptions = json.RawMessage(raw)

	if len(widget.Fields) > maxWidgetFields {
		return ErrInvalidWidget
	}

	// Form widgets need at least one field.
	if (widget.Type == "signup" || widget.Type == "contact") &&
		len(widget.Fields) == 0 {
		return ErrInvalidWidget
	}

	seenNames := make(map[string]struct{}, len(widget.Fields))

	for i := range widget.Fields {
		field := &widget.Fields[i]

		field.Name = strings.ToLower(strings.TrimSpace(field.Name))
		field.Label = strings.TrimSpace(field.Label)
		field.FieldType = strings.ToLower(strings.TrimSpace(field.FieldType))

		if !fieldNamePattern.MatchString(field.Name) {
			return ErrInvalidWidget
		}
		if field.Label == "" || utf8.RuneCountInString(field.Label) > 80 {
			return ErrInvalidWidget
		}

		switch field.FieldType {
		case "text", "email", "textarea", "tel",
			"number", "url", "checkbox", "select":
		default:
			return ErrInvalidWidget
		}

		if _, exists := seenNames[field.Name]; exists {
			return ErrInvalidWidget
		}
		seenNames[field.Name] = struct{}{}

		// The submitted array defines field ordering.
		field.Position = i

		// IDs and timestamps are generated by the server/database.
		field.ID = ""
		field.WidgetID = ""
	}

	return nil
}

func generatePublicID() (string, error) {
	randomBytes := make([]byte, 18)

	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}

	return hex.EncodeToString(randomBytes), nil
}
