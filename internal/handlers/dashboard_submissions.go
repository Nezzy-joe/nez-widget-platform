package handlers

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/Nezzy-joe/nez-widget-platform/internal/middleware"
	"github.com/Nezzy-joe/nez-widget-platform/internal/models"
	"github.com/Nezzy-joe/nez-widget-platform/internal/services"
)

const (
	defaultDashboardSubmissionLimit = 20
	maxDashboardSubmissionLimit     = 100
	maxDashboardSubmissionOffset    = 1000000
)

type dashboardSubmissionLister interface {
	ListByTenant(
		ctx context.Context,
		tenantID string,
		widgetID string,
		limit int,
		offset int,
	) ([]models.Submission, int, error)
}

type DashboardSubmissionHandler struct {
	submissions dashboardSubmissionLister
}

func NewDashboardSubmissionHandler(
	submissions dashboardSubmissionLister,
) *DashboardSubmissionHandler {
	return &DashboardSubmissionHandler{
		submissions: submissions,
	}
}

type dashboardSubmissionResponse struct {
	Submissions []models.Submission  `json:"submissions"`
	Pagination  submissionPagination `json:"pagination"`
}

type submissionPagination struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
	Total  int `json:"total"`
}

// List returns submissions belonging to the authenticated tenant.
func (h *DashboardSubmissionHandler) List(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	identity, ok := middleware.IdentityFromContext(r.Context())
	if !ok || strings.TrimSpace(identity.TenantID) == "" {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	query := r.URL.Query()

	limit := defaultDashboardSubmissionLimit
	if raw := query.Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil ||
			parsed < 1 ||
			parsed > maxDashboardSubmissionLimit {
			writeError(
				w,
				http.StatusBadRequest,
				"limit must be between 1 and 100",
			)
			return
		}
		limit = parsed
	}

	offset := 0
	if raw := query.Get("offset"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil ||
			parsed < 0 ||
			parsed > maxDashboardSubmissionOffset {
			writeError(
				w,
				http.StatusBadRequest,
				"offset must be between 0 and 1000000",
			)
			return
		}
		offset = parsed
	}

	widgetID := strings.TrimSpace(query.Get("widget_id"))
	if widgetID != "" && !widgetIDPattern.MatchString(widgetID) {
		writeError(w, http.StatusBadRequest, "invalid widget ID")
		return
	}

	// Never take the tenant ID from query parameters.
	// The verified authentication context is the source of authority.
	submissions, total, err := h.submissions.ListByTenant(
		r.Context(),
		identity.TenantID,
		widgetID,
		limit,
		offset,
	)
	if err != nil {
		if errors.Is(err, services.ErrInvalidSubmissionQuery) {
			writeError(w, http.StatusBadRequest, "invalid submission query")
			return
		}

		log.Printf("list dashboard submissions failed: %v", err)
		writeError(
			w,
			http.StatusInternalServerError,
			"could not retrieve submissions",
		)
		return
	}

	if submissions == nil {
		submissions = make([]models.Submission, 0)
	}

	writeJSON(w, http.StatusOK, dashboardSubmissionResponse{
		Submissions: submissions,
		Pagination: submissionPagination{
			Limit:  limit,
			Offset: offset,
			Total:  total,
		},
	})
}
