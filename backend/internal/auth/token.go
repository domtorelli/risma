package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	// AccessTokenTTL è breve perché l'access token si verifica con la sola
	// firma: una revoca o un cambio di ruolo hanno effetto entro questo tempo.
	AccessTokenTTL  = 15 * time.Minute
	RefreshTokenTTL = 7 * 24 * time.Hour

	issuer = "risma"
)

// Claims sono i dati contenuti nell'access token: Subject è l'id utente.
type Claims struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
}

// tokenManager firma e verifica gli access token con HMAC-SHA256.
type tokenManager struct {
	secret []byte
	now    func() time.Time
}

// issue crea un access token per l'utente con il ruolo indicato.
func (m tokenManager) issue(userID, role string) (string, time.Time, error) {
	now := m.now()
	expiresAt := now.Add(AccessTokenTTL)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		Role: role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	})

	signed, err := token.SignedString(m.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("signing access token: %w", err)
	}

	return signed, expiresAt, nil
}

// parse verifica firma, algoritmo, emittente e scadenza dell'access token.
// Accettare solo HS256 impedisce gli attacchi che cambiano l'algoritmo
// nell'header (es. "alg": "none").
func (m tokenManager) parse(tokenString string) (Claims, error) {
	var claims Claims

	_, err := jwt.ParseWithClaims(tokenString, &claims,
		func(*jwt.Token) (any, error) { return m.secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(issuer),
		jwt.WithExpirationRequired(),
		jwt.WithTimeFunc(m.now),
	)
	if err != nil {
		return Claims{}, fmt.Errorf("%w: %w", ErrInvalidToken, err)
	}

	if claims.Subject == "" {
		return Claims{}, fmt.Errorf("%w: missing subject", ErrInvalidToken)
	}

	return claims, nil
}

// newRandomToken genera un token casuale (128 bit di entropia) per i
// refresh token e i link di reset. Il token è opaco: viene salvato solo il
// suo hash e ha valore solo se presente nel database.
func newRandomToken() string {
	return rand.Text()
}

// hashToken calcola lo SHA-256 esadecimale di un token. Basta un hash veloce
// perché il token è casuale e lungo: non si può indovinare per tentativi.
func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
