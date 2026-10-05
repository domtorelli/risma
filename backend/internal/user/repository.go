package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// uniqueViolation è il codice SQLSTATE di PostgreSQL per un vincolo UNIQUE violato.
const uniqueViolation = "23505"

var (
	ErrNotFound      = errors.New("user not found")
	ErrAlreadyExists = errors.New("user already exists")
)

// userColumns elenca le colonne nello stesso ordine in cui scanUser le legge.
const userColumns = `id, role_id, username, email, password_hash,
	first_name, last_name, date_of_birth,
	failed_login_attempts, status, last_login_at,
	created_at, updated_at, deleted_at`

const findByIDQuery = `
	SELECT ` + userColumns + `
	FROM users
	WHERE id = $1 AND deleted_at IS NULL`

const findByUsernameQuery = `
	SELECT ` + userColumns + `
	FROM users
	WHERE lower(username) = lower($1) AND deleted_at IS NULL`

const createQuery = `
	INSERT INTO users (role_id, username, email, password_hash, first_name, last_name, date_of_birth)
	VALUES ($1, $2, $3, $4, $5, $6, $7)
	RETURNING ` + userColumns

type Repository struct{ pool *pgxpool.Pool }

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// scanUser legge una riga con le colonne di userColumns.
func scanUser(row pgx.Row) (User, error) {
	var u User

	err := row.Scan(
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
		return User{}, err
	}

	return u, nil
}

// FindByID cerca un utente non disattivato.
func (r *Repository) FindByID(ctx context.Context, id string) (User, error) {
	u, err := scanUser(r.pool.QueryRow(ctx, findByIDQuery, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrNotFound
		}

		return User{}, fmt.Errorf("finding user by id: %w", err)
	}

	return u, nil
}

// FindByUsername cerca un utente non disattivato, senza distinguere maiuscole e minuscole.
func (r *Repository) FindByUsername(ctx context.Context, username string) (User, error) {
	u, err := scanUser(r.pool.QueryRow(ctx, findByUsernameQuery, username))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrNotFound
		}

		return User{}, fmt.Errorf("finding user by username: %w", err)
	}

	return u, nil
}

// Create inserisce un utente e restituisce la riga creata.
// Usa i campi RoleID, Username, Email, PasswordHash, FirstName, LastName e DateOfBirth.
func (r *Repository) Create(ctx context.Context, u User) (User, error) {
	created, err := scanUser(r.pool.QueryRow(ctx, createQuery,
		u.RoleID,
		u.Username,
		u.Email,
		u.PasswordHash,
		u.FirstName,
		u.LastName,
		u.DateOfBirth,
	))
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
			return User{}, ErrAlreadyExists
		}

		return User{}, fmt.Errorf("creating user: %w", err)
	}

	return created, nil
}
