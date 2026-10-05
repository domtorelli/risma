package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRequireCSRF(t *testing.T) {
	tests := []struct {
		name   string
		method string
		cookie string // vuoto: nessun cookie
		header string // vuoto: nessun header
		want   int
	}{
		{"safe method without token", http.MethodGet, "", "", http.StatusOK},
		{"matching token", http.MethodPost, "token-123", "token-123", http.StatusOK},
		{"matching token on delete", http.MethodDelete, "token-123", "token-123", http.StatusOK},
		{"missing header", http.MethodPost, "token-123", "", http.StatusForbidden},
		{"missing cookie", http.MethodPut, "", "token-123", http.StatusForbidden},
		{"different values", http.MethodPost, "token-123", "token-456", http.StatusForbidden},
		{"both missing", http.MethodDelete, "", "", http.StatusForbidden},
	}

	const cookieName = "__Host-csrf_token"
	mw := NewMiddleware(nil, cookieName)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(tt.method, "/api/users", nil)
			if tt.cookie != "" {
				r.AddCookie(&http.Cookie{Name: cookieName, Value: tt.cookie})
			}
			if tt.header != "" {
				r.Header.Set(CSRFHeader, tt.header)
			}

			w := httptest.NewRecorder()
			mw.RequireCSRF(next).ServeHTTP(w, r)

			if w.Code != tt.want {
				t.Errorf("status = %d, want %d", w.Code, tt.want)
			}
		})
	}
}

func TestSetCSRFCookie(t *testing.T) {
	const cookieName = "csrf_token"
	r := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
	r.AddCookie(&http.Cookie{Name: cookieName, Value: "existing"})

	tokenSet := func(rotate bool) *http.Cookie {
		w := httptest.NewRecorder()
		setCSRFCookie(w, r, cookieName, testExpiry(), rotate)
		return w.Result().Cookies()[0]
	}

	kept := tokenSet(false)
	if kept.Value != "existing" {
		t.Errorf("refresh: token = %q, want the existing one", kept.Value)
	}

	rotated := tokenSet(true)
	if rotated.Value == "existing" || rotated.Value == "" {
		t.Errorf("login: token = %q, want a new one", rotated.Value)
	}

	if rotated.HttpOnly || !rotated.Secure || rotated.Path != "/" || rotated.SameSite != http.SameSiteStrictMode {
		t.Errorf("cookie attributes = %+v", rotated)
	}
}

func testExpiry() time.Time {
	return time.Now().Add(time.Hour)
}

func TestCSRFCookieName(t *testing.T) {
	if got := CSRFCookieName(true); got != "__Host-csrf_token" {
		t.Errorf("https: %q", got)
	}
	if got := CSRFCookieName(false); got != "csrf_token" {
		t.Errorf("http: %q", got)
	}
}
