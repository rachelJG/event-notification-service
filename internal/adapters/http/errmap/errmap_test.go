package errmap

import (
	"context"
	"net/http"
	"testing"

	"github.com/rachelJG/event-notification-service/internal/core/apperror"
)

func TestFromErrorMapping(t *testing.T) {
	cases := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{"invalid_argument", apperror.InvalidArgument("bad", nil), http.StatusBadRequest, "invalid_argument"},
		{"unauthenticated", apperror.Unauthenticated("nope", nil), http.StatusUnauthorized, "unauthenticated"},
		{"permission_denied", apperror.PermissionDenied("nope", nil), http.StatusForbidden, "permission_denied"},
		{"not_found", apperror.NotFound("missing", nil), http.StatusNotFound, "not_found"},
		{"conflict", apperror.Conflict("conflict", nil), http.StatusConflict, "conflict"},
		{"timeout", apperror.Timeout("slow", nil), http.StatusGatewayTimeout, "timeout"},
		{"deadline", context.DeadlineExceeded, http.StatusGatewayTimeout, "timeout"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := FromError(tc.err)
			if got.Status != tc.wantStatus {
				t.Fatalf("status: got %d, want %d", got.Status, tc.wantStatus)
			}
			if got.Code != tc.wantCode {
				t.Fatalf("code: got %s, want %s", got.Code, tc.wantCode)
			}
		})
	}
}
