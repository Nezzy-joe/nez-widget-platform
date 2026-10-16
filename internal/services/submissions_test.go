package services

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/Nezzy-joe/nez-widget-platform/internal/models"
)

type submissionWidgetLookupStub struct {
	widget *models.Widget
	err    error
}

func (s *submissionWidgetLookupStub) GetPublic(
	_ context.Context,
	publicID string,
) (*models.Widget, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.widget == nil {
		return nil, errors.New("unexpected widget lookup")
	}
	if publicID != s.widget.PublicID {
		return nil, errors.New("unexpected public ID")
	}
	return s.widget, nil
}

type submissionStoreStub struct {
	calls    int
	tenantID string
	input    *models.Submission
}

func (s *submissionStoreStub) Create(
	_ context.Context,
	tenantID string,
	submission *models.Submission,
) (*models.Submission, error) {
	s.calls++
	s.tenantID = tenantID
	s.input = submission

	saved := *submission
	saved.ID = "saved-submission-id"
	return &saved, nil
}

func validSubmissionWidget() *models.Widget {
	return &models.Widget{
		ID:       "internal-widget-id",
		TenantID: "trusted-tenant-id",
		PublicID: "public-widget-id",
		Fields: []models.WidgetField{
			{
				Name:      "full_name",
				FieldType: "text",
				Required:  true,
			},
			{
				Name:      "email",
				FieldType: "email",
				Required:  true,
			},
		},
	}
}

func TestSubmissionServiceCreateValidSubmission(t *testing.T) {
	widget := validSubmissionWidget()
	store := &submissionStoreStub{}

	service := NewSubmissionService(
		&submissionWidgetLookupStub{widget: widget},
		store,
	)

	submission, err := service.Create(
		context.Background(),
		widget.PublicID,
		json.RawMessage(`{
			"full_name": "Joseph Amos",
			"email": "joseph@example.com"
		}`),
		"203.0.113.10",
	)
	if err != nil {
		t.Fatalf("Create returned an error: %v", err)
	}

	if store.calls != 1 {
		t.Fatalf("expected one persistence call, got %d", store.calls)
	}
	if store.tenantID != widget.TenantID {
		t.Fatalf("wrong tenant used for persistence: %q", store.tenantID)
	}
	if store.input.WidgetID != widget.ID {
		t.Fatalf("wrong widget ID: %q", store.input.WidgetID)
	}
	if store.input.TenantID != widget.TenantID {
		t.Fatalf("wrong submission tenant: %q", store.input.TenantID)
	}
	if store.input.IPAddress != "203.0.113.10" {
		t.Fatalf("unexpected IP address: %q", store.input.IPAddress)
	}
	if submission.ID != "saved-submission-id" {
		t.Fatalf("unexpected saved submission ID: %q", submission.ID)
	}
}

func TestSubmissionServiceRejectsInvalidPayloads(t *testing.T) {
	oversizedValue := `{"full_name":"` +
		strings.Repeat("x", maxSubmissionStringSize+1) +
		`","email":"joseph@example.com"}`

	tests := []struct {
		name string
		data json.RawMessage
	}{
		{
			name: "malformed JSON",
			data: json.RawMessage(`{"full_name":`),
		},
		{
			name: "top-level array",
			data: json.RawMessage(`[]`),
		},
		{
			name: "missing required email",
			data: json.RawMessage(`{"full_name":"Joseph Amos"}`),
		},
		{
			name: "blank required name",
			data: json.RawMessage(
				`{"full_name":" ","email":"joseph@example.com"}`,
			),
		},
		{
			name: "invalid email",
			data: json.RawMessage(
				`{"full_name":"Joseph Amos","email":"not-an-email"}`,
			),
		},
		{
			name: "unknown field",
			data: json.RawMessage(
				`{"full_name":"Joseph Amos","email":"joseph@example.com","role":"admin"}`,
			),
		},
		{
			name: "wrong field type",
			data: json.RawMessage(
				`{"full_name":"Joseph Amos","email":123}`,
			),
		},
		{
			name: "oversized field value",
			data: json.RawMessage(oversizedValue),
		},
		{
			name: "oversized request body",
			data: json.RawMessage(
				strings.Repeat("x", maxSubmissionBytes+1),
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			widget := validSubmissionWidget()
			store := &submissionStoreStub{}

			service := NewSubmissionService(
				&submissionWidgetLookupStub{widget: widget},
				store,
			)

			_, err := service.Create(
				context.Background(),
				widget.PublicID,
				tt.data,
				"",
			)

			if !errors.Is(err, ErrInvalidSubmission) {
				t.Fatalf("expected ErrInvalidSubmission, got %v", err)
			}
			if store.calls != 0 {
				t.Fatal("invalid submission reached the repository")
			}
		})
	}
}

func TestSubmissionServiceRejectsUnknownWidget(t *testing.T) {
	store := &submissionStoreStub{}
	service := NewSubmissionService(
		&submissionWidgetLookupStub{
			err: ErrWidgetNotFound,
		},
		store,
	)

	_, err := service.Create(
		context.Background(),
		"missing-public-id",
		json.RawMessage(`{"full_name":"Joseph Amos"}`),
		"",
	)

	if !errors.Is(err, ErrWidgetNotFound) {
		t.Fatalf("expected ErrWidgetNotFound, got %v", err)
	}
	if store.calls != 0 {
		t.Fatal("unknown widget must not reach the repository")
	}
}
