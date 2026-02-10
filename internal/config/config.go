package config

import (
	"os"
)

type Config struct {
	APIAddr string
	PGDSN   string
}

func Load() Config {
	return Config{
		APIAddr: getenvDefault("API_ADDR", ":8080"),
		PGDSN:   getenvDefault("PG_DSN", "postgres://postgres:postgres@localhost:5432/events?sslmode=disable"),
	}
}

func getenvDefault(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
