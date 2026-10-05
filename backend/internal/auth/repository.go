package auth

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Ogni query scrive tutto ciò che riguarda un evento in una sola istruzione
// (CTE con più scritture): l'istruzione è atomica, quindi il log di accesso,
// il contatore dei tentativi e la sessione non possono restare a metà.

// recordFailureQuery registra un tentativo fallito e aggiorna il contatore,
// bloccando l'account al raggiungimento di $3 tentativi. Il contatore si
// ferma al massimo, così i tentativi su un account bloccato non lo fanno
// crescere senza limite.
const recordFailureQuery = `
	WITH attempt AS (
		INSERT INTO access_logs (user_id, result, ip_address)
		VALUES ($1, 'failure', $2)
	)
	UPDATE users
	SET failed_login_attempts = LEAST(failed_login_attempts + 1, $3),
		status = CASE
			WHEN failed_login_attempts + 1 >= $3 THEN 'blocked'::user_status
			ELSE status
		END
	WHERE id = $1
	RETURNING status`

// recordLoginQuery registra un accesso riuscito: azzera i tentativi falliti,
// aggiorna l'ultimo login e apre la sessione salvando l'hash del refresh token.
const recordLoginQuery = `
	WITH attempt AS (
		INSERT INTO access_logs (user_id, result, ip_address)
		VALUES ($1, 'success', $2)
	),
	session AS (
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $3, $4)
	)
	UPDATE users
	SET failed_login_attempts = 0, last_login_at = now()
	WHERE id = $1
	RETURNING last_login_at`

// rotateQuery revoca il refresh token $1 e ne inserisce uno nuovo ($2) per
// lo stesso utente. Il vecchio token è valido solo se non revocato, non
// scaduto e l'utente è ancora attivo e non disattivato. Con due rotazioni
// concorrenti dello stesso token solo la prima trova revoked_at NULL.
const rotateQuery = `
	WITH old AS (
		UPDATE refresh_tokens rt
		SET revoked_at = now()
		FROM users u
		WHERE rt.token_hash = $1
			AND rt.revoked_at IS NULL
			AND rt.expires_at > now()
			AND u.id = rt.user_id
			AND u.deleted_at IS NULL
			AND u.status = 'active'
		RETURNING rt.user_id
	)
	INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
	SELECT user_id, $2, $3 FROM old
	RETURNING user_id`

const revokeQuery = `
	UPDATE refresh_tokens
	SET revoked_at = now()
	WHERE token_hash = $1 AND revoked_at IS NULL`

// createResetQuery fa scadere i link di reset ancora validi dell'utente e
// ne inserisce uno nuovo: resta valido solo l'ultimo link inviato. I link
// precedenti si fanno scadere invece di segnarli come usati, perché non lo
// sono stati.
const createResetQuery = `
	WITH expired AS (
		UPDATE password_resets
		SET expires_at = now()
		WHERE user_id = $1 AND used_at IS NULL AND expires_at > now()
	)
	INSERT INTO password_resets (user_id, token_hash, expires_at)
	VALUES ($1, $2, $3)`

// resetPasswordQuery consuma un token di reset valido e, nella stessa
// istruzione, imposta la nuova password, sblocca l'account e revoca tutti
// i refresh token dell'utente (chi conosceva la vecchia password perde
// l'accesso). Con due richieste concorrenti con lo stesso token solo la
// prima trova used_at NULL.
const resetPasswordQuery = `
	WITH reset AS (
		UPDATE password_resets pr
		SET used_at = now()
		FROM users u
		WHERE pr.token_hash = $1
			AND pr.used_at IS NULL
			AND pr.expires_at > now()
			AND u.id = pr.user_id
			AND u.deleted_at IS NULL
		RETURNING pr.user_id
	),
	revoked AS (
		UPDATE refresh_tokens
		SET revoked_at = now()
		WHERE user_id IN (SELECT user_id FROM reset) AND revoked_at IS NULL
	)
	UPDATE users
	SET password_hash = $2, failed_login_attempts = 0, status = 'active'
	WHERE id IN (SELECT user_id FROM reset)
	RETURNING id`

type Repository struct{ pool *pgxpool.Pool }

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// RecordFailure registra un tentativo di login fallito e restituisce lo
// stato dell'utente dopo l'aggiornamento.
func (r *Repository) RecordFailure(ctx context.Context, userID string, ip netip.Addr, maxAttempts int) (string, error) {
	var status string

	err := r.pool.QueryRow(ctx, recordFailureQuery, userID, ip, maxAttempts).Scan(&status)
	if err != nil {
		return "", fmt.Errorf("recording failed login: %w", err)
	}

	return status, nil
}

// RecordLogin registra un login riuscito, salva l'hash del refresh token e
// restituisce l'ora del login.
func (r *Repository) RecordLogin(ctx context.Context, userID string, ip netip.Addr, tokenHash string, expiresAt time.Time) (time.Time, error) {
	var loggedInAt time.Time

	err := r.pool.QueryRow(ctx, recordLoginQuery, userID, ip, tokenHash, expiresAt).Scan(&loggedInAt)
	if err != nil {
		return time.Time{}, fmt.Errorf("recording login: %w", err)
	}

	return loggedInAt, nil
}

// RotateRefreshToken sostituisce un refresh token valido con uno nuovo e
// restituisce l'id dell'utente. Restituisce ErrInvalidToken se il vecchio
// token non esiste, è scaduto, revocato o l'utente non è più attivo.
func (r *Repository) RotateRefreshToken(ctx context.Context, oldHash, newHash string, expiresAt time.Time) (string, error) {
	var userID string

	err := r.pool.QueryRow(ctx, rotateQuery, oldHash, newHash, expiresAt).Scan(&userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrInvalidToken
		}

		return "", fmt.Errorf("rotating refresh token: %w", err)
	}

	return userID, nil
}

// RevokeRefreshToken revoca un refresh token; un token sconosciuto o già
// revocato non è un errore.
func (r *Repository) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	if _, err := r.pool.Exec(ctx, revokeQuery, tokenHash); err != nil {
		return fmt.Errorf("revoking refresh token: %w", err)
	}

	return nil
}

// CreatePasswordReset salva l'hash di un nuovo token di reset e invalida
// quelli precedenti dell'utente.
func (r *Repository) CreatePasswordReset(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error {
	if _, err := r.pool.Exec(ctx, createResetQuery, userID, tokenHash, expiresAt); err != nil {
		return fmt.Errorf("creating password reset: %w", err)
	}

	return nil
}

// ResetPassword consuma il token di reset e imposta la nuova password.
// Restituisce ErrInvalidToken se il token non esiste, è scaduto, è già
// stato usato o l'utente è stato disattivato.
func (r *Repository) ResetPassword(ctx context.Context, tokenHash, passwordHash string) error {
	var userID string

	err := r.pool.QueryRow(ctx, resetPasswordQuery, tokenHash, passwordHash).Scan(&userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrInvalidToken
		}

		return fmt.Errorf("resetting password: %w", err)
	}

	return nil
}
