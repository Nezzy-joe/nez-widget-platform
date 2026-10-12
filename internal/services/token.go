package services

import (
	"errors"
	"strings"
	"time"

	"github.com/Nezzy-joe/nez-widget-platform/internal/models"
	"github.com/golang-jwt/jwt/v5"
)

const (
	tokenIssuer = "nez-widget-platform"
	tokenTTL    = 15 * time.Minute
)

var (
	ErrInvalidJWTSecret = errors.New("invalid JWT secret")
	ErrInvalidToken     = errors.New("invalid token")
)

type Claims struct {
	TenantID string `json:"tenant_id"`
	jwt.RegisteredClaims
}

type JWTService struct {
	secret []byte
	issuer string
	ttl    time.Duration
}

func NewJWTService(secret string) (*JWTService, error) {
	secret = strings.TrimSpace(secret)

	if len([]byte(secret)) < 32 {
		return nil, ErrInvalidJWTSecret
	}

	return &JWTService{
		secret: []byte(secret),
		issuer: tokenIssuer,
		ttl:    tokenTTL,
	}, nil
}

func (s *JWTService) Generate(user *models.User) (string, error) {
	if user == nil || user.ID == "" || user.TenantID == "" {
		return "", errors.New("user ID and tenant ID are required")
	}

	now := time.Now()

	claims := Claims{
		TenantID: user.TenantID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.issuer,
			Subject:   user.ID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.ttl)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signedToken, err := token.SignedString(s.secret)
	if err != nil {
		return "", err
	}

	return signedToken, nil
}

// Verify validates the signature and required claims before trusting the token.
func (s *JWTService) Verify(tokenString string) (*Claims, error) {
	if strings.TrimSpace(tokenString) == "" {
		return nil, ErrInvalidToken
	}

	claims := &Claims{}

	token, err := jwt.ParseWithClaims(
		tokenString,
		claims,
		func(token *jwt.Token) (interface{}, error) {
			// Never accept a different signing algorithm.
			if token.Method == nil ||
				token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
				return nil, ErrInvalidToken
			}

			return s.secret, nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(s.issuer),
		jwt.WithIssuedAt(),
	)
	if err != nil || token == nil || !token.Valid {
		return nil, ErrInvalidToken
	}

	// Require the identity and time claims  application depends on.
	if claims.Subject == "" ||
		claims.TenantID == "" ||
		claims.IssuedAt == nil ||
		claims.ExpiresAt == nil {
		return nil, ErrInvalidToken
	}

	return claims, nil
}
