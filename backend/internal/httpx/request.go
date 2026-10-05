package httpx

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// maxBodyBytes limita la dimensione del corpo JSON accettato: le API
// ricevono solo piccoli oggetti, un corpo più grande è un errore o un abuso.
const maxBodyBytes = 1 << 20 // 1 MiB

var ErrInvalidBody = errors.New("invalid request body")

// DecodeJSON legge il corpo della richiesta in dst. Rifiuta corpi oltre
// maxBodyBytes, campi sconosciuti e dati dopo il primo oggetto JSON.
// Gli errori restituiti avvolgono ErrInvalidBody e il loro messaggio si può
// mostrare al client (non contiene dati interni).
func DecodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			return fmt.Errorf("%w: body too large", ErrInvalidBody)
		}

		if errors.Is(err, io.EOF) {
			return fmt.Errorf("%w: body is empty", ErrInvalidBody)
		}

		return fmt.Errorf("%w: %w", ErrInvalidBody, err)
	}

	if dec.More() {
		return fmt.Errorf("%w: body must contain a single JSON object", ErrInvalidBody)
	}

	return nil
}

// IsUUID indica se s è un UUID in forma testuale canonica
// (8-4-4-4-12 cifre esadecimali). Gli id ricevuti nei percorsi e nei corpi
// si controllano prima della query: un valore malformato è un id che non
// esiste, non un errore del database.
func IsUUID(s string) bool {
	if len(s) != 36 {
		return false
	}

	for i, c := range s {
		switch i {
		case 8, 13, 18, 23:
			if c != '-' {
				return false
			}
		default:
			isHex := (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
			if !isHex {
				return false
			}
		}
	}

	return true
}
