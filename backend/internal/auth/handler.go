package auth

import (
	"errors"
	"log/slog"
	"net/http"
	"net/netip"
	"strings"
	"time"

	"github.com/matteotorelli/risma/backend/internal/httpx"
	"github.com/matteotorelli/risma/backend/internal/user"
)

// Nomi e percorsi dei cookie di sessione. Il refresh token viene inviato
// dal browser solo alle rotte /api/auth (rinnovo e logout).
const (
	accessCookieName  = "access_token"
	refreshCookieName = "refresh_token"
	accessCookiePath  = "/api"
	refreshCookiePath = "/api/auth"
)

// maxEmailBytes è la lunghezza massima di un indirizzo email (RFC 5321),
// pari alla colonna users.email.
const maxEmailBytes = 254

const weakPasswordMessage = "la password deve avere da 8 a 72 caratteri e contenere almeno una lettera maiuscola, un numero e un carattere speciale"

type Handler struct {
	svc            *Service
	logger         *slog.Logger
	trustedProxies []netip.Prefix
}

func NewHandler(svc *Service, logger *slog.Logger, trustedProxies []netip.Prefix) *Handler {
	return &Handler{svc: svc, logger: logger, trustedProxies: trustedProxies}
}

// RegisterRoutes collega le rotte di autenticazione al mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux, mw *Middleware) {
	mux.HandleFunc("POST /api/auth/login", h.login)
	mux.HandleFunc("POST /api/auth/refresh", h.refresh)
	mux.HandleFunc("POST /api/auth/logout", h.logout)
	mux.HandleFunc("POST /api/auth/forgot-password", h.forgotPassword)
	mux.HandleFunc("POST /api/auth/reset-password", h.resetPassword)
	mux.Handle("GET /api/auth/me", mw.RequireAuth(http.HandlerFunc(h.me)))
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "corpo della richiesta non valido")
		return
	}

	// Al login si controlla solo la presenza dei campi: una password che non
	// rispetta le regole AGID non può essere quella giusta, quindi segue il
	// percorso di una password errata e conta come tentativo fallito.
	username := strings.TrimSpace(req.Username)
	if username == "" || req.Password == "" {
		httpx.Error(w, http.StatusBadRequest, "username e password sono obbligatori")
		return
	}

	session, err := h.svc.Login(r.Context(), username, req.Password, httpx.ClientIP(r, h.trustedProxies))
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidCredentials):
			httpx.Error(w, http.StatusUnauthorized, "credenziali non valide")
		case errors.Is(err, ErrAccountBlocked):
			httpx.Error(w, http.StatusForbidden, "account bloccato dopo troppi tentativi falliti: usa il recupero password")
		default:
			httpx.InternalError(w, r, h.logger, err)
		}
		return
	}

	setSessionCookies(w, session)
	httpx.JSON(w, http.StatusOK, user.NewResponse(session.User))
}

func (h *Handler) refresh(w http.ResponseWriter, r *http.Request) {
	var token string
	if cookie, err := r.Cookie(refreshCookieName); err == nil {
		token = cookie.Value
	}

	session, err := h.svc.Refresh(r.Context(), token)
	if err != nil {
		if errors.Is(err, ErrInvalidToken) {
			clearSessionCookies(w)
			httpx.Error(w, http.StatusUnauthorized, "sessione scaduta, accedi di nuovo")
			return
		}

		httpx.InternalError(w, r, h.logger, err)
		return
	}

	setSessionCookies(w, session)
	httpx.JSON(w, http.StatusOK, user.NewResponse(session.User))
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	var token string
	if cookie, err := r.Cookie(refreshCookieName); err == nil {
		token = cookie.Value
	}

	if err := h.svc.Logout(r.Context(), token); err != nil {
		httpx.InternalError(w, r, h.logger, err)
		return
	}

	clearSessionCookies(w)
	w.WriteHeader(http.StatusNoContent)
}

type forgotPasswordRequest struct {
	Email string `json:"email"`
}

// forgotPassword risponde sempre allo stesso modo, che l'email sia
// registrata o no.
func (h *Handler) forgotPassword(w http.ResponseWriter, r *http.Request) {
	var req forgotPasswordRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "corpo della richiesta non valido")
		return
	}

	email := strings.TrimSpace(req.Email)
	if email == "" || len(email) > maxEmailBytes {
		httpx.Error(w, http.StatusBadRequest, "indica l'email del tuo account")
		return
	}

	if err := h.svc.RequestPasswordReset(r.Context(), email); err != nil {
		httpx.InternalError(w, r, h.logger, err)
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]string{
		"message": "Se l'email è associata a un account, riceverai un link per reimpostare la password, valido 1 ora.",
	})
}

type resetPasswordRequest struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

func (h *Handler) resetPassword(w http.ResponseWriter, r *http.Request) {
	var req resetPasswordRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "corpo della richiesta non valido")
		return
	}

	err := h.svc.ResetPassword(r.Context(), req.Token, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, user.ErrWeakPassword):
			httpx.Error(w, http.StatusBadRequest, weakPasswordMessage)
		case errors.Is(err, ErrInvalidToken):
			httpx.Error(w, http.StatusBadRequest, "il link di recupero non è valido o è scaduto: richiedine uno nuovo")
		default:
			httpx.InternalError(w, r, h.logger, err)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	identity, _ := IdentityFrom(r.Context())

	u, err := h.svc.CurrentUser(r.Context(), identity.UserID)
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			httpx.Error(w, http.StatusUnauthorized, "autenticazione richiesta")
			return
		}

		httpx.InternalError(w, r, h.logger, err)
		return
	}

	httpx.JSON(w, http.StatusOK, user.NewResponse(u))
}

// sessionCookie crea un cookie non leggibile da JavaScript (HttpOnly),
// inviato solo su HTTPS (Secure) e mai da richieste di altri siti
// (SameSite=Strict).
func sessionCookie(name, value, path string, maxAge time.Duration) *http.Cookie {
	return &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     path,
		MaxAge:   int(maxAge.Seconds()),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	}
}

func setSessionCookies(w http.ResponseWriter, s Session) {
	http.SetCookie(w, sessionCookie(accessCookieName, s.AccessToken, accessCookiePath, time.Until(s.AccessExpiresAt)))
	http.SetCookie(w, sessionCookie(refreshCookieName, s.RefreshToken, refreshCookiePath, time.Until(s.RefreshExpiresAt)))
}

// clearSessionCookies chiede al browser di cancellare i cookie (MaxAge < 0).
func clearSessionCookies(w http.ResponseWriter) {
	http.SetCookie(w, sessionCookie(accessCookieName, "", accessCookiePath, -time.Second))
	http.SetCookie(w, sessionCookie(refreshCookieName, "", refreshCookiePath, -time.Second))
}
