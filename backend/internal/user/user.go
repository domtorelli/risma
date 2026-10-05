package user

import "time"

// Valori del tipo ENUM user_status.
const (
	StatusActive  = "active"
	StatusBlocked = "blocked" // dopo troppi tentativi di login falliti
)

type User struct {
	ID                  string
	RoleID              string
	RoleName            string // letto con una JOIN su roles, non è una colonna di users
	Username            string
	Email               string
	PasswordHash        string
	FirstName           string
	LastName            string
	DateOfBirth         time.Time
	FailedLoginAttempts int
	Status              string
	LastLoginAt         *time.Time
	CreatedAt           time.Time
	UpdatedAt           time.Time
	DeletedAt           *time.Time
}
