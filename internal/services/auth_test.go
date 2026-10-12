package services

import (
	"context"
	"errors"
	"testing"

	"github.com/Nezzy-joe/nez-widget-platform/internal/models"
	"github.com/Nezzy-joe/nez-widget-platform/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

type fakeUserStore struct {
	findByEmailFunc         func(context.Context, string) (*models.User, error)
	createTenantAndUserFunc func(
		context.Context,
		string,
		string,
		string,
	) (*models.User, error)
}

func (f *fakeUserStore) FindByEmail(
	ctx context.Context,
	email string,
) (*models.User, error) {
	return f.findByEmailFunc(ctx, email)
}

func (f *fakeUserStore) CreateTenantAndUser(
	ctx context.Context,
	tenantName string,
	email string,
	passwordHash string,
) (*models.User, error) {
	return f.createTenantAndUserFunc(
		ctx,
		tenantName,
		email,
		passwordHash,
	)
}

func TestRegisterHashesPassword(t *testing.T) {
	store := &fakeUserStore{
		findByEmailFunc: func(context.Context, string) (*models.User, error) {
			return nil, repository.ErrUserNotFound
		},
		createTenantAndUserFunc: func(
			_ context.Context,
			tenantName string,
			email string,
			passwordHash string,
		) (*models.User, error) {
			if tenantName != "Acme" {
				t.Fatalf("expected tenant Acme, got %q", tenantName)
			}

			if email != "joe@example.com" {
				t.Fatalf("expected normalized email, got %q", email)
			}

			if passwordHash == "secret123" {
				t.Fatal("password was stored in plaintext")
			}

			user := &models.User{
				ID:           "user-id",
				TenantID:     "tenant-id",
				Email:        email,
				PasswordHash: passwordHash,
			}

			return user, nil
		},
	}

	service := NewAuthService(store)

	user, err := service.Register(
		context.Background(),
		"Acme",
		" JOE@EXAMPLE.COM ",
		"secret123",
	)
	if err != nil {
		t.Fatalf("Register() returned error: %v", err)
	}

	if user.PasswordHash != "" {
		t.Fatal("expected password hash to be removed from returned user")
	}
}

func TestRegisterRejectsInvalidPassword(t *testing.T) {
	store := &fakeUserStore{
		findByEmailFunc: func(context.Context, string) (*models.User, error) {
			return nil, errors.New("should not be called")
		},
		createTenantAndUserFunc: func(
			context.Context,
			string,
			string,
			string,
		) (*models.User, error) {
			return nil, errors.New("should not be called")
		},
	}

	service := NewAuthService(store)

	_, err := service.Register(
		context.Background(),
		"Acme",
		"joe@example.com",
		"short",
	)

	if !errors.Is(err, ErrInvalidRegistration) {
		t.Fatalf("expected ErrInvalidRegistration, got %v", err)
	}
}

func TestLoginSuccess(t *testing.T) {
	const password = "secret123"

	hash, err := bcrypt.GenerateFromPassword(
		[]byte(password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		t.Fatalf("failed to create test hash: %v", err)
	}

	store := &fakeUserStore{
		findByEmailFunc: func(
			_ context.Context,
			email string,
		) (*models.User, error) {
			if email != "joe@example.com" {
				t.Fatalf("expected normalized email, got %q", email)
			}

			return &models.User{
				ID:           "user-id",
				TenantID:     "tenant-id",
				Email:        email,
				PasswordHash: string(hash),
			}, nil
		},
		createTenantAndUserFunc: func(
			context.Context,
			string,
			string,
			string,
		) (*models.User, error) {
			return nil, errors.New("should not be called during login")
		},
	}

	service := NewAuthService(store)

	user, err := service.Login(
		context.Background(),
		" JOE@EXAMPLE.COM ",
		password,
	)
	if err != nil {
		t.Fatalf("Login() returned error: %v", err)
	}

	if user.Email != "joe@example.com" {
		t.Fatalf("expected normalized email, got %q", user.Email)
	}

	if user.PasswordHash != "" {
		t.Fatal("expected password hash to be removed from returned user")
	}
}

func TestLoginRejectsWrongPassword(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword(
		[]byte("correct-password"),
		bcrypt.DefaultCost,
	)
	if err != nil {
		t.Fatalf("failed to create test hash: %v", err)
	}

	store := &fakeUserStore{
		findByEmailFunc: func(
			context.Context,
			string,
		) (*models.User, error) {
			return &models.User{
				ID:           "user-id",
				TenantID:     "tenant-id",
				Email:        "joe@example.com",
				PasswordHash: string(hash),
			}, nil
		},
		createTenantAndUserFunc: func(
			context.Context,
			string,
			string,
			string,
		) (*models.User, error) {
			return nil, errors.New("should not be called during login")
		},
	}

	service := NewAuthService(store)

	_, err = service.Login(
		context.Background(),
		"joe@example.com",
		"wrong-password",
	)

	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestLoginRejectsUnknownUser(t *testing.T) {
	store := &fakeUserStore{
		findByEmailFunc: func(
			context.Context,
			string,
		) (*models.User, error) {
			return nil, repository.ErrUserNotFound
		},
		createTenantAndUserFunc: func(
			context.Context,
			string,
			string,
			string,
		) (*models.User, error) {
			return nil, errors.New("should not be called during login")
		},
	}

	service := NewAuthService(store)

	_, err := service.Login(
		context.Background(),
		"unknown@example.com",
		"secret123",
	)

	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestRegisterMapsDuplicateEmailError(t *testing.T) {
	store := &fakeUserStore{
		findByEmailFunc: func(
			context.Context,
			string,
		) (*models.User, error) {
			return nil, repository.ErrUserNotFound
		},
		createTenantAndUserFunc: func(
			context.Context,
			string,
			string,
			string,
		) (*models.User, error) {
			return nil, repository.ErrEmailAlreadyExists
		},
	}

	service := NewAuthService(store)

	_, err := service.Register(
		context.Background(),
		"NezWidget Demo",
		"joe@example.com",
		"DemoPass123!",
	)

	if !errors.Is(err, ErrEmailAlreadyExists) {
		t.Fatalf(
			"expected ErrEmailAlreadyExists, got %v",
			err,
		)
	}
}
