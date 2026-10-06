package auth

import (
	"context"
	"errors"
	"net/netip"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/domtorelli/risma/backend/internal/mail"
	"github.com/domtorelli/risma/backend/internal/user"
)

const testPassword = "Password1!"

var testIP = netip.MustParseAddr("203.0.113.7")

// fakeUsers contiene un solo utente.
type fakeUsers struct {
	u user.User
}

func (f *fakeUsers) FindByUsername(ctx context.Context, username string) (user.User, error) {
	if username != f.u.Username {
		return user.User{}, user.ErrNotFound
	}
	return f.u, nil
}

func (f *fakeUsers) FindByEmail(ctx context.Context, email string) (user.User, error) {
	if email != f.u.Email {
		return user.User{}, user.ErrNotFound
	}
	return f.u, nil
}

func (f *fakeUsers) FindByID(ctx context.Context, id string) (user.User, error) {
	if id != f.u.ID {
		return user.User{}, user.ErrNotFound
	}
	return f.u, nil
}

// fakeStore imita le query del repository sullo stato dell'utente di fakeUsers.
type fakeStore struct {
	users    *fakeUsers
	failures int
	logins   int
	tokens   map[string]string    // hash -> user id dei token validi
	resets   map[string]time.Time // hash -> scadenza dei token di reset validi
	newHash  string               // ultimo hash di password salvato
}

func (f *fakeStore) RecordFailure(ctx context.Context, userID string, ip netip.Addr, maxAttempts int) (string, error) {
	f.failures++
	u := &f.users.u
	u.FailedLoginAttempts = min(u.FailedLoginAttempts+1, maxAttempts)
	if u.FailedLoginAttempts >= maxAttempts {
		u.Status = user.StatusBlocked
	}
	return u.Status, nil
}

func (f *fakeStore) RecordLogin(ctx context.Context, userID string, ip netip.Addr, tokenHash string, expiresAt time.Time) (time.Time, error) {
	f.logins++
	f.users.u.FailedLoginAttempts = 0
	f.tokens[tokenHash] = userID
	return time.Now(), nil
}

func (f *fakeStore) RotateRefreshToken(ctx context.Context, oldHash, newHash string, expiresAt time.Time) (string, error) {
	userID, ok := f.tokens[oldHash]
	if !ok {
		return "", ErrInvalidToken
	}
	delete(f.tokens, oldHash)
	f.tokens[newHash] = userID
	return userID, nil
}

func (f *fakeStore) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	delete(f.tokens, tokenHash)
	return nil
}

func (f *fakeStore) CreatePasswordReset(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error {
	clear(f.resets) // i link precedenti non valgono più
	f.resets[tokenHash] = expiresAt
	return nil
}

func (f *fakeStore) ResetPassword(ctx context.Context, tokenHash, passwordHash string) error {
	if _, ok := f.resets[tokenHash]; !ok {
		return ErrInvalidToken
	}
	delete(f.resets, tokenHash)
	f.newHash = passwordHash
	return nil
}

type fakeMailer struct {
	sent []mail.Message
}

func (f *fakeMailer) SendAsync(ctx context.Context, msg mail.Message) {
	f.sent = append(f.sent, msg)
}

func newTestService(t *testing.T) (*Service, *fakeStore) {
	svc, store, _ := newTestServiceWithMailer(t)
	return svc, store
}

func newTestServiceWithMailer(t *testing.T) (*Service, *fakeStore, *fakeMailer) {
	t.Helper()

	hash, err := bcrypt.GenerateFromPassword([]byte(testPassword), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}

	users := &fakeUsers{u: user.User{
		ID:           "user-1",
		Username:     "mrossi",
		Email:        "mario.rossi@example.com",
		RoleName:     "operatore",
		PasswordHash: string(hash),
		Status:       user.StatusActive,
	}}
	store := &fakeStore{users: users, tokens: map[string]string{}, resets: map[string]time.Time{}}
	mailer := &fakeMailer{}

	svc, err := NewService(users, store, mailer, testSecret, "https://risma.example.it")
	if err != nil {
		t.Fatal(err)
	}

	return svc, store, mailer
}

func TestLoginSuccess(t *testing.T) {
	svc, store := newTestService(t)

	session, err := svc.Login(context.Background(), "mrossi", testPassword, testIP)
	if err != nil {
		t.Fatalf("Login: %v", err)
	}

	if store.logins != 1 || store.failures != 0 {
		t.Errorf("logins = %d, failures = %d", store.logins, store.failures)
	}

	if session.User.LastLoginAt == nil {
		t.Error("LastLoginAt not set in returned user")
	}

	if _, ok := store.tokens[hashToken(session.RefreshToken)]; !ok {
		t.Error("refresh token hash not stored")
	}

	if _, ok := store.tokens[session.RefreshToken]; ok {
		t.Error("refresh token stored in clear text")
	}

	claims, err := svc.VerifyAccessToken(session.AccessToken)
	if err != nil || claims.Subject != "user-1" || claims.Role != "operatore" {
		t.Errorf("claims = %+v, err = %v", claims, err)
	}
}

func TestLoginUnknownUser(t *testing.T) {
	svc, store := newTestService(t)

	_, err := svc.Login(context.Background(), "nobody", testPassword, testIP)
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("err = %v, want ErrInvalidCredentials", err)
	}

	if store.failures != 0 {
		t.Error("attempt recorded for unknown user")
	}
}

func TestLoginBlocksAfterFiveFailures(t *testing.T) {
	svc, store := newTestService(t)
	ctx := context.Background()

	for attempt := 1; attempt < maxFailedAttempts; attempt++ {
		if _, err := svc.Login(ctx, "mrossi", "Wrong1!pass", testIP); !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("attempt %d: err = %v, want ErrInvalidCredentials", attempt, err)
		}
	}

	if _, err := svc.Login(ctx, "mrossi", "Wrong1!pass", testIP); !errors.Is(err, ErrAccountBlocked) {
		t.Fatalf("fifth attempt: err = %v, want ErrAccountBlocked", err)
	}

	// Con l'account bloccato anche la password giusta viene rifiutata,
	// e il tentativo viene comunque registrato.
	if _, err := svc.Login(ctx, "mrossi", testPassword, testIP); !errors.Is(err, ErrAccountBlocked) {
		t.Fatalf("correct password on blocked account: err = %v, want ErrAccountBlocked", err)
	}

	if store.failures != maxFailedAttempts+1 || store.logins != 0 {
		t.Errorf("failures = %d, logins = %d", store.failures, store.logins)
	}
}

func TestLoginSuccessResetsFailures(t *testing.T) {
	svc, store := newTestService(t)
	ctx := context.Background()

	for range maxFailedAttempts - 1 {
		_, _ = svc.Login(ctx, "mrossi", "Wrong1!pass", testIP)
	}

	if _, err := svc.Login(ctx, "mrossi", testPassword, testIP); err != nil {
		t.Fatalf("Login: %v", err)
	}

	if store.users.u.FailedLoginAttempts != 0 {
		t.Errorf("FailedLoginAttempts = %d, want 0", store.users.u.FailedLoginAttempts)
	}

	if _, err := svc.Login(ctx, "mrossi", "Wrong1!pass", testIP); !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("err = %v, want ErrInvalidCredentials (counter must restart)", err)
	}
}

func TestRefreshRotatesToken(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	first, err := svc.Login(ctx, "mrossi", testPassword, testIP)
	if err != nil {
		t.Fatalf("Login: %v", err)
	}

	second, err := svc.Refresh(ctx, first.RefreshToken)
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}

	if second.RefreshToken == first.RefreshToken {
		t.Error("refresh token not rotated")
	}

	if _, err := svc.Refresh(ctx, first.RefreshToken); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("reusing rotated token: err = %v, want ErrInvalidToken", err)
	}

	if err := svc.Logout(ctx, second.RefreshToken); err != nil {
		t.Fatalf("Logout: %v", err)
	}

	if _, err := svc.Refresh(ctx, second.RefreshToken); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("refresh after logout: err = %v, want ErrInvalidToken", err)
	}
}

func TestRefreshEmptyToken(t *testing.T) {
	svc, _ := newTestService(t)

	if _, err := svc.Refresh(context.Background(), ""); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("err = %v, want ErrInvalidToken", err)
	}
}

func TestRequestPasswordResetUnknownEmail(t *testing.T) {
	svc, store, mailer := newTestServiceWithMailer(t)

	if err := svc.RequestPasswordReset(context.Background(), "nobody@example.com"); err != nil {
		t.Fatalf("err = %v, want nil (no hint that the email is unknown)", err)
	}

	if len(mailer.sent) != 0 || len(store.resets) != 0 {
		t.Error("reset created for unknown email")
	}
}

func TestPasswordResetFlow(t *testing.T) {
	svc, store, mailer := newTestServiceWithMailer(t)
	ctx := context.Background()
	before := time.Now()

	if err := svc.RequestPasswordReset(ctx, "mario.rossi@example.com"); err != nil {
		t.Fatalf("RequestPasswordReset: %v", err)
	}

	if len(mailer.sent) != 1 {
		t.Fatalf("sent %d emails, want 1", len(mailer.sent))
	}
	msg := mailer.sent[0]

	const prefix = "https://risma.example.it/reset-password/"
	if len(msg.Secrets) != 1 || !strings.HasPrefix(msg.Secrets[0], prefix) || !strings.Contains(msg.Body, msg.Secrets[0]) {
		t.Fatalf("link not in body or not marked as secret: %+v", msg)
	}
	token := strings.TrimPrefix(msg.Secrets[0], prefix)

	expiresAt, ok := store.resets[hashToken(token)]
	if !ok {
		t.Fatal("token hash not stored")
	}
	if _, ok := store.resets[token]; ok {
		t.Error("reset token stored in clear text")
	}
	if ttl := expiresAt.Sub(before); ttl < PasswordResetTTL || ttl > PasswordResetTTL+time.Minute {
		t.Errorf("token valid for %v, want %v", ttl, PasswordResetTTL)
	}

	if err := svc.ResetPassword(ctx, token, "weak"); !errors.Is(err, user.ErrWeakPassword) {
		t.Fatalf("weak password: err = %v, want ErrWeakPassword", err)
	}

	if err := svc.ResetPassword(ctx, token, "NuovaPassword1!"); err != nil {
		t.Fatalf("ResetPassword: %v", err)
	}
	if bcrypt.CompareHashAndPassword([]byte(store.newHash), []byte("NuovaPassword1!")) != nil {
		t.Error("new password not hashed with bcrypt")
	}

	if err := svc.ResetPassword(ctx, token, "AltraPassword1!"); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("reusing token: err = %v, want ErrInvalidToken", err)
	}
}

func TestNewResetRequestInvalidatesPreviousLink(t *testing.T) {
	svc, _, mailer := newTestServiceWithMailer(t)
	ctx := context.Background()

	_ = svc.RequestPasswordReset(ctx, "mario.rossi@example.com")
	_ = svc.RequestPasswordReset(ctx, "mario.rossi@example.com")

	const prefix = "https://risma.example.it/reset-password/"
	first := strings.TrimPrefix(mailer.sent[0].Secrets[0], prefix)

	if err := svc.ResetPassword(ctx, first, "NuovaPassword1!"); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("old link: err = %v, want ErrInvalidToken", err)
	}
}

func TestInviteUser(t *testing.T) {
	svc, store, mailer := newTestServiceWithMailer(t)
	before := time.Now()

	if err := svc.InviteUser(context.Background(), store.users.u); err != nil {
		t.Fatalf("InviteUser: %v", err)
	}

	if len(mailer.sent) != 1 || mailer.sent[0].Type != mail.TypeNewUser {
		t.Fatalf("sent = %+v, want one new_user email", mailer.sent)
	}

	msg := mailer.sent[0]
	if len(msg.Secrets) != 1 || !strings.Contains(msg.Body, msg.Secrets[0]) || !strings.Contains(msg.Body, "mrossi") {
		t.Fatalf("body must contain username and secret link: %+v", msg)
	}

	token := strings.TrimPrefix(msg.Secrets[0], "https://risma.example.it/reset-password/")
	expiresAt := store.resets[hashToken(token)]
	if ttl := expiresAt.Sub(before); ttl < InviteTTL || ttl > InviteTTL+time.Minute {
		t.Errorf("invite valid for %v, want %v", ttl, InviteTTL)
	}

	if err := svc.ResetPassword(context.Background(), token, "PrimaPassword1!"); err != nil {
		t.Errorf("setting first password with invite link: %v", err)
	}
}
