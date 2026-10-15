package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/Nezzy-joe/nez-widget-platform/internal/config"
	"github.com/Nezzy-joe/nez-widget-platform/internal/handlers"
	"github.com/Nezzy-joe/nez-widget-platform/internal/middleware"
	"github.com/Nezzy-joe/nez-widget-platform/internal/repository"
	"github.com/Nezzy-joe/nez-widget-platform/internal/services"
	"github.com/joho/godotenv"
)

type HealthResponse struct {
	Status   string `json:"status"`
	Database string `json:"database"`
}

func main() {
	if err := run(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

func run() error {
	// Local development only; deployed environments should inject variables.
	if err := godotenv.Load(); err != nil {
		log.Printf("warning: could not load .env: %v", err)
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	jwtService, err := services.NewJWTService(cfg.JWTSecret)
	if err != nil {
		return err
	}

	startupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	db, err := repository.NewPostgresPool(startupCtx, cfg.DatabaseURL)
	cancel()
	if err != nil {
		return err
	}
	defer db.Close()

	userRepository := repository.NewUserRepository(db)
	authService := services.NewAuthService(userRepository)
	authHandler := handlers.NewAuthHandler(authService, jwtService)

	widgetRepository := repository.NewWidgetRepository(db)
	widgetService := services.NewWidgetService(widgetRepository)
	widgetHandler := handlers.NewWidgetHandler(widgetService)

	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		healthCtx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		w.Header().Set("Content-Type", "application/json")

		if err := db.Ping(healthCtx); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(HealthResponse{
				Status:   "error",
				Database: "unavailable",
			})
			return
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(HealthResponse{
			Status:   "ok",
			Database: "ok",
		})
	})

	mux.HandleFunc("/auth/register", authHandler.Register)
	mux.HandleFunc("/auth/login", authHandler.Login)
	mux.Handle(
		"/auth/me",
		middleware.RequireAuth(jwtService)(http.HandlerFunc(authHandler.Me)),
	)
	mux.Handle(
		"/widgets",
		middleware.RequireAuth(jwtService)(
			http.HandlerFunc(widgetHandler.Collection),
		),
	)

	mux.Handle(
		"/widgets/",
		middleware.RequireAuth(jwtService)(
			http.HandlerFunc(widgetHandler.Item),
		),
	)

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("API server listening on http://localhost:%s", cfg.Port)

	return server.ListenAndServe()
}
