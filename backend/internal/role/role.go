package role

import "time"

// Nomi dei ruoli inseriti dalla migrazione 00010_seed_roles.sql.
// Il codice cerca i ruoli per nome, mai per UUID.
const (
	Admin    = "admin"
	Operator = "operatore"
)

type Role struct {
	ID          string
	Name        string
	Description *string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
