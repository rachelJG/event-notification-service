// Package config provides configuration management for the application.
// It handles loading and validation of configuration from environment variables
// with sensible defaults for development environments.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all configuration parameters for the application.
type Config struct {
	// APIAddr is the address and port where the HTTP server will listen (e.g., ":8080")
	APIAddr string
	// PGDSN is the PostgreSQL connection string
	PGDSN string
	// LogLevel defines the logging level (debug, info, warn, error, fatal, panic)
	LogLevel string
	// AppEnv specifies the application environment (development, staging, production)
	AppEnv string
	// JWTSecret is used for signing JWT tokens (required in production)
	JWTSecret string

	// MaxBodyBytes is the maximum size of HTTP request body in bytes (default: 1MB)
	MaxBodyBytes int64
	// RateLimitRPS defines the maximum requests per second (default: 10)
	RateLimitRPS float64
	// RateLimitBurst is the maximum burst of requests (default: 20)
	RateLimitBurst int
	// ReadHeaderTimeout is the maximum duration for reading request headers (default: 5s)
	ReadHeaderTimeout time.Duration
	// ReadTimeout is the maximum duration for reading the entire request (default: 15s)
	ReadTimeout time.Duration
	// WriteTimeout is the maximum duration for writing the response (default: 15s)
	WriteTimeout time.Duration
	// IdleTimeout is the maximum amount of time to wait for the next request (default: 60s)
	IdleTimeout time.Duration

	// CORSAllowAllOrigins enables CORS for all origins (default: true in development)
	CORSAllowAllOrigins bool
	// CORSAllowedOrigins lists allowed origins when CORSAllowAllOrigins is false
	CORSAllowedOrigins []string
	// CORSAllowedHeaders lists allowed CORS headers
	CORSAllowedHeaders []string
	// EnableHSTS enables HTTP Strict Transport Security (default: true in production)
	EnableHSTS bool
	// HSTSMaxAgeSeconds defines the HSTS max-age in seconds (default: 31536000 - 1 year)
	HSTSMaxAgeSeconds int
}

// Load loads the application configuration from environment variables with sensible defaults.
// Environment variables and their defaults:
//   - API_ADDR: Server address (default: ":8080")
//   - PG_DSN: PostgreSQL connection string (default: local development database)
//   - LOG_LEVEL: Logging level (default: "info")
//   - APP_ENV: Application environment (default: "development")
//   - JWT_SECRET: Secret for JWT signing (required in production)
//   - MAX_BODY_BYTES: Max request body size in bytes (default: 1MB)
//   - RATE_LIMIT_RPS: Requests per second limit (default: 10)
//   - RATE_LIMIT_BURST: Burst limit for rate limiting (default: 20)
//   - READ_HEADER_TIMEOUT: Max time to read request headers in seconds (default: 5)
//   - READ_TIMEOUT: Max time to read request in seconds (default: 15)
//   - WRITE_TIMEOUT: Max time to write response in seconds (default: 15)
//   - IDLE_TIMEOUT: Max idle connection time in seconds (default: 60)
//   - CORS_ALLOW_ALL: Enable CORS for all origins (default: true in development)
//   - CORS_ALLOWED_ORIGINS: Comma-separated allowed origins (if CORS_ALLOW_ALL=false)
//   - CORS_ALLOWED_HEADERS: Comma-separated allowed CORS headers
//   - HSTS_ENABLED: Enable HSTS (default: true in production)
//   - HSTS_MAX_AGE_SECONDS: HSTS max-age in seconds (default: 31536000)
//
// Returns a populated Config struct.
func Load() Config {
	appEnv := getenvDefault("APP_ENV", "development")
	defaultAllowAll := appEnv != "production"
	return Config{
		APIAddr:   getenvDefault("API_ADDR", ":8080"),
		PGDSN:     getenvDefault("PG_DSN", "postgres://postgres:postgres@localhost:5432/events?sslmode=disable"),
		LogLevel:  getenvDefault("LOG_LEVEL", "info"),
		AppEnv:    appEnv,
		JWTSecret: os.Getenv("JWT_SECRET"),

		MaxBodyBytes:      getenvInt64Default("MAX_BODY_BYTES", 1<<20),
		RateLimitRPS:      getenvFloatDefault("RATE_LIMIT_RPS", 10),
		RateLimitBurst:    getenvIntDefault("RATE_LIMIT_BURST", 20),
		ReadHeaderTimeout: getenvDurationSecondsDefault("READ_HEADER_TIMEOUT", 5),
		ReadTimeout:       getenvDurationSecondsDefault("READ_TIMEOUT", 15),
		WriteTimeout:      getenvDurationSecondsDefault("WRITE_TIMEOUT", 15),
		IdleTimeout:       getenvDurationSecondsDefault("IDLE_TIMEOUT", 60),

		CORSAllowAllOrigins: getenvBoolDefault("CORS_ALLOW_ALL", defaultAllowAll),
		CORSAllowedOrigins:  getenvCSVDefault("CORS_ALLOWED_ORIGINS", nil),
		CORSAllowedHeaders:  getenvCSVDefault("CORS_ALLOWED_HEADERS", []string{"Origin", "Content-Length", "Content-Type", "Idempotency-Key", "Authorization"}),
		EnableHSTS:          getenvBoolDefault("HSTS_ENABLED", appEnv == "production"),
		HSTSMaxAgeSeconds:   getenvIntDefault("HSTS_MAX_AGE_SECONDS", 31536000),
	}
}

// Validate checks the configuration for logical errors and required fields.
// It ensures that CORS and HSTS settings are properly configured.
//
// Returns an error if the configuration is invalid, or nil if valid.
func (c Config) Validate() error {
	if strings.EqualFold(c.AppEnv, "production") && strings.TrimSpace(c.JWTSecret) == "" {
		return fmt.Errorf("invalid auth config: JWT_SECRET is required in production")
	}
	if c.CORSAllowAllOrigins && len(c.CORSAllowedOrigins) > 0 {
		return fmt.Errorf("invalid CORS config: CORS_ALLOW_ALL=true conflicts with CORS_ALLOWED_ORIGINS")
	}
	if !c.CORSAllowAllOrigins && len(c.CORSAllowedOrigins) == 0 {
		return fmt.Errorf("invalid CORS config: CORS_ALLOW_ALL=false requires CORS_ALLOWED_ORIGINS")
	}
	if c.EnableHSTS && c.HSTSMaxAgeSeconds <= 0 {
		return fmt.Errorf("invalid HSTS config: HSTS_MAX_AGE_SECONDS must be > 0")
	}
	return nil
}

func getenvDefault(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func getenvIntDefault(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func getenvInt64Default(key string, fallback int64) int64 {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return fallback
	}
	return parsed
}

func getenvFloatDefault(key string, fallback float64) float64 {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fallback
	}
	return parsed
}

func getenvDurationSecondsDefault(key string, fallbackSeconds int) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return time.Duration(fallbackSeconds) * time.Second
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return time.Duration(fallbackSeconds) * time.Second
	}
	return time.Duration(parsed) * time.Second
}

func getenvBoolDefault(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		switch strings.ToLower(value) {
		case "1", "t", "true", "yes", "y":
			return true
		case "0", "f", "false", "no", "n":
			return false
		}
		return fallback
	}
	return parsed
}

func getenvCSVDefault(key string, fallback []string) []string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		out = append(out, trimmed)
	}
	return out
}
