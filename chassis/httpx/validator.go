package httpx

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

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
	return func(_ context.Context, err error, w http.ResponseWriter, _ *http.Request, opts middleware.ErrorHandlerOpts) {
		switch opts.StatusCode {
		case http.StatusNotFound:
			WriteError(w, logger, &Error{Status: http.StatusNotFound, Code: "not_found", Message: "route not found"})
		case http.StatusBadRequest:
			WriteError(w, logger, validationFailed(validationFields(err)))
		default:
			WriteError(w, logger, err)
		}
	}
}

func validationFields(err error) map[string]string {
	field := "body"
	var requestErr *openapi3filter.RequestError
	if errors.As(err, &requestErr) && requestErr.Parameter != nil {
		field = requestErr.Parameter.Name
	}
	reason := "invalid"
	var schemaErr *openapi3.SchemaError
	if errors.As(err, &schemaErr) {
		if path := schemaErr.JSONPointer(); len(path) != 0 {
			field = strings.Join(path, ".")
		}
		if schemaErr.SchemaField != "" {
			reason = schemaErr.SchemaField
		}
	}
	return map[string]string{field: reason}
}

func validationFailed(fields map[string]string) *Error {
	return &Error{
		Status:  http.StatusBadRequest,
		Code:    "validation_failed",
		Message: "request does not match the API contract",
		Fields:  fields,
	}
}

func StrictRequestErrorHandler(logger *slog.Logger) func(http.ResponseWriter, *http.Request, error) {
	return func(w http.ResponseWriter, _ *http.Request, err error) {
		WriteError(w, logger, validationFailed(validationFields(err)))
	}
}

func StrictErrorHandler(logger *slog.Logger) func(http.ResponseWriter, *http.Request, error) {
	return func(w http.ResponseWriter, _ *http.Request, err error) {
		WriteError(w, logger, err)
	}
}
