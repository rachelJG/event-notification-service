package postgres

import (
	"context"
	"time"
)

// withQueryTimeout wraps the given context with a timeout if the duration is
// positive. The caller must always call the returned cancel function.
func withQueryTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout > 0 {
		return context.WithTimeout(ctx, timeout)
	}
	return ctx, func() {}
}
