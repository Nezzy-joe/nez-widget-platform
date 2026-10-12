package services

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Nezzy-joe/nez-widget-platform/internal/models"
)

const testJWTSecret = "test-secret-that-is-at-least-32-bytes-long"

func TestJWTServiceGeneratesAndVerifiesToken(t *testing.T) {
	service, err := NewJWTService(testJWTSecret)
	if err != nil {
		t.Fatalf("NewJWTService() returned error: %v", err)
	}

	user := &models.User{
		ID:       "user-123",
		TenantID: "tenant-456",
		Email:    "joe@example.com",
	}

	tokenString, err := service.Generate(user)
	if err != nil {
		t.Fatalf("Generate() returned error: %v", err)
	}

	claims, err := service.Verify(tokenString)
	if err != nil {
		t.Fatalf("Verify() returned error: %v", err)
	}

	if claims.Subject != user.ID {
		t.Fatalf("expected subject %q, got %q", user.ID, claims.Subject)
	}

	if claims.TenantID != user.TenantID {
		t.Fatalf(
			"expected tenant ID %q, got %q",
			user.TenantID,
			claims.TenantID,
		)
	}

	if claims.Issuer != tokenIssuer {
		t.Fatalf("expected issuer %q, got %q", tokenIssuer, claims.Issuer)
	}

	if claims.ExpiresAt == nil {
		t.Fatal("expected token expiration")
	}

	if !claims.ExpiresAt.Time.After(time.Now()) {
		t.Fatal("expected token expiration to be in the future")
	}
}

func TestJWTServiceRejectsTamperedToken(t *testing.T) {
	service, err := NewJWTService(testJWTSecret)
	if err != nil {
		t.Fatalf("NewJWTService() returned error: %v", err)
	}

	tokenString, err := service.Generate(&models.User{
		ID:       "user-123",
		TenantID: "tenant-456",
	})
	if err != nil {
		t.Fatalf("Generate() returned error: %v", err)
	}

	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		t.Fatal("expected a JWT with three parts")
	}

	// Replace the signature to simulate a modified token.
	parts[2] = "invalidsignature"
	tamperedToken := strings.Join(parts, ".")

	_, err = service.Verify(tamperedToken)
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}

func TestJWTServiceRejectsWrongSecret(t *testing.T) {
	issuer, err := NewJWTService(testJWTSecret)
	if err != nil {
		t.Fatalf("NewJWTService() returned error: %v", err)
	}

	wrongVerifier, err := NewJWTService(
		"another-test-secret-that-is-also-long-enough",
	)
	if err != nil {
		t.Fatalf("NewJWTService() returned error: %v", err)
	}

	tokenString, err := issuer.Generate(&models.User{
		ID:       "user-123",
		TenantID: "tenant-456",
	})
	if err != nil {
		t.Fatalf("Generate() returned error: %v", err)
	}

	_, err = wrongVerifier.Verify(tokenString)
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}

func TestJWTServiceRejectsShortSecret(t *testing.T) {
	_, err := NewJWTService("too-short")

	if !errors.Is(err, ErrInvalidJWTSecret) {
		t.Fatalf("expected ErrInvalidJWTSecret, got %v", err)
	}
}

func TestJWTServiceRejectsMissingIdentity(t *testing.T) {
	service, err := NewJWTService(testJWTSecret)
	if err != nil {
		t.Fatalf("NewJWTService() returned error: %v", err)
	}

	_, err = service.Generate(&models.User{
		ID:       "user-123",
		TenantID: "",
	})

	if err == nil {
		t.Fatal("expected Generate() to reject a missing tenant ID")
	}
}
