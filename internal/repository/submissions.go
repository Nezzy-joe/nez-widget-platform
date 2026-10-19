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
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin submission transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(context.Background())
	}()
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

	err = tx.QueryRow(
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

	// Queue the confirmation job in the same transaction as the submission.
	_, err = tx.Exec(ctx, `
    INSERT INTO background_jobs (submission_id, job_type)
    VALUES ($1, $2)
    ON CONFLICT (submission_id, job_type) DO NOTHING
`, created.ID, JobTypeConfirmationEmail)
	if err != nil {
		return nil, fmt.Errorf("enqueue confirmation job: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit submission transaction: %w", err)
	}

	return &created, nil
}

// ListByTenant returns a paginated list of submissions belonging to a tenant.
// An optional widget ID narrows the results without removing tenant isolation.
func (r *SubmissionRepository) ListByTenant(
	ctx context.Context,
	tenantID string,
	widgetID string,
	limit int,
	offset int,
) ([]models.Submission, int, error) {
	if err := requireTenantID(tenantID); err != nil {
		return nil, 0, err
	}

	if limit < 1 || limit > 100 {
		return nil, 0, errors.New("limit must be between 1 and 100")
	}

	if offset < 0 {
		return nil, 0, errors.New("offset cannot be negative")
	}

	widgetID = strings.TrimSpace(widgetID)

	// The tenant filter is always present. The optional widget filter
	// can only narrow the tenant's results.
	where := "tenant_id = $1"
	countArgs := []any{tenantID}

	if widgetID != "" {
		where += " AND widget_id = $2"
		countArgs = append(countArgs, widgetID)
	}

	var total int
	countQuery := fmt.Sprintf(
		"SELECT COUNT(*) FROM submissions WHERE %s",
		where,
	)

	if err := r.db.QueryRow(
		ctx,
		countQuery,
		countArgs...,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf(
			"count tenant submissions: %w", err,
		)
	}

	args := append([]any{}, countArgs...)
	args = append(args, limit, offset)

	limitPosition := len(args) - 1
	offsetPosition := len(args)

	query := fmt.Sprintf(`
		SELECT
			id,
			tenant_id,
			widget_id,
			data,
			country,
			city,
			created_at
		FROM submissions
		WHERE %s
		ORDER BY created_at DESC, id DESC
		LIMIT $%d OFFSET $%d
	`, where, limitPosition, offsetPosition)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf(
			"list tenant submissions: %w", err,
		)
	}
	defer rows.Close()

	submissions := make([]models.Submission, 0)

	for rows.Next() {
		var submission models.Submission
		var rawData []byte
		var country, city sql.NullString

		if err := rows.Scan(
			&submission.ID,
			&submission.TenantID,
			&submission.WidgetID,
			&rawData,
			&country,
			&city,
			&submission.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf(
				"scan submission: %w", err,
			)
		}

		submission.Data = json.RawMessage(rawData)

		if country.Valid {
			submission.Country = country.String
		}
		if city.Valid {
			submission.City = city.String
		}

		submissions = append(submissions, submission)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf(
			"iterate submissions: %w", err,
		)
	}

	return submissions, total, nil
}
