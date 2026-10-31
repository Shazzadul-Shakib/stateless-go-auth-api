// Package apperr turns any error into a consistent JSON error response for Gin.
//
// Response shape:
//
//	{"error": {"code": "validation_failed", "message": "...", "details": [{"field": "email", "message": "..."}]}}
package apperr

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"reflect"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

type apiError struct {
	Code    string       `json:"code"`
	Message string       `json:"message"`
	Details []fieldError `json:"details,omitempty"`
}

type fieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// AppError is an error that knows its own HTTP status and machine-readable code.
type AppError struct {
	Status  int
	Code    string
	Message string
}

func (e *AppError) Error() string { return e.Message }

func New(status int, code, message string) *AppError {
	return &AppError{Status: status, Code: code, Message: message}
}

// init makes validator report JSON field names ("email") instead of Go names ("Email").
func init() {
	v, ok := binding.Validator.Engine().(*validator.Validate)
	if !ok {
		return
	}
	v.RegisterTagNameFunc(func(f reflect.StructField) string {
		name, _, _ := strings.Cut(f.Tag.Get("json"), ",")
		switch name {
		case "-":
			return ""
		case "":
			return f.Name
		}
		return name
	})
}

// Respond writes the right error response for any error and stops the handler chain.
func Respond(c *gin.Context, err error) {
	if appErr, ok := errors.AsType[*AppError](err); ok {
		write(c, appErr.Status, appErr.Code, appErr.Message)
		return
	}

	if writeBindError(c, err) {
		return
	}

	slog.ErrorContext(c.Request.Context(), "request failed",
		"method", c.Request.Method,
		"route", c.FullPath(),
		"err", err,
	)
	write(c, http.StatusInternalServerError, "internal_error", "internal server error")
}

func write(c *gin.Context, status int, code, message string, details ...fieldError) {
	c.AbortWithStatusJSON(status, struct {
		Error apiError `json:"error"`
	}{apiError{Code: code, Message: message, Details: details}})
}

// writeBindError returns false if err is not a ShouldBindJSON error.
func writeBindError(c *gin.Context, err error) bool {
	if verrs, ok := errors.AsType[validator.ValidationErrors](err); ok {
		details := make([]fieldError, len(verrs))
		for i, fe := range verrs {
			details[i] = fieldError{Field: fe.Field(), Message: fieldMessage(fe)}
		}
		write(c, http.StatusUnprocessableEntity, "validation_failed", "request body is invalid", details...)
		return true
	}

	_, isSyntax := errors.AsType[*json.SyntaxError](err)
	typeErr, isType := errors.AsType[*json.UnmarshalTypeError](err)
	switch {
	case isType:
		write(c, http.StatusBadRequest, "invalid_json", `field "`+typeErr.Field+`" must be of type `+typeErr.Type.String())
	case isSyntax, errors.Is(err, io.ErrUnexpectedEOF):
		write(c, http.StatusBadRequest, "invalid_json", "request body is not valid JSON")
	case errors.Is(err, io.EOF):
		write(c, http.StatusBadRequest, "invalid_json", "request body is empty")
	default:
		return false
	}
	return true
}

// fieldMessage turns a validator rule into a human sentence.
func fieldMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "is required"
	case "email":
		return "must be a valid email address"
	case "min":
		return "must be at least " + fe.Param() + " characters"
	default:
		return "failed the " + fe.Tag() + " rule"
	}
}
