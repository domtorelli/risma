package product

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/matteotorelli/risma/backend/internal/validation"
)

const (
	maxNameRunes        = 200
	maxDescriptionRunes = 1000
	maxNoteRunes        = 500

	// MaxQuantity limita giacenze, soglie e movimenti: abbondante per un
	// negozio e lontano dal massimo della colonna INT.
	MaxQuantity = 1_000_000
)

// pricePattern accetta i valori di NUMERIC(10, 2): fino a 8 cifre intere e
// 2 decimali, separatore punto (es. "12", "12.5", "12.50").
var pricePattern = regexp.MustCompile(`^\d{1,8}(\.\d{1,2})?$`)

// details sono i dati descrittivi comuni a creazione e modifica.
type details struct {
	TypeID            string
	Name              string
	Description       string
	Price             string
	MinStockThreshold int
}

// normalize toglie gli spazi iniziali e finali dai campi di testo.
func (d details) normalize() details {
	d.Name = strings.TrimSpace(d.Name)
	d.Description = strings.TrimSpace(d.Description)
	d.Price = strings.TrimSpace(d.Price)
	return d
}

// validate controlla i campi già normalizzati e registra gli errori in errs.
func (d details) validate(errs *validation.Collector) {
	if d.TypeID == "" {
		errs.Add("product_type_id", "scegli il tipo di prodotto")
	}

	switch {
	case d.Name == "":
		errs.Add("name", "campo obbligatorio")
	case utf8.RuneCountInString(d.Name) > maxNameRunes:
		errs.Add("name", "al massimo 200 caratteri")
	case strings.ContainsFunc(d.Name, unicode.IsControl):
		errs.Add("name", "contiene caratteri non ammessi")
	}

	if utf8.RuneCountInString(d.Description) > maxDescriptionRunes {
		errs.Add("description", "al massimo 1000 caratteri")
	}

	if !pricePattern.MatchString(d.Price) {
		errs.Add("price", "prezzo non valido: usa il punto per i decimali (es. 12.50), al massimo 2 decimali")
	}

	if d.MinStockThreshold < 0 || d.MinStockThreshold > MaxQuantity {
		errs.Add("min_stock_threshold", "valore tra 0 e 1.000.000")
	}
}

// optional restituisce nil per una stringa vuota, così nel database il
// campo facoltativo resta NULL invece di una stringa vuota.
func optional(s string) *string {
	if s == "" {
		return nil
	}

	return &s
}
