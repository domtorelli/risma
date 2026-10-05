package config

import (
	"fmt"
	"net/netip"
	"net/url"
	"os"
	"strings"
)

// minJWTSecretBytes è la lunghezza minima del segreto HS256: deve essere
// almeno pari all'output di SHA-256 (RFC 7518, §3.2).
const minJWTSecretBytes = 32

type Config struct {
	HTTPPort   string
	DBUser     string
	DBPassword string
	DBName     string
	DBHost     string
	DBPort     string
	DBSSLMode  string
	JWTSecret  []byte
	// TrustedProxies sono le reti dei reverse proxy da cui si accetta
	// l'header X-Real-IP; vuoto significa nessun proxy fidato.
	TrustedProxies []netip.Prefix
	// AppBaseURL è l'indirizzo pubblico del frontend, usato nei link delle
	// email (es. https://risma.example.it), senza barra finale.
	AppBaseURL   string
	SMTPHost     string
	SMTPPort     string
	SMTPUsername string // vuoto: nessuna autenticazione (es. Mailpit)
	SMTPPassword string
	SMTPFrom     string
}

func getEnv(key string, fallback string) string {
	value, ok := os.LookupEnv(key)

	if !ok || value == "" {
		return fallback
	}

	return value
}

func requireEnv(key string) (string, error) {
	value, ok := os.LookupEnv(key)

	if !ok || value == "" {
		return "", fmt.Errorf("missing required env variable %s", key)
	}

	return value, nil
}

// parsePrefixes legge un elenco di IP o reti CIDR separati da virgola;
// un IP singolo diventa una rete con un solo indirizzo.
func parsePrefixes(list string) ([]netip.Prefix, error) {
	var prefixes []netip.Prefix

	for item := range strings.SplitSeq(list, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}

		if strings.Contains(item, "/") {
			prefix, err := netip.ParsePrefix(item)
			if err != nil {
				return nil, fmt.Errorf("parsing network %q: %w", item, err)
			}
			prefixes = append(prefixes, prefix.Masked())

			continue
		}

		addr, err := netip.ParseAddr(item)
		if err != nil {
			return nil, fmt.Errorf("parsing address %q: %w", item, err)
		}
		prefixes = append(prefixes, netip.PrefixFrom(addr, addr.BitLen()))
	}

	return prefixes, nil
}

// parseBaseURL verifica che value sia un URL assoluto http(s) e lo
// restituisce senza barra finale.
func parseBaseURL(value string) (string, error) {
	u, err := url.Parse(value)
	if err != nil {
		return "", err
	}

	if (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return "", fmt.Errorf("%q is not an absolute http(s) URL", value)
	}

	return strings.TrimSuffix(value, "/"), nil
}

func Load() (Config, error) {
	var cfg Config
	var err error

	cfg.HTTPPort = getEnv("HTTP_PORT", "8080")
	cfg.DBUser, err = requireEnv("DB_USER")
	if err != nil {
		return Config{}, err
	}
	cfg.DBPassword, err = requireEnv("DB_PASSWORD")
	if err != nil {
		return Config{}, err
	}
	cfg.DBName, err = requireEnv("DB_NAME")
	if err != nil {
		return Config{}, err
	}
	cfg.DBHost = getEnv("DB_HOST", "localhost")
	cfg.DBPort = getEnv("DB_PORT", "5432")
	cfg.DBSSLMode, err = requireEnv("DB_SSLMODE")
	if err != nil {
		return Config{}, err
	}

	secret, err := requireEnv("JWT_SECRET")
	if err != nil {
		return Config{}, err
	}
	if len(secret) < minJWTSecretBytes {
		return Config{}, fmt.Errorf("JWT_SECRET must be at least %d bytes long", minJWTSecretBytes)
	}
	cfg.JWTSecret = []byte(secret)

	cfg.TrustedProxies, err = parsePrefixes(getEnv("TRUSTED_PROXIES", ""))
	if err != nil {
		return Config{}, fmt.Errorf("reading TRUSTED_PROXIES: %w", err)
	}

	baseURL, err := requireEnv("APP_BASE_URL")
	if err != nil {
		return Config{}, err
	}
	cfg.AppBaseURL, err = parseBaseURL(baseURL)
	if err != nil {
		return Config{}, fmt.Errorf("reading APP_BASE_URL: %w", err)
	}

	cfg.SMTPHost = getEnv("SMTP_HOST", "localhost")
	cfg.SMTPPort = getEnv("SMTP_PORT", "1025")
	cfg.SMTPUsername = getEnv("SMTP_USERNAME", "")
	cfg.SMTPPassword = getEnv("SMTP_PASSWORD", "")
	cfg.SMTPFrom = getEnv("SMTP_FROM", "Risma <noreply@risma.local>")

	return cfg, nil
}
