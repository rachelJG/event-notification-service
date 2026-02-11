package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	APIAddr   string
	PGDSN     string
	LogLevel  string
	AppEnv    string
	JWTSecret string

	MaxBodyBytes      int64
	RateLimitRPS      float64
	RateLimitBurst    int
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
}

func Load() Config {
	return Config{
		APIAddr:   getenvDefault("API_ADDR", ":8080"),
		PGDSN:     getenvDefault("PG_DSN", "postgres://postgres:postgres@localhost:5432/events?sslmode=disable"),
		LogLevel:  getenvDefault("LOG_LEVEL", "info"),
		AppEnv:    getenvDefault("APP_ENV", "development"),
		JWTSecret: os.Getenv("JWT_SECRET"),

		MaxBodyBytes:      getenvInt64Default("MAX_BODY_BYTES", 1<<20),
		RateLimitRPS:      getenvFloatDefault("RATE_LIMIT_RPS", 10),
		RateLimitBurst:    getenvIntDefault("RATE_LIMIT_BURST", 20),
		ReadHeaderTimeout: getenvDurationSecondsDefault("READ_HEADER_TIMEOUT", 5),
		ReadTimeout:       getenvDurationSecondsDefault("READ_TIMEOUT", 15),
		WriteTimeout:      getenvDurationSecondsDefault("WRITE_TIMEOUT", 15),
		IdleTimeout:       getenvDurationSecondsDefault("IDLE_TIMEOUT", 60),
	}
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
