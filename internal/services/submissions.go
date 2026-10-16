package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"net/mail"
	"net/url"
	"strings"
	"unicode/utf8"

	"github.com/Nezzy-joe/nez-widget-platform/internal/models"
	"github.com/Nezzy-joe/nez-widget-platform/internal/repository"
)

const (
	maxSubmissionBytes      = 16 * 1024
	maxSubmissionStringSize = 4096
)

var ErrInvalidSubmission = errors.New("invalid submission")

// PublicWidgetLookup retrieves widget configuration by public ID.
type PublicWidgetLookup interface {
	GetPublic(
		ctx context.Context,
		publicID string,
	) (*models.Widget, error)
}

// SubmissionStore persists submissions under a trusted tenant ID.
type SubmissionStore interface {
	Create(
		ctx context.Context,
		tenantID string,
		submission *models.Submission,
	) (*models.Submission, error)
}
type GeoLookup interface {
	Lookup(
		ctx context.Context,
		ipAddress string,
	) (models.GeoLocation, error)
}
type SubmissionService struct {
	widgets     PublicWidgetLookup
	submissions SubmissionStore
	geolocator  GeoLookup
}

func NewSubmissionService(
	widgets PublicWidgetLookup,
	submissions SubmissionStore,
	geolocator GeoLookup,
) *SubmissionService {
	return &SubmissionService{
		widgets:     widgets,
		submissions: submissions,
		geolocator:  geolocator,
	}
}

// Create validates visitor input against the widget's configured fields.
// Tenant and widget IDs are taken from the stored widget, never from the visitor.
func (s *SubmissionService) Create(
	ctx context.Context,
	publicID string,
	data json.RawMessage,
	ipAddress string,
) (*models.Submission, error) {
	publicID = strings.TrimSpace(publicID)
	if publicID == "" {
		return nil, ErrInvalidSubmission
	}

	widget, err := s.widgets.GetPublic(ctx, publicID)
	if err != nil {
		return nil, translateWidgetError(err)
	}
	if widget == nil ||
		strings.TrimSpace(widget.ID) == "" ||
		strings.TrimSpace(widget.TenantID) == "" {
		return nil, ErrWidgetNotFound
	}
	if len(widget.Fields) == 0 {
		return nil, ErrInvalidSubmission
	}

	validatedData, err := validateSubmissionData(data, widget.Fields)
	if err != nil {
		return nil, err
	}

	var location models.GeoLocation

	if s.geolocator != nil && strings.TrimSpace(ipAddress) != "" {
		resolved, geoErr := s.geolocator.Lookup(ctx, ipAddress)
		if geoErr != nil {
			// Geolocation is optional. A provider outage must not
			// prevent a valid submission from being stored.
			log.Printf(
				"geolocation unavailable; continuing without location data",
			)
		} else {
			location = resolved
		}
	}

	submission := &models.Submission{
		TenantID:  widget.TenantID,
		WidgetID:  widget.ID,
		Data:      validatedData,
		IPAddress: ipAddress,
		Country:   location.Country,
		City:      location.City,
	}

	created, err := s.submissions.Create(
		ctx,
		widget.TenantID,
		submission,
	)
	if err != nil {
		if errors.Is(err, repository.ErrWidgetNotFound) {
			return nil, ErrWidgetNotFound
		}
		if errors.Is(err, repository.ErrTenantIDRequired) {
			return nil, ErrTenantIDRequired
		}
		return nil, fmt.Errorf("store submission: %w", err)
	}

	return created, nil
}

func validateSubmissionData(
	data json.RawMessage,
	fields []models.WidgetField,
) (json.RawMessage, error) {
	if len(data) == 0 || len(data) > maxSubmissionBytes {
		return nil, ErrInvalidSubmission
	}

	var values map[string]json.RawMessage
	if err := json.Unmarshal(data, &values); err != nil || values == nil {
		return nil, ErrInvalidSubmission
	}

	// Build the allowlist from the widget's stored form definition.
	allowed := make(map[string]models.WidgetField, len(fields))
	for _, field := range fields {
		name := strings.TrimSpace(field.Name)
		if name == "" {
			return nil, ErrInvalidSubmission
		}
		if _, exists := allowed[name]; exists {
			return nil, ErrInvalidSubmission
		}
		allowed[name] = field
	}

	// Do not accept fields that the widget owner did not configure.
	for name := range values {
		if _, exists := allowed[name]; !exists {
			return nil, ErrInvalidSubmission
		}
	}

	for name, field := range allowed {
		rawValue, present := values[name]
		if !present {
			if field.Required {
				return nil, ErrInvalidSubmission
			}
			continue
		}

		value := strings.TrimSpace(string(rawValue))
		if value == "" || value == "null" {
			return nil, ErrInvalidSubmission
		}

		switch field.FieldType {
		case "text", "textarea", "tel", "email", "url", "select":
			var text string
			if err := json.Unmarshal(rawValue, &text); err != nil {
				return nil, ErrInvalidSubmission
			}
			if utf8.RuneCountInString(text) > maxSubmissionStringSize {
				return nil, ErrInvalidSubmission
			}
			if field.Required && strings.TrimSpace(text) == "" {
				return nil, ErrInvalidSubmission
			}

			if text == "" {
				continue
			}

			switch field.FieldType {
			case "email":
				address, err := mail.ParseAddress(text)
				if err != nil || address.Address != text {
					return nil, ErrInvalidSubmission
				}
			case "url":
				parsed, err := url.Parse(text)
				if err != nil ||
					(parsed.Scheme != "http" && parsed.Scheme != "https") ||
					parsed.Host == "" {
					return nil, ErrInvalidSubmission
				}
			}

		case "number":
			var number float64
			if err := json.Unmarshal(rawValue, &number); err != nil ||
				math.IsNaN(number) || math.IsInf(number, 0) {
				return nil, ErrInvalidSubmission
			}

		case "checkbox":
			var checked bool
			if err := json.Unmarshal(rawValue, &checked); err != nil {
				return nil, ErrInvalidSubmission
			}
			if field.Required && !checked {
				return nil, ErrInvalidSubmission
			}

		default:
			return nil, ErrInvalidSubmission
		}
	}

	// Store validated JSON, preserving the submitted values.
	return json.RawMessage(data), nil
}
