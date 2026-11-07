package httpx

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	middleware "github.com/oapi-codegen/nethttp-middleware"
)

func Validator(spec *openapi3.T, logger *slog.Logger) func(http.Handler) http.Handler {
	return middleware.OapiRequestValidatorWithOptions(spec, validatorOptions(logger))
}

func validatorOptions(logger *slog.Logger) *middleware.Options {
	return &middleware.Options{
		DoNotValidateServers: true,
		Options: openapi3filter.Options{
			AuthenticationFunc: openapi3filter.NoopAuthenticationFunc,
		},
		ErrorHandlerWithOpts: rejectRequest(logger),
	}
}

func rejectRequest(logger *slog.Logger) middleware.ErrorHandlerWithOpts {
	return func(_ context.Context, _ error, w http.ResponseWriter, _ *http.Request, opts middleware.ErrorHandlerOpts) {
		WriteError(w, logger, validationFailed(opts.StatusCode))
	}
}

func validationFailed(status int) *Error {
	return &Error{
		Status:  status,
		Code:    "validation_failed",
		Message: "request does not match the API contract",
	}
}

func StrictErrorHandler(logger *slog.Logger) func(http.ResponseWriter, *http.Request, error) {
	return func(w http.ResponseWriter, _ *http.Request, err error) {
		WriteError(w, logger, err)
	}
}
