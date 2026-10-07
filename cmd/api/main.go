package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
)

type HealthResponse struct {
	Status string `json:"status"`
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_ = json.NewEncoder(w).Encode(HealthResponse{
			Status: "ok",
		})
	})

	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	log.Printf("API server listening on http://localhost:%s", port)

	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
