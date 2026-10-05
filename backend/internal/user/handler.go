package user

import "time"

// Response è la rappresentazione JSON di un utente. I campi sono elencati
// esplicitamente: PasswordHash e i dati interni non escono mai dall'API.
type Response struct {
	ID          string     `json:"id"`
	RoleID      string     `json:"role_id"`
	RoleName    string     `json:"role_name"`
	Username    string     `json:"username"`
	Email       string     `json:"email"`
	FirstName   string     `json:"first_name"`
	LastName    string     `json:"last_name"`
	DateOfBirth string     `json:"date_of_birth"` // AAAA-MM-GG
	Status      string     `json:"status"`
	LastLoginAt *time.Time `json:"last_login_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func NewResponse(u User) Response {
	return Response{
		ID:          u.ID,
		RoleID:      u.RoleID,
		RoleName:    u.RoleName,
		Username:    u.Username,
		Email:       u.Email,
		FirstName:   u.FirstName,
		LastName:    u.LastName,
		DateOfBirth: u.DateOfBirth.Format(time.DateOnly),
		Status:      u.Status,
		LastLoginAt: u.LastLoginAt,
		CreatedAt:   u.CreatedAt,
		UpdatedAt:   u.UpdatedAt,
	}
}
