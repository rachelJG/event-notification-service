package httpadapter

import "context"

// HealthChecker abstracts the ability to check the health of an external dependency.
type HealthChecker interface {
	Ping(ctx context.Context) error
}

// RouterOptions holds the configuration values needed by the HTTP router.
// This struct decouples the HTTP adapter from the application-wide config package.
type RouterOptions struct {
	JWTSecret           string
	MaxBodyBytes        int64
	RateLimitRPS        float64
	RateLimitBurst      int
	CORSAllowAllOrigins bool
	CORSAllowedOrigins  []string
	CORSAllowedHeaders  []string
	EnableHSTS          bool
	HSTSMaxAgeSeconds   int
}
