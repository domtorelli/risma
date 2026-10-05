package user

import "time"

type User struct {
	ID                  string
	RoleID              string
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
