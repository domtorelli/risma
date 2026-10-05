package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("user not found")

const findByIDQuery = `
	SELECT id, role_id, username, email, password_hash,
	       first_name, last_name, date_of_birth,
	       failed_login_attempts, status, last_login_at,
	       created_at, updated_at, deleted_at
	FROM users
	WHERE id = $1 AND deleted_at IS NULL`

type Repository struct{ pool *pgxpool.Pool }

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// FindByID cerca un utente non disattivato.
func (r *Repository) FindByID(ctx context.Context, id string) (User, error) {
	var u User

	err := r.pool.QueryRow(ctx, findByIDQuery, id).Scan(
		&u.ID,
		&u.RoleID,
		&u.Username,
		&u.Email,
		&u.PasswordHash,
		&u.FirstName,
		&u.LastName,
		&u.DateOfBirth,
		&u.FailedLoginAttempts,
		&u.Status,
		&u.LastLoginAt,
		&u.CreatedAt,
		&u.UpdatedAt,
		&u.DeletedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrNotFound
		}

		return User{}, fmt.Errorf("finding user by id: %w", err)
	}

	return u, nil
}
