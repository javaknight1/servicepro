package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
)

// genericErrorMessage replaces error text that may carry internals.
const genericErrorMessage = "The request could not be processed"

// internalErrorPattern matches driver/ORM/infrastructure details that must
// never reach a client (SQL, table/column names, hosts, stack-ish text).
var internalErrorPattern = regexp.MustCompile(`(?i)(sql|sqlstate|pq:|pgx|gorm|relation |column |constraint|duplicate key|syntax error|dial |connection|timeout|redis|stripe|context deadline|record not found|panic|runtime error|no such|violates|ERROR:)`)

// publicError maps an error to a message that is safe to return to clients
// (audit F-17). Validation and malformed-JSON errors are described; plain
// sentinel errors from our services (short, unwrapped messages such as
// "payment amount exceeds outstanding balance") pass through; anything
// wrapped ("failed to X: <cause>") or mentioning internals becomes a generic
// message. Callers still log the full error server-side.
func publicError(err error) string {
	if err == nil {
		return ""
	}

	var ve validator.ValidationErrors
	if errors.As(err, &ve) {
		parts := make([]string, 0, len(ve))
		for _, fe := range ve {
			parts = append(parts, fmt.Sprintf("%s is invalid (%s)", strings.ToLower(fe.Field()), fe.Tag()))
		}
		return "Validation failed: " + strings.Join(parts, "; ")
	}

	var syntaxErr *json.SyntaxError
	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &syntaxErr) || errors.As(err, &typeErr) || errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return "Malformed JSON request body"
	}

	msg := err.Error()
	if len(msg) > 160 || strings.Contains(msg, ": ") || internalErrorPattern.MatchString(msg) {
		return genericErrorMessage
	}
	return msg
}
