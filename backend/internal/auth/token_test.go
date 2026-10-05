package auth

import (
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var testSecret = []byte("test-secret-at-least-32-bytes-long!!")

func fixedClock(t time.Time) func() time.Time {
	return func() time.Time { return t }
}

func TestTokenRoundTrip(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	m := tokenManager{secret: testSecret, now: fixedClock(now)}

	token, expiresAt, err := m.issue("user-1", "admin")
	if err != nil {
		t.Fatalf("issue: %v", err)
	}

	if !expiresAt.Equal(now.Add(AccessTokenTTL)) {
		t.Errorf("expiresAt = %v", expiresAt)
	}

	claims, err := m.parse(token)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if claims.Subject != "user-1" || claims.Role != "admin" {
		t.Errorf("claims = %+v", claims)
	}
}

func TestTokenRejected(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	m := tokenManager{secret: testSecret, now: fixedClock(now)}

	valid, _, err := m.issue("user-1", "admin")
	if err != nil {
		t.Fatalf("issue: %v", err)
	}

	otherSecret := tokenManager{secret: []byte("another-secret-at-least-32-bytes!!"), now: fixedClock(now)}
	forged, _, _ := otherSecret.issue("user-1", "admin")

	expired := tokenManager{secret: testSecret, now: fixedClock(now.Add(-AccessTokenTTL - time.Second))}
	old, _, _ := expired.issue("user-1", "admin")

	unsigned, err := jwt.NewWithClaims(jwt.SigningMethodNone, Claims{
		Role: "admin",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			Subject:   "user-1",
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
		},
	}).SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("signing none token: %v", err)
	}

	tests := []struct {
		name  string
		token string
	}{
		{"empty", ""},
		{"garbage", "not-a-jwt"},
		{"tampered", valid + "x"},
		{"wrong secret", forged},
		{"expired", old},
		{"alg none", unsigned},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := m.parse(tt.token); !errors.Is(err, ErrInvalidToken) {
				t.Errorf("err = %v, want ErrInvalidToken", err)
			}
		})
	}
}
