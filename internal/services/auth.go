package services

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"

	"github.com/Nezzy-joe/nez-widget-platform/internal/models"
	"github.com/Nezzy-joe/nez-widget-platform/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidRegistration = errors.New("invalid registration input")
	ErrInvalidCredentials  = errors.New("invalid email or password")
	ErrEmailAlreadyExists  = errors.New("email already exists")
)

type UserStore interface {
	FindByEmail(ctx context.Context, email string) (*models.User, error)

	CreateTenantAndUser(
		ctx context.Context,
		tenantName string,
		email string,
		passwordHash string,
	) (*models.User, error)
}

type AuthService struct {
	users UserStore
}

func NewAuthService(users UserStore) *AuthService {
	return &AuthService{
		users: users,
	}
}

// Registration Function
func (s *AuthService) Register(
	ctx context.Context,
	tenantName string,
	email string,
	password string,
) (*models.User, error) {
	tenantName = strings.TrimSpace(tenantName)
	email = normalizeEmail(email)

	if err := validateRegistration(tenantName, email, password); err != nil {
		return nil, err
	}

	passwordHash, err := bcrypt.GenerateFromPassword(
		[]byte(password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user, err := s.users.CreateTenantAndUser(
		ctx,
		tenantName,
		email,
		string(passwordHash),
	)
	if err != nil {
		if errors.Is(err, repository.ErrEmailAlreadyExists) {
			return nil, ErrEmailAlreadyExists
		}

		return nil, fmt.Errorf("register user: %w", err)
	}

	user.PasswordHash = ""

	return user, nil
}

// Login Function
func (s *AuthService) Login(
	ctx context.Context,
	email string,
	password string,
) (*models.User, error) {
	email = normalizeEmail(email)

	if email == "" || password == "" {
		return nil, ErrInvalidCredentials
	}

	user, err := s.users.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}

		return nil, fmt.Errorf("find user for login: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword(
		[]byte(user.PasswordHash),
		[]byte(password),
	); err != nil {
		return nil, ErrInvalidCredentials
	}

	user.PasswordHash = ""

	return user, nil
}

func validateRegistration(
	tenantName string,
	email string,
	password string,
) error {
	if tenantName == "" || len([]rune(tenantName)) > 100 {
		return ErrInvalidRegistration
	}

	parsedEmail, err := mail.ParseAddress(email)
	if err != nil || parsedEmail.Address != email {
		return ErrInvalidRegistration
	}

	// bcrypt accepts passwords up to 72 bytes.
	if len([]byte(password)) < 8 || len([]byte(password)) > 72 {
		return ErrInvalidRegistration
	}

	return nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
