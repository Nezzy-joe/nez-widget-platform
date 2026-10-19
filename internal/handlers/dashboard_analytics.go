package handlers

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/Nezzy-joe/nez-widget-platform/internal/middleware"
	"github.com/Nezzy-joe/nez-widget-platform/internal/models"
	"github.com/Nezzy-joe/nez-widget-platform/internal/services"
)

type dashboardAnalyticsGetter interface {
	GetByTenant(
		ctx context.Context,
		tenantID string,
		widgetID string,
	) (*models.SubmissionAnalytics, error)
}

type DashboardAnalyticsHandler struct {
	analytics dashboardAnalyticsGetter
}

func NewDashboardAnalyticsHandler(
	analytics dashboardAnalyticsGetter,
) *DashboardAnalyticsHandler {
	return &DashboardAnalyticsHandler{
		analytics: analytics,
	}
}

// Get returns analytics for the authenticated tenant.
func (h *DashboardAnalyticsHandler) Get(
	w http.ResponseWriter,
	r *http.Request,
) {
	w.Header().Set("Cache-Control", "no-store")

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
	widgetID := strings.TrimSpace(query.Get("widget_id"))

	if widgetID != "" && !widgetIDPattern.MatchString(widgetID) {
		writeError(w, http.StatusBadRequest, "invalid widget ID")
		return
	}

	// Tenant ownership comes from the verified JWT, never from query input.
	analytics, err := h.analytics.GetByTenant(
		r.Context(),
		identity.TenantID,
		widgetID,
	)
	if err != nil {
		if errors.Is(err, services.ErrTenantIDRequired) {
			writeError(w, http.StatusUnauthorized, "authentication required")
			return
		}

		log.Printf("get dashboard analytics failed: %v", err)
		writeError(
			w,
			http.StatusInternalServerError,
			"could not retrieve analytics",
		)
		return
	}

	if analytics == nil {
		log.Printf("analytics service returned a nil result")
		writeError(
			w,
			http.StatusInternalServerError,
			"could not retrieve analytics",
		)
		return
	}

	writeJSON(w, http.StatusOK, analytics)
}
