package database

import "testing"

func TestContainsPattern(t *testing.T) {
	tests := map[string]string{
		"":       "%%",
		"rossi":  "%rossi%",
		"a_b":    `%a\_b%`,
		"100%":   `%100\%%`,
		`c:\dir`: `%c:\\dir%`,
	}

	for input, want := range tests {
		if got := ContainsPattern(input); got != want {
			t.Errorf("ContainsPattern(%q) = %q, want %q", input, got, want)
		}
	}
}
