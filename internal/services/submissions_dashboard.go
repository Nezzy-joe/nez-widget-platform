package services

import (
	"context"
	"errors"
	"strings"

	"github.com/Nezzy-joe/nez-widget-platform/internal/models"
)

// ErrInvalidSubmissionQuery indicates invalid dashboard pagination.
var ErrInvalidSubmissionQuery = errors.New("invalid submission query")

// TenantSubmissionReader retrieves submissions scoped to a tenant.
type TenantSubmissionReader interface {
	ListByTenant(
		ctx context.Context,
		tenantID string,
		widgetID string,
		limit int,
		offset int,
	) ([]models.Submission, int, error)
}

// DashboardSubmissionService handles authenticated dashboard queries.
type DashboardSubmissionService struct {
	submissions TenantSubmissionReader
}

func NewDashboardSubmissionService(
	submissions TenantSubmissionReader,
) *DashboardSubmissionService {
	return &DashboardSubmissionService{
		submissions: submissions,
	}
}

// ListByTenant retrieves a page of submissions for the authenticated tenant.
func (s *DashboardSubmissionService) ListByTenant(
	ctx context.Context,
	tenantID string,
	widgetID string,
	limit int,
	offset int,
) ([]models.Submission, int, error) {
	tenantID = strings.TrimSpace(tenantID)
	widgetID = strings.TrimSpace(widgetID)

	if tenantID == "" {
		return nil, 0, ErrTenantIDRequired
	}

	if limit < 1 || limit > 100 || offset < 0 {
		return nil, 0, ErrInvalidSubmissionQuery
	}

	return s.submissions.ListByTenant(
		ctx,
		tenantID,
		widgetID,
		limit,
		offset,
	)
}
