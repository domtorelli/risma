package auth

import (
	"net/http"
	"slices"

	"github.com/matteotorelli/risma/backend/internal/httpx"
	"github.com/matteotorelli/risma/backend/internal/identity"
)

type tokenVerifier interface {
	VerifyAccessToken(token string) (Claims, error)
}

var _ tokenVerifier = (*Service)(nil)

type Middleware struct {
	verifier tokenVerifier
}

func NewMiddleware(verifier tokenVerifier) *Middleware {
	return &Middleware{verifier: verifier}
}

// RequireAuth lascia passare solo le richieste con un access token valido
// nel cookie e salva l'utente nel contesto della richiesta (identity).
func (m *Middleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(accessCookieName)
		if err != nil {
			httpx.Error(w, http.StatusUnauthorized, "autenticazione richiesta")
			return
		}

		claims, err := m.verifier.VerifyAccessToken(cookie.Value)
		if err != nil {
			httpx.Error(w, http.StatusUnauthorized, "autenticazione richiesta")
			return
		}

		ctx := identity.NewContext(r.Context(), identity.Identity{UserID: claims.Subject, Role: claims.Role})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireRole lascia passare solo gli utenti con uno dei ruoli indicati.
// Va usato dopo RequireAuth.
func (m *Middleware) RequireRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id, ok := identity.FromContext(r.Context())
			if !ok {
				httpx.Error(w, http.StatusUnauthorized, "autenticazione richiesta")
				return
			}

			if !slices.Contains(roles, id.Role) {
				httpx.Error(w, http.StatusForbidden, "operazione non consentita per il tuo ruolo")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
