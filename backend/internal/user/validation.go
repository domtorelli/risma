package user

import (
	"net/mail"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/matteotorelli/risma/backend/internal/validation"
)

const (
	maxEmailBytes = 254 // RFC 5321, pari alla colonna users.email
	maxNameRunes  = 100
)

var (
	usernamePattern = regexp.MustCompile(`^[A-Za-z0-9._-]{3,50}$`)
	minBirthDate    = time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC)
)

// profile sono i dati anagrafici e di accesso comuni a creazione e modifica.
type profile struct {
	Username    string
	Email       string
	FirstName   string
	LastName    string
	DateOfBirth time.Time
}

// normalize toglie gli spazi iniziali e finali dai campi di testo.
func (p profile) normalize() profile {
	p.Username = strings.TrimSpace(p.Username)
	p.Email = strings.TrimSpace(p.Email)
	p.FirstName = strings.TrimSpace(p.FirstName)
	p.LastName = strings.TrimSpace(p.LastName)
	return p
}

// validate controlla i campi già normalizzati; today è la data odierna,
// passata dal chiamante per rendere la regola verificabile nei test.
func (p profile) validate(today time.Time) error {
	var errs validation.Collector

	if !usernamePattern.MatchString(p.Username) {
		errs.Add("username", "da 3 a 50 caratteri tra lettere, numeri, punto, trattino e trattino basso")
	}

	if !validEmail(p.Email) {
		errs.Add("email", "indirizzo email non valido")
	}

	if msg := validateName(p.FirstName); msg != "" {
		errs.Add("first_name", msg)
	}

	if msg := validateName(p.LastName); msg != "" {
		errs.Add("last_name", msg)
	}

	switch {
	case p.DateOfBirth.IsZero():
		errs.Add("date_of_birth", "data di nascita obbligatoria, nel formato AAAA-MM-GG")
	case p.DateOfBirth.Before(minBirthDate) || !p.DateOfBirth.Before(today):
		errs.Add("date_of_birth", "la data di nascita deve essere successiva al 1900 e precedente a oggi")
	}

	return errs.Err()
}

// validEmail accetta solo un indirizzo semplice, senza nome visualizzato
// (es. "mario@example.com", non "Mario <mario@example.com>").
func validEmail(email string) bool {
	if email == "" || len(email) > maxEmailBytes {
		return false
	}

	addr, err := mail.ParseAddress(email)
	return err == nil && addr.Address == email
}

// validateName restituisce il messaggio d'errore per un nome o cognome,
// oppure una stringa vuota se è valido.
func validateName(name string) string {
	if name == "" {
		return "campo obbligatorio"
	}

	if utf8.RuneCountInString(name) > maxNameRunes {
		return "al massimo 100 caratteri"
	}

	if strings.ContainsFunc(name, unicode.IsControl) {
		return "contiene caratteri non ammessi"
	}

	return ""
}
