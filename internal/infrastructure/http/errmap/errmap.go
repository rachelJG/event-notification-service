package errmap

import (
	"context"
	"errors"
	"net/http"

	apperror "github.com/rachelJG/event-notification-service/internal/domain/errors"
)

type HTTPError struct {
	Status int
	Code   string
}

func FromError(err error) HTTPError {
	if errors.Is(err, context.DeadlineExceeded) {
		return HTTPError{Status: http.StatusGatewayTimeout, Code: "timeout"}
	}
	var appErr *apperror.AppError
	if errors.As(err, &appErr) {
		switch appErr.Code {
		case apperror.CodeInvalidArgument:
			return HTTPError{Status: http.StatusBadRequest, Code: string(appErr.Code)}
		case apperror.CodeUnauthenticated:
			return HTTPError{Status: http.StatusUnauthorized, Code: string(appErr.Code)}
		case apperror.CodePermissionDenied:
			return HTTPError{Status: http.StatusForbidden, Code: string(appErr.Code)}
		case apperror.CodeNotFound:
			return HTTPError{Status: http.StatusNotFound, Code: string(appErr.Code)}
		case apperror.CodeConflict:
			return HTTPError{Status: http.StatusConflict, Code: string(appErr.Code)}
		case apperror.CodeTimeout:
			return HTTPError{Status: http.StatusGatewayTimeout, Code: string(appErr.Code)}
		default:
			return HTTPError{Status: http.StatusInternalServerError, Code: string(apperror.CodeInternal)}
		}
	}
	return HTTPError{Status: http.StatusInternalServerError, Code: string(apperror.CodeInternal)}
}

func Message(err error) string {
	if errors.Is(err, context.DeadlineExceeded) {
		return "request timeout"
	}
	var appErr *apperror.AppError
	if errors.As(err, &appErr) && appErr.Message != "" {
		return appErr.Message
	}
	return "internal error"
}
