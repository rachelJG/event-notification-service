package httpadapter

import "context"

// DBStats holds observable metrics from the database connection pool.
type DBStats struct {
	MaxConns      int32 `json:"max_conns"`
	TotalConns    int32 `json:"total_conns"`
	IdleConns     int32 `json:"idle_conns"`
	AcquiredConns int32 `json:"acquired_conns"`
}

// HealthChecker abstracts the ability to check the health of an external dependency.
type HealthChecker interface {
	Ping(ctx context.Context) error
	Stats() DBStats
}

// RouterOptions holds the configuration values needed by the HTTP router.
// This struct decouples the HTTP adapter from the application-wide config package.
type RouterOptions struct {
	JWTSecret           string
	JWTIssuer           string // optional; validated when non-empty
	JWTAudience         string // optional; validated when non-empty
	MaxBodyBytes        int64
	RateLimitRPS        float64
	RateLimitBurst      int
	CORSAllowAllOrigins bool
	CORSAllowedOrigins  []string
	CORSAllowedHeaders  []string
	EnableHSTS          bool
	HSTSMaxAgeSeconds   int
	Version             string
	Commit              string
	TrustedProxies      []string
}
