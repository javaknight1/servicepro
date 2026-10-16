// Package apierror writes the standard API error envelope (T223):
//
//	{"error": "<snake_case code>", "message": "<safe, human text>",
//	 "request_id": "<X-Request-ID>", "fields": {...}?, "details": [...]?}
//
// See docs/api/errors.md for the contract. Handlers and middleware write
// every error response through this package so the shape, the request ID
// and the "no internals in responses" rule (T116) hold everywhere.
package apierror

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"regexp"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"

	apperrors "github.com/javaknight1/servicepro/backend/internal/errors"
	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/pkg/clients/logging"
)

// Codes for request-shape failures.
const (
	CodeMalformedJSON  = "malformed_json"
	CodeInvalidRequest = "invalid_request"
)

// codePattern is what a machine code looks like. Legacy call sites that put
// a sentence in "error" are normalised: the sentence becomes the message and
// the code is derived from the HTTP status.
var codePattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

func init() {
	// Report validation problems under the request's JSON field names
	// ("customer_id"), not Go struct field names ("CustomerID").
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterTagNameFunc(jsonFieldName)
	}
}

func jsonFieldName(f reflect.StructField) string {
	for _, tag := range []string{"json", "form", "uri"} {
		name := strings.SplitN(f.Tag.Get(tag), ",", 2)[0]
		if name == "-" {
			return ""
		}
		if name != "" {
			return name
		}
	}
	return f.Name
}

// RequestID returns the request's correlation ID (set by middleware.RequestID).
func RequestID(c *gin.Context) string {
	if c.Request != nil {
		if id := logging.ExtractRequestID(c.Request.Context()); id != "" {
			return id
		}
	}
	return c.GetString("request_id")
}

// Write sends an error envelope with the given status, code and message and
// aborts the handler chain.
func Write(c *gin.Context, status int, code, message string) {
	WriteResponse(c, status, models.ErrorResponse{Error: code, Message: message})
}

// WriteResponse sends resp as the error envelope and aborts the chain. It
// fills request_id and normalises the body: "error" is always a snake_case
// code and "message" is always set.
func WriteResponse(c *gin.Context, status int, resp models.ErrorResponse) {
	c.AbortWithStatusJSON(status, Normalize(c, status, resp))
}

// Normalize returns resp in envelope form (see WriteResponse).
func Normalize(c *gin.Context, status int, resp models.ErrorResponse) models.ErrorResponse {
	if !codePattern.MatchString(resp.Error) {
		if resp.Message == "" {
			resp.Message = resp.Error
		}
		resp.Error = apperrors.CodeForStatus(status)
	}
	if resp.Message == "" {
		resp.Message = humanize(resp.Error)
	}
	if resp.RequestID == "" {
		resp.RequestID = RequestID(c)
	}
	return resp
}

// humanize turns a code into a fallback message: "invalid_tenant_id" ->
// "Invalid tenant id".
func humanize(code string) string {
	s := strings.ReplaceAll(code, "_", " ")
	if s == "" {
		return "Error"
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// Respond maps err to the envelope:
//   - a domain error (internal/errors) -> its status, code, message, fields;
//   - validation / malformed JSON errors -> 400;
//   - anything else is an infrastructure failure -> logged with the request
//     ID, answered with a generic 500 (internalMessage, or a default if "").
func Respond(c *gin.Context, err error, internalMessage string) {
	if err == nil {
		return
	}
	if de, ok := apperrors.As(err); ok {
		WriteResponse(c, de.Status, models.ErrorResponse{Error: de.Code, Message: de.Message, Fields: de.Fields})
		return
	}
	if writeRequestShapeError(c, err) {
		return
	}
	Internal(c, err, internalMessage)
}

// Internal logs err (with the request ID) and answers 500 with a safe
// message; err's text never reaches the client.
func Internal(c *gin.Context, err error, message string) {
	if message == "" {
		message = apperrors.MessageForStatus(http.StatusInternalServerError)
	}
	fields := map[string]any{"method": c.Request.Method, "path": c.FullPath(), "status": http.StatusInternalServerError}
	if err != nil {
		fields["error"] = err.Error()
	}
	logging.Error(c.Request.Context(), "[API] "+message, fields)
	Write(c, http.StatusInternalServerError, apperrors.CodeInternal, message)
}

// BindError answers a failed ShouldBind*/ShouldBindJSON with 400: per-field
// problems for validation errors, "malformed_json" for unparsable bodies,
// and a generic invalid_request otherwise.
func BindError(c *gin.Context, err error) {
	if de, ok := apperrors.As(err); ok {
		WriteResponse(c, de.Status, models.ErrorResponse{Error: de.Code, Message: de.Message, Fields: de.Fields})
		return
	}
	if writeRequestShapeError(c, err) {
		return
	}
	Write(c, http.StatusBadRequest, CodeInvalidRequest, "The request body is invalid")
}

// writeRequestShapeError handles validation and JSON-decoding errors.
func writeRequestShapeError(c *gin.Context, err error) bool {
	var ve validator.ValidationErrors
	if errors.As(err, &ve) {
		fields := FieldErrors(ve)
		WriteResponse(c, http.StatusBadRequest, models.ErrorResponse{
			Error:   apperrors.CodeValidation,
			Message: validationSummary(fields),
			Fields:  fields,
		})
		return true
	}
	var syntaxErr *json.SyntaxError
	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &typeErr) {
		field := typeErr.Field
		if field == "" {
			field = "body"
		}
		WriteResponse(c, http.StatusBadRequest, models.ErrorResponse{
			Error:   apperrors.CodeValidation,
			Message: fmt.Sprintf("Validation failed: %s has the wrong type", field),
			Fields:  map[string]string{field: fmt.Sprintf("%s must be a %s", field, typeErr.Type.String())},
		})
		return true
	}
	if errors.As(err, &syntaxErr) || errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		Write(c, http.StatusBadRequest, CodeMalformedJSON, "Malformed JSON request body")
		return true
	}
	return false
}

// FieldErrors describes each failed validation rule under its JSON field name.
func FieldErrors(ve validator.ValidationErrors) map[string]string {
	fields := make(map[string]string, len(ve))
	for _, fe := range ve {
		name := fe.Field()
		if _, seen := fields[name]; seen {
			continue
		}
		fields[name] = describe(name, fe)
	}
	return fields
}

func describe(field string, fe validator.FieldError) string {
	switch fe.Tag() {
	case "required", "required_if", "required_unless", "required_with", "required_without":
		return field + " is required"
	case "email":
		return field + " must be a valid email address"
	case "uuid", "uuid4":
		return field + " must be a valid ID"
	case "min":
		return fmt.Sprintf("%s must be at least %s", field, fe.Param())
	case "max":
		return fmt.Sprintf("%s must be at most %s", field, fe.Param())
	case "len":
		return fmt.Sprintf("%s must be %s long", field, fe.Param())
	case "gt":
		return fmt.Sprintf("%s must be greater than %s", field, fe.Param())
	case "gte":
		return fmt.Sprintf("%s must be at least %s", field, fe.Param())
	case "lt":
		return fmt.Sprintf("%s must be less than %s", field, fe.Param())
	case "lte":
		return fmt.Sprintf("%s must be at most %s", field, fe.Param())
	case "oneof":
		return fmt.Sprintf("%s must be one of: %s", field, strings.ReplaceAll(fe.Param(), " ", ", "))
	case "url", "http_url":
		return field + " must be a valid URL"
	case "e164":
		return field + " must be a phone number in international format"
	default:
		return fmt.Sprintf("%s is invalid (%s)", field, fe.Tag())
	}
}

// validationSummary is a one-line message listing the field problems, for
// clients that only show "message".
func validationSummary(fields map[string]string) string {
	if len(fields) == 0 {
		return "Validation failed"
	}
	names := make([]string, 0, len(fields))
	for name := range fields {
		names = append(names, name)
	}
	sort.Strings(names)
	parts := make([]string, 0, len(names))
	for _, n := range names {
		parts = append(parts, fields[n])
	}
	return "Validation failed: " + strings.Join(parts, "; ")
}
