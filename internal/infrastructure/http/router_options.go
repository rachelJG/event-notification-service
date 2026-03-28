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
	// MaxBodyBytes is the maximum allowed size for request bodies in bytes.
	// Enforced by the body limit middleware to prevent excessive memory usage from large payloads.
	MaxBodyBytes int64


	// CORSAllowAllOrigins when true, allows CORS requests from all origins (Access-Control-Allow-Origin: *).
	// Cannot be combined with CORSAllowedOrigins.
	CORSAllowAllOrigins bool

	// CORSAllowedOrigins is a list of specific origins allowed for CORS requests.
	// Example: []string{"https://app.example.com"}. Mutually exclusive with CORSAllowAllOrigins.
	CORSAllowedOrigins []string

	// CORSAllowedHeaders is a list of HTTP headers that are allowed in CORS requests.
	// Example: []string{"Authorization", "Content-Type"}.
	CORSAllowedHeaders []string

	// EnableHSTS when true, enables HTTP Strict Transport Security (HSTS) headers
	// to force clients to use HTTPS.
	EnableHSTS bool

	// HSTSMaxAgeSeconds is the HSTS max-age directive in seconds.
	// Tells browsers how long to remember that the site should only be accessed via HTTPS.
	// Must be greater than 0 when HSTS is enabled.
	HSTSMaxAgeSeconds int

	// Version is the application version string, typically injected at build time via -ldflags
	// from 'git describe'. Exposed in the /health endpoint response.
	Version string

	// Commit is the git commit hash, typically injected at build time via -ldflags
	// from 'git rev-parse HEAD'. Exposed in the /health endpoint response.
	Commit string

	// TrustedProxies is a list of trusted proxy IP addresses or CIDR ranges.
	// Used by Gin to correctly parse client IP addresses from X-Forwarded-For headers
	// when behind a reverse proxy.
	TrustedProxies []string

	// ShutdownCh is closed when the application is shutting down.
	// Used by background goroutines to know when to stop.
	ShutdownCh <-chan struct{}

	// OTelServiceName is the service name used by the OpenTelemetry tracing middleware.
	OTelServiceName string
}
