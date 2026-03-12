package config

import (
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	cfg := Load()
	if cfg.APIAddr != ":8080" {
		t.Errorf("APIAddr = %q, want %q", cfg.APIAddr, ":8080")
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, "info")
	}
	if cfg.AppEnv != "development" {
		t.Errorf("AppEnv = %q, want %q", cfg.AppEnv, "development")
	}
	if cfg.MaxBodyBytes != 1<<20 {
		t.Errorf("MaxBodyBytes = %d, want %d", cfg.MaxBodyBytes, 1<<20)
	}
	if cfg.RateLimitRPS != 10 {
		t.Errorf("RateLimitRPS = %f, want %f", cfg.RateLimitRPS, 10.0)
	}
	if cfg.RateLimitBurst != 20 {
		t.Errorf("RateLimitBurst = %d, want %d", cfg.RateLimitBurst, 20)
	}
	if cfg.ReadHeaderTimeout != 5*time.Second {
		t.Errorf("ReadHeaderTimeout = %v, want %v", cfg.ReadHeaderTimeout, 5*time.Second)
	}
	if cfg.DBQueryTimeout != 5*time.Second {
		t.Errorf("DBQueryTimeout = %v, want %v", cfg.DBQueryTimeout, 5*time.Second)
	}
	if cfg.DBPoolMaxConns != 10 {
		t.Errorf("DBPoolMaxConns = %d, want %d", cfg.DBPoolMaxConns, 10)
	}
	if cfg.DBPoolMinConns != 2 {
		t.Errorf("DBPoolMinConns = %d, want %d", cfg.DBPoolMinConns, 2)
	}
	if cfg.SMTPPort != "587" {
		t.Errorf("SMTPPort = %q, want %q", cfg.SMTPPort, "587")
	}
	if cfg.WorkerBatchSize != 50 {
		t.Errorf("WorkerBatchSize = %d, want %d", cfg.WorkerBatchSize, 50)
	}
	// In development, CORSAllowAllOrigins defaults to true
	if !cfg.CORSAllowAllOrigins {
		t.Error("CORSAllowAllOrigins should default to true in development")
	}
	if cfg.EnableHSTS {
		t.Error("EnableHSTS should default to false in development")
	}
}

func TestLoadFromEnv(t *testing.T) {
	t.Setenv("API_ADDR", ":9090")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("APP_ENV", "production")
	t.Setenv("MAX_BODY_BYTES", "2048")
	t.Setenv("RATE_LIMIT_RPS", "5.5")
	t.Setenv("RATE_LIMIT_BURST", "10")
	t.Setenv("READ_HEADER_TIMEOUT", "10")
	t.Setenv("DB_QUERY_TIMEOUT", "3")
	t.Setenv("DB_POOL_MAX_CONNS", "20")
	t.Setenv("DB_POOL_MIN_CONNS", "5")
	t.Setenv("DB_POOL_MAX_CONN_LIFETIME", "7200")
	t.Setenv("DB_POOL_MAX_CONN_IDLE_TIME", "900")
	t.Setenv("CORS_ALLOW_ALL", "false")
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://example.com,https://other.com")
	t.Setenv("CORS_ALLOWED_HEADERS", "X-Custom")
	t.Setenv("HSTS_ENABLED", "true")
	t.Setenv("HSTS_MAX_AGE_SECONDS", "3600")
	t.Setenv("SMTP_HOST", "smtp.example.com")
	t.Setenv("SMTP_FROM", "noreply@example.com")
	t.Setenv("WORKER_BATCH_SIZE", "100")
	t.Setenv("TRUSTED_PROXIES", "10.0.0.0/8, 192.168.1.0/24")
	t.Setenv("SHUTDOWN_TIMEOUT", "30")

	cfg := Load()

	if cfg.APIAddr != ":9090" {
		t.Errorf("APIAddr = %q, want %q", cfg.APIAddr, ":9090")
	}
	if cfg.MaxBodyBytes != 2048 {
		t.Errorf("MaxBodyBytes = %d, want %d", cfg.MaxBodyBytes, 2048)
	}
	if cfg.RateLimitRPS != 5.5 {
		t.Errorf("RateLimitRPS = %f, want %f", cfg.RateLimitRPS, 5.5)
	}
	if cfg.RateLimitBurst != 10 {
		t.Errorf("RateLimitBurst = %d, want %d", cfg.RateLimitBurst, 10)
	}
	if cfg.ReadHeaderTimeout != 10*time.Second {
		t.Errorf("ReadHeaderTimeout = %v, want %v", cfg.ReadHeaderTimeout, 10*time.Second)
	}
	if cfg.DBQueryTimeout != 3*time.Second {
		t.Errorf("DBQueryTimeout = %v, want %v", cfg.DBQueryTimeout, 3*time.Second)
	}
	if cfg.DBPoolMaxConns != 20 {
		t.Errorf("DBPoolMaxConns = %d, want %d", cfg.DBPoolMaxConns, 20)
	}
	if cfg.DBPoolMinConns != 5 {
		t.Errorf("DBPoolMinConns = %d, want %d", cfg.DBPoolMinConns, 5)
	}
	if cfg.DBPoolMaxConnLifetime != 7200*time.Second {
		t.Errorf("DBPoolMaxConnLifetime = %v, want %v", cfg.DBPoolMaxConnLifetime, 7200*time.Second)
	}
	if cfg.DBPoolMaxConnIdleTime != 900*time.Second {
		t.Errorf("DBPoolMaxConnIdleTime = %v, want %v", cfg.DBPoolMaxConnIdleTime, 900*time.Second)
	}
	if cfg.CORSAllowAllOrigins {
		t.Error("CORSAllowAllOrigins should be false")
	}
	if len(cfg.CORSAllowedOrigins) != 2 {
		t.Errorf("CORSAllowedOrigins len = %d, want 2", len(cfg.CORSAllowedOrigins))
	}
	if len(cfg.CORSAllowedHeaders) != 1 || cfg.CORSAllowedHeaders[0] != "X-Custom" {
		t.Errorf("CORSAllowedHeaders = %v, want [X-Custom]", cfg.CORSAllowedHeaders)
	}
	if !cfg.EnableHSTS {
		t.Error("EnableHSTS should be true")
	}
	if cfg.HSTSMaxAgeSeconds != 3600 {
		t.Errorf("HSTSMaxAgeSeconds = %d, want %d", cfg.HSTSMaxAgeSeconds, 3600)
	}
	if cfg.WorkerBatchSize != 100 {
		t.Errorf("WorkerBatchSize = %d, want %d", cfg.WorkerBatchSize, 100)
	}
	if len(cfg.TrustedProxies) != 2 {
		t.Errorf("TrustedProxies len = %d, want 2", len(cfg.TrustedProxies))
	}
	if cfg.ShutdownTimeout != 30*time.Second {
		t.Errorf("ShutdownTimeout = %v, want %v", cfg.ShutdownTimeout, 30*time.Second)
	}
}

func TestLoadInvalidEnvFallsBackToDefaults(t *testing.T) {
	t.Setenv("MAX_BODY_BYTES", "not-a-number")
	t.Setenv("RATE_LIMIT_RPS", "not-a-float")
	t.Setenv("RATE_LIMIT_BURST", "xyz")
	t.Setenv("DB_POOL_MAX_CONNS", "abc")
	t.Setenv("DB_POOL_MIN_CONNS", "abc")
	t.Setenv("DB_QUERY_TIMEOUT", "abc")

	cfg := Load()
	if cfg.MaxBodyBytes != 1<<20 {
		t.Errorf("MaxBodyBytes should fall back to default, got %d", cfg.MaxBodyBytes)
	}
	if cfg.RateLimitRPS != 10 {
		t.Errorf("RateLimitRPS should fall back to default, got %f", cfg.RateLimitRPS)
	}
	if cfg.RateLimitBurst != 20 {
		t.Errorf("RateLimitBurst should fall back to default, got %d", cfg.RateLimitBurst)
	}
	if cfg.DBPoolMaxConns != 10 {
		t.Errorf("DBPoolMaxConns should fall back to default, got %d", cfg.DBPoolMaxConns)
	}
}

// Validate tests

func TestValidateValidDevConfig(t *testing.T) {
	cfg := Config{
		AppEnv:              "development",
		CORSAllowAllOrigins: true,
		RateLimitRPS:        10,
		RateLimitBurst:      20,
		MaxBodyBytes:        1 << 20,
		DBQueryTimeout:      5 * time.Second,
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected valid config, got %v", err)
	}
}

func TestValidateProductionRequiresSMTPHostFromValidate(t *testing.T) {
	cfg := Config{
		AppEnv:              "production",
		CORSAllowAllOrigins: false,
		CORSAllowedOrigins:  []string{"https://example.com"},
		RateLimitRPS:        10,
		RateLimitBurst:      20,
		MaxBodyBytes:        1 << 20,
		DBQueryTimeout:      5 * time.Second,
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for missing SMTP_HOST in production")
	}
}

func TestValidateTLSRequiresBothFiles(t *testing.T) {
	cfg := Config{
		AppEnv:              "development",
		TLSCertFile:         "/path/cert.pem",
		CORSAllowAllOrigins: true,
		RateLimitRPS:        10,
		RateLimitBurst:      20,
		MaxBodyBytes:        1 << 20,
		DBQueryTimeout:      5 * time.Second,
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error when only TLS cert is set")
	}
}

func TestValidateCORSConflict(t *testing.T) {
	cfg := Config{
		AppEnv:              "development",
		CORSAllowAllOrigins: true,
		CORSAllowedOrigins:  []string{"https://example.com"},
		RateLimitRPS:        10,
		RateLimitBurst:      20,
		MaxBodyBytes:        1 << 20,
		DBQueryTimeout:      5 * time.Second,
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for CORS conflict")
	}
}

func TestValidateCORSRequiresOrigins(t *testing.T) {
	cfg := Config{
		AppEnv:              "development",
		CORSAllowAllOrigins: false,
		RateLimitRPS:        10,
		RateLimitBurst:      20,
		MaxBodyBytes:        1 << 20,
		DBQueryTimeout:      5 * time.Second,
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error when CORS_ALLOW_ALL=false with no origins")
	}
}

func TestValidateHSTSMaxAge(t *testing.T) {
	cfg := Config{
		AppEnv:              "development",
		CORSAllowAllOrigins: true,
		EnableHSTS:          true,
		HSTSMaxAgeSeconds:   0,
		RateLimitRPS:        10,
		RateLimitBurst:      20,
		MaxBodyBytes:        1 << 20,
		DBQueryTimeout:      5 * time.Second,
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for HSTS max age <= 0")
	}
}

func TestValidateRateLimitRPSPositive(t *testing.T) {
	cfg := Config{
		AppEnv:              "development",
		CORSAllowAllOrigins: true,
		RateLimitRPS:        0,
		RateLimitBurst:      20,
		MaxBodyBytes:        1 << 20,
		DBQueryTimeout:      5 * time.Second,
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for RateLimitRPS <= 0")
	}
}

func TestValidateRateLimitBurstPositive(t *testing.T) {
	cfg := Config{
		AppEnv:              "development",
		CORSAllowAllOrigins: true,
		RateLimitRPS:        10,
		RateLimitBurst:      0,
		MaxBodyBytes:        1 << 20,
		DBQueryTimeout:      5 * time.Second,
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for RateLimitBurst <= 0")
	}
}

func TestValidateMaxBodyBytesPositive(t *testing.T) {
	cfg := Config{
		AppEnv:              "development",
		CORSAllowAllOrigins: true,
		RateLimitRPS:        10,
		RateLimitBurst:      20,
		MaxBodyBytes:        0,
		DBQueryTimeout:      5 * time.Second,
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for MaxBodyBytes <= 0")
	}
}

func TestValidateDBQueryTimeoutPositive(t *testing.T) {
	cfg := Config{
		AppEnv:              "development",
		CORSAllowAllOrigins: true,
		RateLimitRPS:        10,
		RateLimitBurst:      20,
		MaxBodyBytes:        1 << 20,
		DBQueryTimeout:      0,
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for DBQueryTimeout <= 0")
	}
}

func TestValidateProductionSMTPHostRequired(t *testing.T) {
	cfg := Config{
		AppEnv:              "production",
		CORSAllowAllOrigins: false,
		CORSAllowedOrigins:  []string{"https://example.com"},
		RateLimitRPS:        10,
		RateLimitBurst:      20,
		MaxBodyBytes:        1 << 20,
		DBQueryTimeout:      5 * time.Second,
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for missing SMTP_HOST in production")
	}
}

func TestValidateSMTPHostRequiresSMTPFrom(t *testing.T) {
	cfg := Config{
		AppEnv:              "development",
		CORSAllowAllOrigins: true,
		RateLimitRPS:        10,
		RateLimitBurst:      20,
		MaxBodyBytes:        1 << 20,
		DBQueryTimeout:      5 * time.Second,
		SMTPHost:            "smtp.example.com",
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for SMTP_HOST without SMTP_FROM")
	}
}

// Helper function tests

func TestGetenvBoolDefaultEdgeCases(t *testing.T) {
	// strconv.ParseBool handles standard values; test the custom fallbacks
	t.Setenv("TEST_BOOL_YES", "yes")
	t.Setenv("TEST_BOOL_NO", "no")
	t.Setenv("TEST_BOOL_Y", "y")
	t.Setenv("TEST_BOOL_N", "n")
	t.Setenv("TEST_BOOL_INVALID", "maybe")

	if !getenvBoolDefault("TEST_BOOL_YES", false) {
		t.Error("expected 'yes' to be true")
	}
	if getenvBoolDefault("TEST_BOOL_NO", true) {
		t.Error("expected 'no' to be false")
	}
	if !getenvBoolDefault("TEST_BOOL_Y", false) {
		t.Error("expected 'y' to be true")
	}
	if getenvBoolDefault("TEST_BOOL_N", true) {
		t.Error("expected 'n' to be false")
	}
	// "maybe" is invalid, strconv.ParseBool fails, custom switch doesn't match → fallback
	if getenvBoolDefault("TEST_BOOL_INVALID", false) {
		t.Error("expected 'maybe' to fall back to false")
	}
}

func TestGetenvCSVDefaultSkipsEmpty(t *testing.T) {
	t.Setenv("TEST_CSV", "a, , b, ,c")
	result := getenvCSVDefault("TEST_CSV", nil)
	if len(result) != 3 {
		t.Fatalf("expected 3 items, got %d: %v", len(result), result)
	}
	if result[0] != "a" || result[1] != "b" || result[2] != "c" {
		t.Errorf("unexpected values: %v", result)
	}
}

func TestGetenvCSVDefaultEmpty(t *testing.T) {
	result := getenvCSVDefault("NONEXISTENT_CSV_KEY", []string{"default"})
	if len(result) != 1 || result[0] != "default" {
		t.Errorf("expected default, got %v", result)
	}
}
