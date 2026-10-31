package errors

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"reflect"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type ApiError struct {
	Code    string       `json:"code"`
	Message string       `json:"message"`
	Details []FieldError `json:"details,omitempty"`
}

type FieldError struct {
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

// Respond writes the right error response for any error and stops the handler chain.
func Respond(c *gin.Context, err error) {
	var appErr *AppError
	if errors.As(err, &appErr) {
		ErrorResponse(c, appErr.Status, appErr.Code, appErr.Message)
		return
	}
	if isBindError(err) {
		BindError(c, err)
		return
	}
	log.Printf("internal error: %v", err)
	ErrorResponse(c, http.StatusInternalServerError, "internal_error", "internal server error")
}

// isBindError reports whether err came from ShouldBindJSON.
func isBindError(err error) bool {
	var (
		verrs     validator.ValidationErrors
		typeErr   *json.UnmarshalTypeError
		syntaxErr *json.SyntaxError
	)
	return errors.As(err, &verrs) || errors.As(err, &typeErr) || errors.As(err, &syntaxErr) ||
		errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF)
}

// ErrorResponse writes the error and stops the handler chain.
func ErrorResponse(c *gin.Context, status int, code string, message string, details ...FieldError) {
	c.AbortWithStatusJSON(status, gin.H{"error": ApiError{
		Code:    code,
		Message: message,
		Details: details,
	}})
}

// BindError turns a ShouldBindJSON error into a 400 or 422 response.
func BindError(c *gin.Context, err error) {
	var (
		verrs     validator.ValidationErrors
		typeErr   *json.UnmarshalTypeError
		syntaxErr *json.SyntaxError
	)
	switch {
	case errors.As(err, &verrs):
		details := make([]FieldError, 0, len(verrs))
		for _, fe := range verrs {
			details = append(details, FieldError{Field: fe.Field(), Message: fieldMessage(fe)})
		}
		ErrorResponse(c, http.StatusUnprocessableEntity, "validation_failed", "request body is invalid", details...)
	case errors.As(err, &typeErr):
		ErrorResponse(c, http.StatusBadRequest, "invalid_json",
			fmt.Sprintf("field %q must be of type %s", typeErr.Field, typeErr.Type))
	case errors.As(err, &syntaxErr), errors.Is(err, io.ErrUnexpectedEOF):
		ErrorResponse(c, http.StatusBadRequest, "invalid_json", "request body is not valid JSON")
	case errors.Is(err, io.EOF):
		ErrorResponse(c, http.StatusBadRequest, "invalid_json", "request body is empty")
	default:
		ErrorResponse(c, http.StatusBadRequest, "bad_request", err.Error())
	}
}

// fieldMessage turns a validator rule into a human sentence.
func fieldMessage(fe validator.FieldError) string {
	unit := ""
	if fe.Kind() == reflect.String {
		unit = " characters"
	}
	switch fe.Tag() {
	case "required":
		return "is required"
	case "min":
		return "must be at least " + fe.Param() + unit
	case "max":
		return "must be at most " + fe.Param() + unit
	default:
		return "failed the " + fe.Tag() + " rule"
	}
}
