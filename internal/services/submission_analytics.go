package services

import (
	"context"
	"strings"

	"github.com/Nezzy-joe/nez-widget-platform/internal/models"
)

// SubmissionAnalyticsReader retrieves analytics scoped to a tenant.
type SubmissionAnalyticsReader interface {
	GetAnalyticsByTenant(
		ctx context.Context,
		tenantID string,
		widgetID string,
	) (*models.SubmissionAnalytics, error)
}

// DashboardAnalyticsService provides analytics for the authenticated dashboard.
type DashboardAnalyticsService struct {
	analytics SubmissionAnalyticsReader
}

func NewDashboardAnalyticsService(
	analytics SubmissionAnalyticsReader,
) *DashboardAnalyticsService {
	return &DashboardAnalyticsService{
		analytics: analytics,
	}
}

// GetByTenant retrieves analytics using the trusted tenant identity.
func (s *DashboardAnalyticsService) GetByTenant(
	ctx context.Context,
	tenantID string,
	widgetID string,
) (*models.SubmissionAnalytics, error) {
	tenantID = strings.TrimSpace(tenantID)
	widgetID = strings.TrimSpace(widgetID)

	if tenantID == "" {
		return nil, ErrTenantIDRequired
	}

	return s.analytics.GetAnalyticsByTenant(
		ctx,
		tenantID,
		widgetID,
	)
}
