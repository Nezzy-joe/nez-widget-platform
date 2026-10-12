package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/Nezzy-joe/nez-widget-platform/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrEmailAlreadyExists = errors.New("email already exists")
)

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{
		db: db,
	}
}

func (r *UserRepository) FindByEmail(
	ctx context.Context,
	email string,
) (*models.User, error) {
	const query = `
		SELECT
			id,
			tenant_id,
			email,
			password_hash,
			created_at,
			updated_at
		FROM users
		WHERE email = $1
	`

	var user models.User

	err := r.db.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.TenantID,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}

		return nil, fmt.Errorf("find user by email: %w", err)
	}

	return &user, nil
}

func (r *UserRepository) CreateTenantAndUser(
	ctx context.Context,
	tenantName string,
	email string,
	passwordHash string,
) (*models.User, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin registration transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	const insertTenantQuery = `
		INSERT INTO tenants (name)
		VALUES ($1)
		RETURNING id
	`

	var tenantID string

	if err := tx.QueryRow(
		ctx,
		insertTenantQuery,
		tenantName,
	).Scan(&tenantID); err != nil {
		return nil, fmt.Errorf("create tenant: %w", err)
	}

	const insertUserQuery = `
		INSERT INTO users (
			tenant_id,
			email,
			password_hash
		)
		VALUES ($1, $2, $3)
		RETURNING
			id,
			tenant_id,
			email,
			password_hash,
			created_at,
			updated_at
	`

	var user models.User

	if err := tx.QueryRow(
		ctx,
		insertUserQuery,
		tenantID,
		email,
		passwordHash,
	).Scan(
		&user.ID,
		&user.TenantID,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
	); err != nil {
		var pgErr *pgconn.PgError

		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrEmailAlreadyExists
		}

		return nil, fmt.Errorf("create user: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit registration transaction: %w", err)
	}

	return &user, nil
}
