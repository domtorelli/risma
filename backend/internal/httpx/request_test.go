package httpx

import "testing"

func TestIsUUID(t *testing.T) {
	tests := map[string]bool{
		"01a10c63-cc5c-737d-b18c-1e7ff36c38fc": true,
		"01A10C63-CC5C-737D-B18C-1E7FF36C38FC": true,
		"01a10c63cc5c737db18c1e7ff36c38fc":     false,
		"01a10c63-cc5c-737d-b18c-1e7ff36c38fg": false,
		"' OR 1=1 --":                          false,
		"":                                     false,
	}

	for input, want := range tests {
		if got := IsUUID(input); got != want {
			t.Errorf("IsUUID(%q) = %v, want %v", input, got, want)
		}
	}
}
