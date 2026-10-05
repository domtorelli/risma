// Il pacchetto identity trasporta l'utente autenticato nel contesto della
// richiesta. È separato da auth così i domini (user, product…) possono
// leggere chi sta agendo senza importare auth, che a sua volta importa user.
package identity

import "context"

// Identity è l'utente autenticato di una richiesta, letto dall'access token.
type Identity struct {
	UserID string
	Role   string
}

type contextKey struct{}

// NewContext restituisce una copia di ctx che contiene id.
func NewContext(ctx context.Context, id Identity) context.Context {
	return context.WithValue(ctx, contextKey{}, id)
}

// FromContext restituisce l'utente autenticato salvato dal middleware.
func FromContext(ctx context.Context) (Identity, bool) {
	id, ok := ctx.Value(contextKey{}).(Identity)
	return id, ok
}
