package postgres

import (
	"context"
	"testing"
	"time"
)

func TestWithQueryTimeout(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		timeout        time.Duration
		wantDeadline   bool
		cancelShouldOp bool // cancel function should be callable without panic
	}{
		{
			name:           "positive timeout sets deadline",
			timeout:        5 * time.Second,
			wantDeadline:   true,
			cancelShouldOp: true,
		},
		{
			name:           "zero timeout returns original context",
			timeout:        0,
			wantDeadline:   false,
			cancelShouldOp: true,
		},
		{
			name:           "negative timeout returns original context",
			timeout:        -1 * time.Second,
			wantDeadline:   false,
			cancelShouldOp: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			parent := context.Background()
			ctx, cancel := withQueryTimeout(parent, tt.timeout)
			defer cancel()

			_, hasDeadline := ctx.Deadline()
			if hasDeadline != tt.wantDeadline {
				t.Errorf("Deadline() present = %v, want %v", hasDeadline, tt.wantDeadline)
			}
		})
	}
}

func TestWithQueryTimeout_PreservesParentValues(t *testing.T) {
	t.Parallel()

	type ctxKey string
	key := ctxKey("test-key")
	parent := context.WithValue(context.Background(), key, "test-value")

	ctx, cancel := withQueryTimeout(parent, 5*time.Second)
	defer cancel()

	got := ctx.Value(key)
	if got != "test-value" {
		t.Errorf("expected parent value to be preserved, got %v", got)
	}
}

func TestWithQueryTimeout_ZeroReturnsExactParent(t *testing.T) {
	t.Parallel()

	parent := context.Background()
	ctx, cancel := withQueryTimeout(parent, 0)
	defer cancel()

	if ctx != parent {
		t.Error("expected zero timeout to return the exact parent context")
	}
}

func TestWithQueryTimeout_CancelStopsContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := withQueryTimeout(context.Background(), 10*time.Second)
	cancel()

	select {
	case <-ctx.Done():
		// expected
	default:
		t.Error("expected context to be canceled after cancel() call")
	}
}
