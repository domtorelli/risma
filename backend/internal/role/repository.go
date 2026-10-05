package role

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("role not found")

// roleColumns elenca le colonne nello stesso ordine in cui scanRole le legge.
const roleColumns = `id, name, description, created_at, updated_at`

const findByNameQuery = `
	SELECT ` + roleColumns + `
	FROM roles
	WHERE name = $1`

const listQuery = `
	SELECT ` + roleColumns + `
	FROM roles
	ORDER BY name`

type Repository struct{ pool *pgxpool.Pool }

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// scanRole legge una riga con le colonne di roleColumns.
func scanRole(row pgx.Row) (Role, error) {
	var r Role

	err := row.Scan(
		&r.ID,
		&r.Name,
		&r.Description,
		&r.CreatedAt,
		&r.UpdatedAt,
	)
	if err != nil {
		return Role{}, err
	}

	return r, nil
}

// FindByName cerca un ruolo per nome (es. Admin, Operator).
func (r *Repository) FindByName(ctx context.Context, name string) (Role, error) {
	found, err := scanRole(r.pool.QueryRow(ctx, findByNameQuery, name))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Role{}, ErrNotFound
		}

		return Role{}, fmt.Errorf("finding role by name: %w", err)
	}

	return found, nil
}

// List restituisce tutti i ruoli ordinati per nome.
func (r *Repository) List(ctx context.Context) ([]Role, error) {
	rows, err := r.pool.Query(ctx, listQuery)
	if err != nil {
		return nil, fmt.Errorf("listing roles: %w", err)
	}

	roles, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (Role, error) {
		return scanRole(row)
	})
	if err != nil {
		return nil, fmt.Errorf("reading roles: %w", err)
	}

	return roles, nil
}
