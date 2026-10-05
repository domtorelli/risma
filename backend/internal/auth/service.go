package auth

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/matteotorelli/risma/backend/internal/mail"
	"github.com/matteotorelli/risma/backend/internal/user"
)

const (
	// maxFailedAttempts è il numero di tentativi falliti dopo cui l'account
	// viene bloccato (scheda, §4.1).
	maxFailedAttempts = 5

	// PasswordResetTTL è la validità del link di recupero password (scheda, §4.1).
	PasswordResetTTL = time.Hour
)

var (
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrAccountBlocked     = errors.New("account blocked after too many failed login attempts")
	ErrInvalidToken       = errors.New("invalid or expired token")
)

type userFinder interface {
	FindByUsername(ctx context.Context, username string) (user.User, error)
	FindByEmail(ctx context.Context, email string) (user.User, error)
	FindByID(ctx context.Context, id string) (user.User, error)
}

type authStore interface {
	RecordFailure(ctx context.Context, userID string, ip netip.Addr, maxAttempts int) (string, error)
	RecordLogin(ctx context.Context, userID string, ip netip.Addr, tokenHash string, expiresAt time.Time) (time.Time, error)
	RotateRefreshToken(ctx context.Context, oldHash, newHash string, expiresAt time.Time) (string, error)
	RevokeRefreshToken(ctx context.Context, tokenHash string) error
	CreatePasswordReset(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error
	ResetPassword(ctx context.Context, tokenHash, passwordHash string) error
}

type mailer interface {
	SendAsync(ctx context.Context, msg mail.Message)
}

// Il compilatore verifica che i tipi concreti soddisfino le interfacce.
var (
	_ userFinder = (*user.Repository)(nil)
	_ authStore  = (*Repository)(nil)
	_ mailer     = (*mail.Service)(nil)
)

// Session contiene i token emessi da un login o da un rinnovo.
type Session struct {
	User             user.User
	AccessToken      string
	AccessExpiresAt  time.Time
	RefreshToken     string
	RefreshExpiresAt time.Time
}

type Service struct {
	users      userFinder
	store      authStore
	mailer     mailer
	tokens     tokenManager
	now        func() time.Time
	appBaseURL string

	// dummyHash si confronta quando lo username non esiste, così la risposta
	// richiede lo stesso tempo di una password sbagliata e il tempo non
	// rivela quali username esistono.
	dummyHash []byte
}

// NewService crea il service di autenticazione; appBaseURL è l'indirizzo
// del frontend usato nei link delle email.
func NewService(users userFinder, store authStore, mailer mailer, jwtSecret []byte, appBaseURL string) (*Service, error) {
	dummyHash, err := bcrypt.GenerateFromPassword([]byte(newRandomToken()), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("generating dummy hash: %w", err)
	}

	return &Service{
		users:      users,
		store:      store,
		mailer:     mailer,
		tokens:     tokenManager{secret: jwtSecret, now: time.Now},
		now:        time.Now,
		appBaseURL: appBaseURL,
		dummyHash:  dummyHash,
	}, nil
}

// Login verifica le credenziali e apre una sessione. Ogni tentativo su un
// utente esistente viene registrato in access_logs; al quinto tentativo
// fallito consecutivo l'account viene bloccato. Un account bloccato non
// verifica la password: si sblocca solo con il recupero password.
func (s *Service) Login(ctx context.Context, username, password string, ip netip.Addr) (Session, error) {
	u, err := s.users.FindByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			_ = bcrypt.CompareHashAndPassword(s.dummyHash, []byte(password))
			return Session{}, ErrInvalidCredentials
		}

		return Session{}, fmt.Errorf("logging in: %w", err)
	}

	if u.Status == user.StatusBlocked {
		if _, err := s.store.RecordFailure(ctx, u.ID, ip, maxFailedAttempts); err != nil {
			return Session{}, fmt.Errorf("logging in: %w", err)
		}

		return Session{}, ErrAccountBlocked
	}

	// Qualunque errore del confronto (password errata, hash non valido)
	// conta come tentativo fallito.
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		status, err := s.store.RecordFailure(ctx, u.ID, ip, maxFailedAttempts)
		if err != nil {
			return Session{}, fmt.Errorf("logging in: %w", err)
		}

		if status == user.StatusBlocked {
			return Session{}, ErrAccountBlocked
		}

		return Session{}, ErrInvalidCredentials
	}

	refreshToken := newRandomToken()
	refreshExpiresAt := s.now().Add(RefreshTokenTTL)

	loggedInAt, err := s.store.RecordLogin(ctx, u.ID, ip, hashToken(refreshToken), refreshExpiresAt)
	if err != nil {
		return Session{}, fmt.Errorf("logging in: %w", err)
	}

	// u è stato letto prima del login: lo si allinea a quanto appena salvato.
	u.LastLoginAt = &loggedInAt
	u.FailedLoginAttempts = 0

	accessToken, accessExpiresAt, err := s.tokens.issue(u.ID, u.RoleName)
	if err != nil {
		return Session{}, fmt.Errorf("logging in: %w", err)
	}

	return Session{
		User:             u,
		AccessToken:      accessToken,
		AccessExpiresAt:  accessExpiresAt,
		RefreshToken:     refreshToken,
		RefreshExpiresAt: refreshExpiresAt,
	}, nil
}

// Refresh sostituisce un refresh token valido con uno nuovo (rotazione) ed
// emette un nuovo access token con il ruolo attuale dell'utente.
func (s *Service) Refresh(ctx context.Context, refreshToken string) (Session, error) {
	if refreshToken == "" {
		return Session{}, ErrInvalidToken
	}

	newToken := newRandomToken()
	refreshExpiresAt := s.now().Add(RefreshTokenTTL)

	userID, err := s.store.RotateRefreshToken(ctx, hashToken(refreshToken), hashToken(newToken), refreshExpiresAt)
	if err != nil {
		return Session{}, fmt.Errorf("refreshing session: %w", err)
	}

	u, err := s.users.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			return Session{}, ErrInvalidToken
		}

		return Session{}, fmt.Errorf("refreshing session: %w", err)
	}

	accessToken, accessExpiresAt, err := s.tokens.issue(u.ID, u.RoleName)
	if err != nil {
		return Session{}, fmt.Errorf("refreshing session: %w", err)
	}

	return Session{
		User:             u,
		AccessToken:      accessToken,
		AccessExpiresAt:  accessExpiresAt,
		RefreshToken:     newToken,
		RefreshExpiresAt: refreshExpiresAt,
	}, nil
}

// Logout revoca il refresh token della sessione corrente; le altre sessioni
// dell'utente restano valide.
func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	if refreshToken == "" {
		return nil
	}

	if err := s.store.RevokeRefreshToken(ctx, hashToken(refreshToken)); err != nil {
		return fmt.Errorf("logging out: %w", err)
	}

	return nil
}

// RequestPasswordReset invia all'utente con quell'email un link di
// recupero valido PasswordResetTTL. Se l'email non corrisponde a nessun
// utente attivo non fa nulla e non restituisce errore: il chiamante risponde
// allo stesso modo nei due casi, così non si scopre quali email sono
// registrate. L'email parte in background per lo stesso motivo: la risposta
// non deve durare di più quando l'utente esiste. Un utente bloccato può
// chiedere il recupero: è il modo per sbloccare l'account.
func (s *Service) RequestPasswordReset(ctx context.Context, email string) error {
	u, err := s.users.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			return nil
		}

		return fmt.Errorf("requesting password reset: %w", err)
	}

	token := newRandomToken()

	if err := s.store.CreatePasswordReset(ctx, u.ID, hashToken(token), s.now().Add(PasswordResetTTL)); err != nil {
		return fmt.Errorf("requesting password reset: %w", err)
	}

	s.mailer.SendAsync(ctx, passwordResetEmail(u, s.appBaseURL+"/reset-password/"+token))

	return nil
}

// ResetPassword imposta una nuova password usando un token di reset valido.
// Il token diventa inutilizzabile, l'account viene sbloccato e tutte le
// sessioni dell'utente vengono chiuse.
func (s *Service) ResetPassword(ctx context.Context, token, password string) error {
	if token == "" {
		return ErrInvalidToken
	}

	passwordHash, err := user.HashPassword(password)
	if err != nil {
		return fmt.Errorf("resetting password: %w", err)
	}

	if err := s.store.ResetPassword(ctx, hashToken(token), passwordHash); err != nil {
		return fmt.Errorf("resetting password: %w", err)
	}

	return nil
}

// VerifyAccessToken controlla l'access token e ne restituisce le claims.
func (s *Service) VerifyAccessToken(token string) (Claims, error) {
	return s.tokens.parse(token)
}

// CurrentUser restituisce l'utente autenticato, se è ancora attivo.
func (s *Service) CurrentUser(ctx context.Context, id string) (user.User, error) {
	u, err := s.users.FindByID(ctx, id)
	if err != nil {
		return user.User{}, fmt.Errorf("getting current user: %w", err)
	}

	return u, nil
}
