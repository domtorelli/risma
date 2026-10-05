package mail

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

const createLogQuery = `
	INSERT INTO email_logs (recipient_email, email_type, subject, body, status)
	VALUES ($1, $2, $3, $4, $5)`

type Repository struct{ pool *pgxpool.Pool }

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// CreateLog registra l'esito di un invio; sent_at è l'ora della scrittura,
// cioè subito dopo il tentativo.
func (r *Repository) CreateLog(ctx context.Context, e logEntry) error {
	_, err := r.pool.Exec(ctx, createLogQuery, e.Recipient, e.Type, e.Subject, e.Body, e.Status)
	if err != nil {
		return fmt.Errorf("creating email log: %w", err)
	}

	return nil
}
