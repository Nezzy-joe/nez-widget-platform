package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net"
	"net/http"
	"regexp"
	"strings"

	"github.com/Nezzy-joe/nez-widget-platform/internal/models"
	"github.com/Nezzy-joe/nez-widget-platform/internal/services"
)

var submissionPublicIDPattern = regexp.MustCompile(`^[0-9a-f]{36}$`)

type submissionCreator interface {
	Create(
		ctx context.Context,
		publicID string,
		data json.RawMessage,
		ipAddress string,
	) (*models.Submission, error)
}

type SubmissionHandler struct {
	submissions submissionCreator
}

func NewSubmissionHandler(
	submissions submissionCreator,
) *SubmissionHandler {
	return &SubmissionHandler{submissions: submissions}
}

type publicSubmissionRequest struct {
	PublicID string          `json:"public_id"`
	Data     json.RawMessage `json:"data"`
}

// Create accepts submissions from external websites.
// The service resolves the widget and determines its trusted tenant.
func (h *SubmissionHandler) Create(
	w http.ResponseWriter,
	r *http.Request,
) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Cache-Control", "no-store")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST, OPTIONS")
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var input publicSubmissionRequest
	if err := decodeJSON(w, r, &input); err != nil {
		writeDecodeError(w, err)
		return
	}

	input.PublicID = strings.TrimSpace(input.PublicID)
	if !submissionPublicIDPattern.MatchString(input.PublicID) {
		writeError(w, http.StatusBadRequest, "invalid public widget ID")
		return
	}

	created, err := h.submissions.Create(
		r.Context(),
		input.PublicID,
		input.Data,
		submissionClientIP(r),
	)
	if err != nil {
		writeSubmissionError(w, err)
		return
	}

	if created == nil {
		log.Printf("submission service returned a nil result")
		writeError(w, http.StatusInternalServerError, "submission failed")
		return
	}

	// Do not expose submission IDs, tenant IDs, or visitor data publicly.
	writeJSON(w, http.StatusCreated, map[string]string{
		"status": "accepted",
	})
}

// Trust the direct connection address. Do not trust forwarded IP headers
// until the application has an explicitly configured trusted-proxy policy.
func submissionClientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		if ip := net.ParseIP(host); ip != nil {
			return ip.String()
		}
	}

	if ip := net.ParseIP(r.RemoteAddr); ip != nil {
		return ip.String()
	}

	return ""
}

func writeSubmissionError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, services.ErrInvalidSubmission):
		writeError(w, http.StatusBadRequest, "invalid submission details")

	case errors.Is(err, services.ErrWidgetNotFound):
		writeError(w, http.StatusNotFound, "widget not found")

	default:
		log.Printf("create submission failed: %v", err)
		writeError(w, http.StatusInternalServerError, "submission failed")
	}
}
