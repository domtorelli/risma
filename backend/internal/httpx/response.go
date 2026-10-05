package httpx

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
)

// JSON scrive data come corpo JSON con lo status indicato.
//
// Il corpo viene serializzato prima di scrivere lo status: se la
// serializzazione fallisce è un errore di programmazione (tipo non
// serializzabile), quindi JSON va in panic e il middleware Recovery lo
// registra e risponde 500, invece di inviare uno status 200 con un corpo
// troncato. Un errore di scrittura significa invece che il client si è
// disconnesso e non c'è nulla da fare.
func JSON(w http.ResponseWriter, status int, data any) {
	if data == nil {
		w.WriteHeader(status)
		return
	}

	body, err := json.Marshal(data)
	if err != nil {
		panic(fmt.Sprintf("encoding JSON response: %v", err))
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

// Error scrive un errore nel formato {"error": "messaggio"}. Il messaggio
// viene mostrato all'utente, quindi non deve contenere dettagli interni.
func Error(w http.ResponseWriter, status int, message string) {
	JSON(w, status, map[string]string{"error": message})
}

// InternalError registra err nei log e risponde 500 con un messaggio
// generico: la causa resta nei log, il client non vede dettagli interni.
func InternalError(w http.ResponseWriter, r *http.Request, logger *slog.Logger, err error) {
	logger.Error("handling request", "method", r.Method, "path", r.URL.Path, "error", err)
	Error(w, http.StatusInternalServerError, "errore interno del server")
}
