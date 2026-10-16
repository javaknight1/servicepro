package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPublicError(t *testing.T) {
	assert.Equal(t, "payment amount exceeds outstanding balance",
		publicError(errors.New("payment amount exceeds outstanding balance")), "plain sentinel passes through")

	db := errors.New(`ERROR: relation "invoices" does not exist (SQLSTATE 42P01)`)
	assert.Equal(t, genericErrorMessage, publicError(db))
	assert.Equal(t, genericErrorMessage, publicError(fmt.Errorf("failed to get invoice: %w", db)))
	assert.Equal(t, genericErrorMessage, publicError(errors.New("dial tcp 10.0.0.5:5432: connection refused")))

	var syntax *json.SyntaxError
	err := json.Unmarshal([]byte("{"), &struct{}{})
	if errors.As(err, &syntax) || err != nil {
		assert.Equal(t, "Malformed JSON request body", publicError(err))
	}
	assert.Equal(t, "", publicError(nil))
}
