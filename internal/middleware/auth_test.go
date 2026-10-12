package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Nezzy-joe/nez-widget-platform/internal/models"
	"github.com/Nezzy-joe/nez-widget-platform/internal/services"
)

const middlewareTestSecret = "middleware-test-secret-at-least-32-bytes"

func TestRequireAuthRejectsMissingToken(t *testing.T) {
	jwtService, err := services.NewJWTService(middlewareTestSecret)
	if err != nil {
		t.Fatalf("NewJWTService() returned error: %v", err)
	}

	protected := RequireAuth(jwtService)(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("protected handler must not run without a token")
		}),
	)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	rec := httptest.NewRecorder()

	protected.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusUnauthorized,
			rec.Code,
		)
	}
}

func TestRequireAuthProvidesVerifiedIdentity(t *testing.T) {
	jwtService, err := services.NewJWTService(middlewareTestSecret)
	if err != nil {
		t.Fatalf("NewJWTService() returned error: %v", err)
	}

	token, err := jwtService.Generate(&models.User{
		ID:       "test-user",
		TenantID: "test-tenant",
	})
	if err != nil {
		t.Fatalf("Generate() returned error: %v", err)
	}

	protected := RequireAuth(jwtService)(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			identity, ok := IdentityFromContext(r.Context())
			if !ok {
				t.Fatal("expected identity in request context")
			}

			if identity.UserID != "test-user" {
				t.Errorf("unexpected user ID: %q", identity.UserID)
			}

			if identity.TenantID != "test-tenant" {
				t.Errorf("unexpected tenant ID: %q", identity.TenantID)
			}

			w.WriteHeader(http.StatusNoContent)
		}),
	)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	rec := httptest.NewRecorder()

	protected.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusNoContent,
			rec.Code,
		)
	}
}

func TestRequireAuthRejectsInvalidToken(t *testing.T) {
	jwtService, err := services.NewJWTService(middlewareTestSecret)
	if err != nil {
		t.Fatalf("NewJWTService() returned error: %v", err)
	}

	protected := RequireAuth(jwtService)(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("protected handler must not run for an invalid token")
		}),
	)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer not-a-valid-jwt")

	rec := httptest.NewRecorder()

	protected.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusUnauthorized,
			rec.Code,
		)
	}
}
