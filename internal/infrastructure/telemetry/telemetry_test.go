package telemetry

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestInit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cfg  Config
	}{
		{
			name: "no exporter in production without endpoint",
			cfg: Config{
				ServiceName:    "test-service",
				ServiceVersion: "1.0.0",
				Environment:    "production",
				OTLPEndpoint:   "",
			},
		},
		{
			name: "stdout exporter in development",
			cfg: Config{
				ServiceName:    "test-service",
				ServiceVersion: "1.0.0",
				Environment:    "development",
				OTLPEndpoint:   "",
			},
		},
		{
			name: "otlp exporter with endpoint",
			cfg: Config{
				ServiceName:    "test-service",
				ServiceVersion: "2.0.0",
				Environment:    "staging",
				OTLPEndpoint:   "localhost:4318",
			},
		},
		{
			name: "empty service name",
			cfg: Config{
				ServiceName:    "",
				ServiceVersion: "",
				Environment:    "",
				OTLPEndpoint:   "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			shutdown, err := Init(ctx, tt.cfg)
			require.NoError(t, err, "Init should not return error")
			require.NotNil(t, shutdown, "shutdown function should not be nil")

			// Verify global tracer provider was set to an SDK provider.
			tp := otel.GetTracerProvider()
			assert.IsType(t, &sdktrace.TracerProvider{}, tp, "global tracer provider should be SDK type")

			// Shutdown should not error.
			err = shutdown(ctx)
			assert.NoError(t, err, "shutdown should not return error")
		})
	}
}

func TestInit_ShutdownIsIdempotent(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	shutdown, err := Init(ctx, Config{
		ServiceName: "idempotent-test",
		Environment: "test",
	})
	require.NoError(t, err)

	// First shutdown should succeed.
	require.NoError(t, shutdown(ctx))

	// Second shutdown should not panic or return a fatal error.
	// The SDK returns an error on double shutdown, which is acceptable.
	_ = shutdown(ctx)
}

func TestInit_CanceledContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	// Init with OTLP endpoint and canceled context — the exporter creation
	// may fail or succeed depending on whether it dials eagerly. Either way
	// it must not panic.
	shutdown, err := Init(ctx, Config{
		ServiceName:  "cancel-test",
		Environment:  "test",
		OTLPEndpoint: "localhost:4318",
	})

	if err != nil {
		// Acceptable: exporter creation failed due to canceled context.
		assert.Nil(t, shutdown, "shutdown should be nil when Init fails")
		return
	}

	// If Init succeeded despite canceled context, shutdown should still work.
	require.NotNil(t, shutdown)
	_ = shutdown(context.Background())
}

func TestInit_TracerProducesSpans(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	shutdown, err := Init(ctx, Config{
		ServiceName: "span-test",
		Environment: "test",
	})
	require.NoError(t, err)
	defer func() { _ = shutdown(ctx) }()

	// Create a span using the global tracer and verify it records correctly.
	tracer := otel.Tracer("test-tracer")
	_, span := tracer.Start(ctx, "test-operation")
	assert.True(t, span.SpanContext().IsValid(), "span context should be valid")
	assert.True(t, span.IsRecording(), "span should be recording")
	span.End()
}
