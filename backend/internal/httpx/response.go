package httpx

import (
	"encoding/json"
	"log"
	"net/http"
)

// JSON scrive data come corpo JSON con lo status indicato.
func JSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if data != nil {
		err := json.NewEncoder(w).Encode(data)
		if err != nil {
			log.Printf("encoding JSON response: %v", err)
		}
	}
}

// Error scrive un errore nel formato {"error": "messaggio"}.
func Error(w http.ResponseWriter, status int, message string) {
	JSON(w, status, map[string]string{"error": message})
}
