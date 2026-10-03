package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/matteotorelli/risma/backend/internal/config"
	"github.com/matteotorelli/risma/backend/internal/database"
)

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	response := map[string]string{"status": "ok"}

	json.NewEncoder(w).Encode(response)
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("loading config: %v", err)
	}

	ctx := context.Background()

	db, err := database.Connect(ctx, cfg)
	if err != nil {
		log.Fatalf("connecting to database: %v", err)
	}
	defer db.Close()

	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", healthHandler)

	if err := http.ListenAndServe(":"+cfg.HTTPPort, mux); err != nil {
		log.Fatal(err)
	}
}
