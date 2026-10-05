package user

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Codici SQLSTATE di PostgreSQL.
const (
	uniqueViolation     = "23505"
	foreignKeyViolation = "23503"
)

var (
	ErrNotFound      = errors.New("user not found")
	ErrAlreadyExists = errors.New("user already exists")
	ErrInvalidRole   = errors.New("role does not exist")
)

// userColumns elenca le colonne nello stesso ordine in cui scanUser le legge.
// Gli alias sono u (users, o la CTE che restituisce righe di users) e r (roles).
const userColumns = `u.id, u.role_id, r.name, u.username, u.email, u.password_hash,
	u.first_name, u.last_name, u.date_of_birth,
	u.failed_login_attempts, u.status, u.last_login_at,
	u.created_at, u.updated_at, u.deleted_at`

// joinRoles completa una SELECT di userColumns da una sorgente chiamata u.
const joinRoles = `JOIN roles r ON r.id = u.role_id`

const findByIDQuery = `
	SELECT ` + userColumns + `
	FROM users u ` + joinRoles + `
	WHERE u.id = $1 AND u.deleted_at IS NULL`

const findByUsernameQuery = `
	SELECT ` + userColumns + `
	FROM users u ` + joinRoles + `
	WHERE lower(u.username) = lower($1) AND u.deleted_at IS NULL`

const findByEmailQuery = `
	SELECT ` + userColumns + `
	FROM users u ` + joinRoles + `
	WHERE lower(u.email) = lower($1) AND u.deleted_at IS NULL`

// searchCondition filtra gli utenti attivi il cui username o email
// corrisponde al pattern ILIKE $1 ('%' per non filtrare).
const searchCondition = `
	u.deleted_at IS NULL AND (u.username ILIKE $1 OR u.email ILIKE $1)`

const listQuery = `
	SELECT ` + userColumns + `
	FROM users u ` + joinRoles + `
	WHERE ` + searchCondition + `
	ORDER BY u.created_at DESC, u.id DESC
	LIMIT $2 OFFSET $3`

const countQuery = `
	SELECT count(*)
	FROM users u
	WHERE ` + searchCondition

// Le scritture usano una CTE: la riga restituita da INSERT/UPDATE ... RETURNING
// viene unita a roles per leggere anche il nome del ruolo in un solo giro.
const createQuery = `
	WITH u AS (
		INSERT INTO users (role_id, username, email, password_hash, first_name, last_name, date_of_birth)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING *
	)
	SELECT ` + userColumns + `
	FROM u ` + joinRoles

const updateQuery = `
	WITH u AS (
		UPDATE users
		SET username = $2, email = $3, first_name = $4, last_name = $5, date_of_birth = $6
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING *
	)
	SELECT ` + userColumns + `
	FROM u ` + joinRoles

const updateRoleQuery = `
	WITH u AS (
		UPDATE users
		SET role_id = $2
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING *
	)
	SELECT ` + userColumns + `
	FROM u ` + joinRoles

const softDeleteQuery = `
	UPDATE users
	SET deleted_at = now()
	WHERE id = $1 AND deleted_at IS NULL`

// likeEscaper rende letterali i caratteri speciali di LIKE nel testo cercato,
// così "a_b" non corrisponde anche ad "axb". Il backslash è il carattere di
// escape predefinito di PostgreSQL.
var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// searchPattern trasforma il testo cercato in un pattern "contiene".
func searchPattern(search string) string {
	return "%" + likeEscaper.Replace(search) + "%"
}

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
		&u.RoleName,
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

// knownError traduce un errore di scrittura nell'errore del pacchetto che lo
// descrive, oppure restituisce nil se l'errore è imprevisto.
func knownError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case uniqueViolation:
			return ErrAlreadyExists
		case foreignKeyViolation:
			return ErrInvalidRole
		}
	}

	return nil
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

// FindByEmail cerca un utente non disattivato, senza distinguere maiuscole e minuscole.
func (r *Repository) FindByEmail(ctx context.Context, email string) (User, error) {
	u, err := scanUser(r.pool.QueryRow(ctx, findByEmailQuery, email))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrNotFound
		}

		return User{}, fmt.Errorf("finding user by email: %w", err)
	}

	return u, nil
}

// List restituisce una pagina di utenti non disattivati, dal più recente,
// il cui username o email contiene search (vuoto per non filtrare).
func (r *Repository) List(ctx context.Context, search string, limit, offset int) ([]User, error) {
	rows, err := r.pool.Query(ctx, listQuery, searchPattern(search), limit, offset)
	if err != nil {
		return nil, fmt.Errorf("listing users: %w", err)
	}

	users, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (User, error) {
		return scanUser(row)
	})
	if err != nil {
		return nil, fmt.Errorf("reading users: %w", err)
	}

	return users, nil
}

// Count conta gli utenti che List restituirebbe senza paginazione.
func (r *Repository) Count(ctx context.Context, search string) (int, error) {
	var total int

	err := r.pool.QueryRow(ctx, countQuery, searchPattern(search)).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("counting users: %w", err)
	}

	return total, nil
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
		if known := knownError(err); known != nil {
			return User{}, known
		}

		return User{}, fmt.Errorf("creating user: %w", err)
	}

	return created, nil
}

// Update modifica i dati anagrafici e di accesso di un utente non disattivato.
// Usa i campi ID, Username, Email, FirstName, LastName e DateOfBirth: il ruolo
// si cambia solo con UpdateRole, la password con il recupero password.
func (r *Repository) Update(ctx context.Context, u User) (User, error) {
	updated, err := scanUser(r.pool.QueryRow(ctx, updateQuery,
		u.ID,
		u.Username,
		u.Email,
		u.FirstName,
		u.LastName,
		u.DateOfBirth,
	))
	if err != nil {
		if known := knownError(err); known != nil {
			return User{}, known
		}

		return User{}, fmt.Errorf("updating user: %w", err)
	}

	return updated, nil
}

// UpdateRole assegna un ruolo a un utente non disattivato.
func (r *Repository) UpdateRole(ctx context.Context, id, roleID string) (User, error) {
	updated, err := scanUser(r.pool.QueryRow(ctx, updateRoleQuery, id, roleID))
	if err != nil {
		if known := knownError(err); known != nil {
			return User{}, known
		}

		return User{}, fmt.Errorf("updating user role: %w", err)
	}

	return updated, nil
}

// SoftDelete disattiva un utente impostando deleted_at; la riga resta per
// conservare le tracce nei log e nei movimenti di magazzino.
func (r *Repository) SoftDelete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, softDeleteQuery, id)
	if err != nil {
		return fmt.Errorf("deactivating user: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}
