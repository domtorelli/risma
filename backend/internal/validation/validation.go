// Il pacchetto validation raccoglie gli errori sui campi in ingresso, comuni
// a tutti i domini: i service li restituiscono, gli handler li traducono in
// una risposta 400 con un messaggio per ogni campo.
package validation

import (
	"maps"
	"slices"
	"strings"
)

// Error elenca i campi non validi; le chiavi sono i nomi JSON dei campi e
// i messaggi si possono mostrare all'utente.
type Error struct {
	Fields map[string]string
}

func (e *Error) Error() string {
	return "invalid fields: " + strings.Join(slices.Sorted(maps.Keys(e.Fields)), ", ")
}

// Field restituisce un errore su un solo campo.
func Field(name, message string) *Error {
	return &Error{Fields: map[string]string{name: message}}
}

// Collector accumula gli errori di più campi, così l'utente li vede tutti
// insieme invece di correggerli uno alla volta.
type Collector struct {
	fields map[string]string
}

// Add registra un errore sul campo; il primo errore di un campo prevale.
func (c *Collector) Add(field, message string) {
	if c.fields == nil {
		c.fields = map[string]string{}
	}

	if _, exists := c.fields[field]; !exists {
		c.fields[field] = message
	}
}

// Err restituisce un *Error con i campi raccolti, oppure nil se non ce ne sono.
func (c *Collector) Err() error {
	if len(c.fields) == 0 {
		return nil
	}

	return &Error{Fields: c.fields}
}
