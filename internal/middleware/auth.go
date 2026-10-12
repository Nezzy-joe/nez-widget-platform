package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Nezzy-joe/nez-widget-platform/internal/services"
)

type Identity struct {
	UserID   string
	TenantID string
}

type identityContextKey struct{}

func IdentityFromContext(ctx context.Context) (Identity, bool) {
	identity, ok := ctx.Value(identityContextKey{}).(Identity)
	return identity, ok
}

func RequireAuth(jwtService *services.JWTService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			parts := strings.Fields(r.Header.Get("Authorization"))

			if len(parts) != 2 ||
				!strings.EqualFold(parts[0], "Bearer") {
				writeUnauthorized(w)
				return
			}

			claims, err := jwtService.Verify(parts[1])
			if err != nil {
				writeUnauthorized(w)
				return
			}

			identity := Identity{
				UserID:   claims.Subject,
				TenantID: claims.TenantID,
			}

			ctx := context.WithValue(
				r.Context(),
				identityContextKey{},
				identity,
			)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func writeUnauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)

	_ = json.NewEncoder(w).Encode(map[string]string{
		"error": "authentication required",
	})
}
