package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Nezzy-joe/nez-widget-platform/internal/models"
	"github.com/Nezzy-joe/nez-widget-platform/internal/services"
)

const handlerTestPublicID = "0123456789abcdef0123456789abcdef0123"

type submissionCreatorStub struct {
	calls    int
	publicID string
	data     json.RawMessage
	ip       string
	result   *models.Submission
	err      error
}

func (s *submissionCreatorStub) Create(
	_ context.Context,
	publicID string,
	data json.RawMessage,
	ip string,
) (*models.Submission, error) {
	s.calls++
	s.publicID = publicID
	s.data = data
	s.ip = ip

	return s.result, s.err
}

func TestSubmissionHandlerCreatesSubmission(t *testing.T) {
	store := &submissionCreatorStub{
		result: &models.Submission{
			ID:       "internal-submission-id",
			TenantID: "internal-tenant-id",
			WidgetID: "internal-widget-id",
		},
	}

	handler := NewSubmissionHandler(store)
	body := `{
		"public_id": "` + handlerTestPublicID + `",
		"data": {
			"full_name": "Joseph",
			"email": "joseph@example.com"
		}
	}`

	request := httptest.NewRequest(
		http.MethodPost,
		"/submissions",
		strings.NewReader(body),
	)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	handler.Create(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf(
			"expected 201, got %d: %s",
			response.Code,
			response.Body.String(),
		)
	}

	if store.calls != 1 {
		t.Fatalf("expected one service call, got %d", store.calls)
	}

	if store.publicID != handlerTestPublicID {
		t.Fatalf("unexpected public ID: %q", store.publicID)
	}

	if !strings.Contains(string(store.data), `"email"`) {
		t.Fatalf("submitted data was not passed to service: %s", store.data)
	}

	if !strings.Contains(response.Body.String(), `"status":"accepted"`) {
		t.Fatalf("unexpected response body: %s", response.Body.String())
	}

	for _, secret := range []string{
		"internal-submission-id",
		"internal-tenant-id",
		"internal-widget-id",
		"joseph@example.com",
	} {
		if strings.Contains(response.Body.String(), secret) {
			t.Errorf("response leaked private value %q", secret)
		}
	}

	if response.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatal("missing public CORS header")
	}

	if response.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("submission response must not be cached")
	}
}

func TestSubmissionHandlerRejectsInvalidPublicID(t *testing.T) {
	store := &submissionCreatorStub{}
	handler := NewSubmissionHandler(store)

	request := httptest.NewRequest(
		http.MethodPost,
		"/submissions",
		strings.NewReader(
			`{"public_id":"invalid","data":{"name":"Joseph"}}`,
		),
	)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	handler.Create(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", response.Code)
	}

	if store.calls != 0 {
		t.Fatal("invalid public ID reached the service")
	}
}

func TestSubmissionHandlerMapsUnknownWidget(t *testing.T) {
	store := &submissionCreatorStub{
		err: services.ErrWidgetNotFound,
	}
	handler := NewSubmissionHandler(store)

	request := httptest.NewRequest(
		http.MethodPost,
		"/submissions",
		strings.NewReader(
			`{"public_id":"`+handlerTestPublicID+
				`","data":{"name":"Joseph"}}`,
		),
	)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	handler.Create(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf(
			"expected 404, got %d: %s",
			response.Code,
			response.Body.String(),
		)
	}
}

func TestSubmissionHandlerHandlesCORSPreflight(t *testing.T) {
	store := &submissionCreatorStub{}
	handler := NewSubmissionHandler(store)

	request := httptest.NewRequest(
		http.MethodOptions,
		"/submissions",
		nil,
	)
	response := httptest.NewRecorder()

	handler.Create(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", response.Code)
	}

	if response.Header().Get("Access-Control-Allow-Methods") != "POST, OPTIONS" {
		t.Fatal("unexpected allowed methods")
	}

	if store.calls != 0 {
		t.Fatal("preflight request must not create a submission")
	}
}

func TestSubmissionHandlerRejectsWrongMethod(t *testing.T) {
	store := &submissionCreatorStub{}
	handler := NewSubmissionHandler(store)

	request := httptest.NewRequest(http.MethodGet, "/submissions", nil)
	response := httptest.NewRecorder()

	handler.Create(response, request)

	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", response.Code)
	}

	if response.Header().Get("Allow") != "POST, OPTIONS" {
		t.Fatal("unexpected Allow header")
	}

	if store.calls != 0 {
		t.Fatal("GET request must not create a submission")
	}
}
