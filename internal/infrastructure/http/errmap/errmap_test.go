package errmap

import (
	"context"
	"errors"
	"net/http"
	"testing"

	apperror "github.com/rachelJG/event-notification-service/internal/pkg/apperror"
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
		{"canceled_apperror", apperror.Canceled("aborted", nil), StatusClientClosedRequest, "canceled"},
		{"unavailable", apperror.Unavailable("down", nil), http.StatusServiceUnavailable, "unavailable"},
		{"rate_limited", apperror.New(apperror.CodeRateLimited, "slow down", nil), http.StatusTooManyRequests, "rate_limited"},
		{"deadline", context.DeadlineExceeded, http.StatusGatewayTimeout, "timeout"},
		{"canceled", context.Canceled, StatusClientClosedRequest, "canceled"},
		{"unknown_apperror", apperror.New("unknown_code", "unknown", nil), http.StatusInternalServerError, "internal"},
		{"generic_error", errors.New("something"), http.StatusInternalServerError, "internal"},
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

func TestMessageMapping(t *testing.T) {
	cases := []struct {
		name    string
		err     error
		wantMsg string
	}{
		{"canceled", context.Canceled, "request canceled"},
		{"deadline_exceeded", context.DeadlineExceeded, "request timeout"},
		{"apperror_with_message", apperror.InvalidArgument("bad input", nil), "bad input"},
		{"apperror_empty_message", apperror.New(apperror.CodeInternal, "", nil), "internal error"},
		{"generic_error", errors.New("something"), "internal error"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Message(tc.err)
			if got != tc.wantMsg {
				t.Fatalf("Message() = %q, want %q", got, tc.wantMsg)
			}
		})
	}
}
