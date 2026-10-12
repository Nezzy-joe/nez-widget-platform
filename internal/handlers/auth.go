package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"mime"
	"net/http"
	"time"

	"github.com/Nezzy-joe/nez-widget-platform/internal/middleware"
	"github.com/Nezzy-joe/nez-widget-platform/internal/models"
	"github.com/Nezzy-joe/nez-widget-platform/internal/services"
)

const maxAuthBodyBytes = 16 << 10 // 16 KiB

var (
	errUnsupportedMediaType = errors.New("Content-Type must be application/json")
	errMultipleJSONValues   = errors.New("request must contain one JSON object")
)

type authOperations interface {
	Register(
		ctx context.Context,
		tenantName string,
		email string,
		password string,
	) (*models.User, error)

	Login(
		ctx context.Context,
		email string,
		password string,
	) (*models.User, error)
}

type tokenGenerator interface {
	Generate(user *models.User) (string, error)
}

type AuthHandler struct {
	auth   authOperations
	tokens tokenGenerator
}

func NewAuthHandler(auth authOperations, tokens tokenGenerator) *AuthHandler {
	return &AuthHandler{
		auth:   auth,
		tokens: tokens,
	}
}

type registerRequest struct {
	TenantName string `json:"tenant_name"`
	Email      string `json:"email"`
	Password   string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type safeUserResponse struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func safeUser(user *models.User) safeUserResponse {
	return safeUserResponse{
		ID:        user.ID,
		TenantID:  user.TenantID,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}

	var input registerRequest
	if err := decodeJSON(w, r, &input); err != nil {
		writeDecodeError(w, err)
		return
	}

	user, err := h.auth.Register(
		r.Context(),
		input.TenantName,
		input.Email,
		input.Password,
	)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrInvalidRegistration):
			writeError(w, http.StatusBadRequest, "invalid registration details")
		case errors.Is(err, services.ErrEmailAlreadyExists):
			writeError(w, http.StatusConflict, "email already registered")
		default:
			log.Printf("registration failed: %v", err)
			writeError(w, http.StatusInternalServerError, "registration failed")
		}
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"user": safeUser(user),
	})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}

	var input loginRequest
	if err := decodeJSON(w, r, &input); err != nil {
		writeDecodeError(w, err)
		return
	}

	user, err := h.auth.Login(r.Context(), input.Email, input.Password)
	if err != nil {
		if errors.Is(err, services.ErrInvalidCredentials) {
			writeError(w, http.StatusUnauthorized, "invalid email or password")
			return
		}

		log.Printf("login failed: %v", err)
		writeError(w, http.StatusInternalServerError, "login failed")
		return
	}

	token, err := h.tokens.Generate(user)
	if err != nil {
		log.Printf("token generation failed: %v", err)
		writeError(w, http.StatusInternalServerError, "login failed")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"access_token": token,
		"token_type":   "Bearer",
		"user":         safeUser(user),
	})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	identity, ok := middleware.IdentityFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"user_id":   identity.UserID,
		"tenant_id": identity.TenantID,
	})
}

func requireMethod(w http.ResponseWriter, r *http.Request, method string) bool {
	if r.Method == method {
		return true
	}

	w.Header().Set("Allow", method)
	writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	return false
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		return errUnsupportedMediaType
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxAuthBodyBytes)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		return err
	}

	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return errMultipleJSONValues
		}
		return err
	}

	return nil
}

func writeDecodeError(w http.ResponseWriter, err error) {
	var maxBytesErr *http.MaxBytesError

	switch {
	case errors.As(err, &maxBytesErr):
		writeError(w, http.StatusRequestEntityTooLarge, "request body too large")
	case errors.Is(err, errUnsupportedMediaType):
		writeError(w, http.StatusUnsupportedMediaType, errUnsupportedMediaType.Error())
	default:
		writeError(w, http.StatusBadRequest, "invalid JSON request")
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{
		"error": message,
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
