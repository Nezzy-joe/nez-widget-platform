package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/Nezzy-joe/nez-widget-platform/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SubmissionRepository struct {
	db *pgxpool.Pool
}

func NewSubmissionRepository(db *pgxpool.Pool) *SubmissionRepository {
	return &SubmissionRepository{db: db}
}

// Create stores a submission only when the widget belongs to tenantID.
// The trusted tenantID argument takes precedence over submission.TenantID.
func (r *SubmissionRepository) Create(
	ctx context.Context,
	tenantID string,
	submission *models.Submission,
) (*models.Submission, error) {
	if err := requireTenantID(tenantID); err != nil {
		return nil, err
	}

	if submission == nil {
		return nil, errors.New("submission is required")
	}

	widgetID := strings.TrimSpace(submission.WidgetID)
	if widgetID == "" {
		return nil, errors.New("widget ID is required")
	}

	data := strings.TrimSpace(string(submission.Data))
	var fields map[string]json.RawMessage
	if data == "" ||
		json.Unmarshal([]byte(data), &fields) != nil ||
		fields == nil {
		return nil, errors.New("submission data must be a JSON object")
	}

	ipAddress := strings.TrimSpace(submission.IPAddress)
	if ipAddress != "" && net.ParseIP(ipAddress) == nil {
		return nil, errors.New("submission IP address is invalid")
	}

	// INSERT ... SELECT enforces tenant ownership in the database
	// statement itself. A mismatched tenant cannot create a submission.
	const query = `
		INSERT INTO submissions (
			tenant_id,
			widget_id,
			data,
			ip_address,
			country,
			city
		)
		SELECT
			$1,
			w.id,
			$3::jsonb,
			NULLIF($4, '')::inet,
			NULLIF($5, ''),
			NULLIF($6, '')
		FROM widgets AS w
		WHERE w.id = $2
		  AND w.tenant_id = $1
		RETURNING
			id,
			tenant_id,
			widget_id,
			data,
			host(ip_address),
			country,
			city,
			created_at
	`

	var created models.Submission
	var rawData []byte
	var savedIP, savedCountry, savedCity sql.NullString

	err := r.db.QueryRow(
		ctx,
		query,
		tenantID,
		widgetID,
		data,
		ipAddress,
		strings.TrimSpace(submission.Country),
		strings.TrimSpace(submission.City),
	).Scan(
		&created.ID,
		&created.TenantID,
		&created.WidgetID,
		&rawData,
		&savedIP,
		&savedCountry,
		&savedCity,
		&created.CreatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrWidgetNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("create submission: %w", err)
	}

	created.Data = json.RawMessage(rawData)
	if savedIP.Valid {
		created.IPAddress = savedIP.String
	}
	if savedCountry.Valid {
		created.Country = savedCountry.String
	}
	if savedCity.Valid {
		created.City = savedCity.String
	}

	return &created, nil
}
