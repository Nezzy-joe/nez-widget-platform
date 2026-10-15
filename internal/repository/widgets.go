package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/Nezzy-joe/nez-widget-platform/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrWidgetNotFound   = errors.New("widget not found")
	ErrTenantIDRequired = errors.New("tenant ID is required")
)

type WidgetRepository struct {
	db *pgxpool.Pool
}

func NewWidgetRepository(db *pgxpool.Pool) *WidgetRepository {
	return &WidgetRepository{db: db}
}

type rowScanner interface {
	Scan(dest ...any) error
}

const widgetColumns = `
	id,
	tenant_id,
	public_id,
	type,
	title,
	description,
	button_text,
	display_options,
	created_at,
	updated_at
`

func scanWidget(row rowScanner) (*models.Widget, error) {
	var widget models.Widget
	var displayOptions []byte

	err := row.Scan(
		&widget.ID,
		&widget.TenantID,
		&widget.PublicID,
		&widget.Type,
		&widget.Title,
		&widget.Description,
		&widget.ButtonText,
		&displayOptions,
		&widget.CreatedAt,
		&widget.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	widget.DisplayOptions = json.RawMessage(displayOptions)
	return &widget, nil
}

func requireTenantID(tenantID string) error {
	if strings.TrimSpace(tenantID) == "" {
		return ErrTenantIDRequired
	}
	return nil
}

func validateDisplayOptions(raw json.RawMessage) (string, error) {
	value := strings.TrimSpace(string(raw))

	if value == "" {
		return `{}`, nil
	}

	if !json.Valid([]byte(value)) {
		return "", errors.New("display options must be valid JSON")
	}

	return value, nil
}

// Create inserts a widget and all its fields atomically.
// tenantID is supplied separately and never taken from client-controlled data.
func (r *WidgetRepository) Create(
	ctx context.Context,
	tenantID string,
	widget *models.Widget,
) (*models.Widget, error) {
	if err := requireTenantID(tenantID); err != nil {
		return nil, err
	}
	if widget == nil {
		return nil, errors.New("widget is required")
	}
	if strings.TrimSpace(widget.PublicID) == "" {
		return nil, errors.New("public ID is required")
	}

	displayOptions, err := validateDisplayOptions(widget.DisplayOptions)
	if err != nil {
		return nil, err
	}

	buttonText := strings.TrimSpace(widget.ButtonText)
	if buttonText == "" {
		buttonText = "Submit"
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin create widget transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(context.Background())
	}()

	query := `
		INSERT INTO widgets (
			tenant_id,
			public_id,
			type,
			title,
			description,
			button_text,
			display_options
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb)
		RETURNING ` + widgetColumns

	created, err := scanWidget(tx.QueryRow(
		ctx,
		query,
		tenantID,
		widget.PublicID,
		widget.Type,
		widget.Title,
		widget.Description,
		buttonText,
		displayOptions,
	))
	if err != nil {
		return nil, fmt.Errorf("insert widget: %w", err)
	}

	fields, err := insertWidgetFields(ctx, tx, created.ID, widget.Fields)
	if err != nil {
		return nil, err
	}
	created.Fields = fields

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit create widget transaction: %w", err)
	}

	return created, nil
}

func insertWidgetFields(
	ctx context.Context,
	tx pgx.Tx,
	widgetID string,
	fields []models.WidgetField,
) ([]models.WidgetField, error) {
	createdFields := make([]models.WidgetField, 0, len(fields))

	const query = `
		INSERT INTO widget_fields (
			widget_id,
			name,
			label,
			field_type,
			required,
			position
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at
	`

	for _, field := range fields {
		var id string
		var createdAt = field.CreatedAt

		err := tx.QueryRow(
			ctx,
			query,
			widgetID,
			field.Name,
			field.Label,
			field.FieldType,
			field.Required,
			field.Position,
		).Scan(&id, &createdAt)
		if err != nil {
			return nil, fmt.Errorf("insert widget field: %w", err)
		}

		field.ID = id
		field.WidgetID = widgetID
		field.CreatedAt = createdAt
		createdFields = append(createdFields, field)
	}

	return createdFields, nil
}

// ListByTenant returns only widgets belonging to the specified tenant.
func (r *WidgetRepository) ListByTenant(
	ctx context.Context,
	tenantID string,
) ([]models.Widget, error) {
	if err := requireTenantID(tenantID); err != nil {
		return nil, err
	}

	query := `
		SELECT ` + widgetColumns + `
		FROM widgets
		WHERE tenant_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list widgets: %w", err)
	}

	var widgets []models.Widget

	for rows.Next() {
		widget, err := scanWidget(rows)
		if err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan widget: %w", err)
		}
		widgets = append(widgets, *widget)
	}

	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, fmt.Errorf("iterate widgets: %w", err)
	}

	for i := range widgets {
		fields, err := r.loadWidgetFields(ctx, widgets[i].ID)
		if err != nil {
			return nil, err
		}
		widgets[i].Fields = fields
	}

	return widgets, nil
}

// GetByID scopes the lookup to both the widget ID and tenant ID.
func (r *WidgetRepository) GetByID(
	ctx context.Context,
	tenantID string,
	widgetID string,
) (*models.Widget, error) {

	if err := requireTenantID(tenantID); err != nil {
		return nil, err
	}

	query := `
		SELECT ` + widgetColumns + `
		FROM widgets
		WHERE id = $1 AND tenant_id = $2
	`

	widget, err := scanWidget(r.db.QueryRow(ctx, query, widgetID, tenantID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrWidgetNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get widget: %w", err)
	}

	widget.Fields, err = r.loadWidgetFields(ctx, widget.ID)
	if err != nil {
		return nil, err
	}

	return widget, nil
}

// GetByPublicID fetches a widget using its unique public identifier.
// This lookup is intended for public widget configuration, not management.
func (r *WidgetRepository) GetByPublicID(
	ctx context.Context,
	publicID string,
) (*models.Widget, error) {
	publicID = strings.TrimSpace(publicID)
	if publicID == "" {
		return nil, ErrWidgetNotFound
	}

	query := `
        SELECT ` + widgetColumns + `
        FROM widgets
        WHERE public_id = $1
    `

	widget, err := scanWidget(
		r.db.QueryRow(ctx, query, publicID),
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrWidgetNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get widget by public ID: %w", err)
	}

	widget.Fields, err = r.loadWidgetFields(ctx, widget.ID)
	if err != nil {
		return nil, err
	}

	return widget, nil
}

// Update replaces widget properties and its full field list in one transaction.
// The public ID is deliberately immutable.
func (r *WidgetRepository) Update(
	ctx context.Context,
	tenantID string,
	widgetID string,
	widget *models.Widget,
) (*models.Widget, error) {
	if err := requireTenantID(tenantID); err != nil {
		return nil, err
	}
	if widget == nil {
		return nil, errors.New("widget is required")
	}

	displayOptions, err := validateDisplayOptions(widget.DisplayOptions)
	if err != nil {
		return nil, err
	}

	buttonText := strings.TrimSpace(widget.ButtonText)
	if buttonText == "" {
		buttonText = "Submit"
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin update widget transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(context.Background())
	}()

	query := `
		UPDATE widgets
		SET
			type = $1,
			title = $2,
			description = $3,
			button_text = $4,
			display_options = $5::jsonb,
			updated_at = NOW()
		WHERE id = $6 AND tenant_id = $7
		RETURNING ` + widgetColumns

	updated, err := scanWidget(tx.QueryRow(
		ctx,
		query,
		widget.Type,
		widget.Title,
		widget.Description,
		buttonText,
		displayOptions,
		widgetID,
		tenantID,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrWidgetNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("update widget: %w", err)
	}

	if _, err := tx.Exec(
		ctx,
		`DELETE FROM widget_fields WHERE widget_id = $1`,
		updated.ID,
	); err != nil {
		return nil, fmt.Errorf("replace widget fields: %w", err)
	}

	fields, err := insertWidgetFields(ctx, tx, updated.ID, widget.Fields)
	if err != nil {
		return nil, err
	}
	updated.Fields = fields

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit update widget transaction: %w", err)
	}

	return updated, nil
}

// Delete only removes a widget owned by the specified tenant.
func (r *WidgetRepository) Delete(
	ctx context.Context,
	tenantID string,
	widgetID string,
) error {
	if err := requireTenantID(tenantID); err != nil {
		return err
	}

	result, err := r.db.Exec(
		ctx,
		`DELETE FROM widgets WHERE id = $1 AND tenant_id = $2`,
		widgetID,
		tenantID,
	)
	if err != nil {
		return fmt.Errorf("delete widget: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrWidgetNotFound
	}

	return nil
}

func (r *WidgetRepository) loadWidgetFields(
	ctx context.Context,
	widgetID string,
) ([]models.WidgetField, error) {
	const query = `
		SELECT
			id,
			widget_id,
			name,
			label,
			field_type,
			required,
			position,
			created_at
		FROM widget_fields
		WHERE widget_id = $1
		ORDER BY position, created_at
	`

	rows, err := r.db.Query(ctx, query, widgetID)
	if err != nil {
		return nil, fmt.Errorf("load widget fields: %w", err)
	}
	defer rows.Close()

	fields := make([]models.WidgetField, 0)

	for rows.Next() {
		var field models.WidgetField

		if err := rows.Scan(
			&field.ID,
			&field.WidgetID,
			&field.Name,
			&field.Label,
			&field.FieldType,
			&field.Required,
			&field.Position,
			&field.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan widget field: %w", err)
		}

		fields = append(fields, field)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate widget fields: %w", err)
	}

	return fields, nil
}
