package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/matteotorelli/risma/backend/internal/config"
)

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	response := map[string]string{"status": "ok"}

	json.NewEncoder(w).Encode(response)
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", healthHandler)

	if err := http.ListenAndServe(":"+cfg.HTTPPort, mux); err != nil {
		log.Fatal(err)
	}
}
