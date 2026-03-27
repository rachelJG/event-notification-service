package apperror

import (
	"errors"
	"testing"
)

func TestAppErrorMessageOnly(t *testing.T) {
	e := New(CodeInternal, "something broke", nil)
	if e.Error() != "something broke" {
		t.Errorf("Error() = %q, want %q", e.Error(), "something broke")
	}
	if e.Unwrap() != nil {
		t.Error("expected nil wrapped error")
	}
}

func TestAppErrorWithWrappedError(t *testing.T) {
	cause := errors.New("root cause")
	e := New(CodeInternal, "something broke", cause)
	want := "something broke: root cause"
	if e.Error() != want {
		t.Errorf("Error() = %q, want %q", e.Error(), want)
	}
	if e.Unwrap() != cause {
		t.Error("expected wrapped error to match cause")
	}
}

func TestConstructors(t *testing.T) {
	cases := []struct {
		name string
		fn   func(string, error) *AppError
		code Code
	}{
		{"InvalidArgument", InvalidArgument, CodeInvalidArgument},
		{"Unauthenticated", Unauthenticated, CodeUnauthenticated},
		{"PermissionDenied", PermissionDenied, CodePermissionDenied},
		{"NotFound", NotFound, CodeNotFound},
		{"Conflict", Conflict, CodeConflict},
		{"Timeout", Timeout, CodeTimeout},
		{"Canceled", Canceled, CodeCanceled},
		{"Unavailable", Unavailable, CodeUnavailable},
		{"Internal", Internal, CodeInternal},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := tc.fn("msg", nil)
			if e.Code != tc.code {
				t.Errorf("Code = %q, want %q", e.Code, tc.code)
			}
			if e.Message != "msg" {
				t.Errorf("Message = %q, want %q", e.Message, "msg")
			}
		})
	}
}

func TestErrorsAs(t *testing.T) {
	cause := errors.New("db error")
	e := Internal("failed", cause)

	var appErr *AppError
	if !errors.As(e, &appErr) {
		t.Fatal("expected errors.As to succeed")
	}
	if appErr.Code != CodeInternal {
		t.Errorf("Code = %q, want %q", appErr.Code, CodeInternal)
	}
}
