package config

import (
	"fmt"
	"os"
)

type Config struct {
	HTTPPort   string
	DBUser     string
	DBPassword string
	DBName     string
	DBHost     string
	DBPort     string
	DBSSLMode  string
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

	return cfg, nil
}
