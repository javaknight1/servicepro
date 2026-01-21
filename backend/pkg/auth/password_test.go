package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHashPassword(t *testing.T) {
	password := "SecurePassword123!"

	hash, err := HashPassword(password)
	assert.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.NotEqual(t, password, hash)
}

func TestComparePassword_Success(t *testing.T) {
	password := "SecurePassword123!"
	hash, _ := HashPassword(password)

	err := ComparePassword(hash, password)
	assert.NoError(t, err)
}

func TestComparePassword_Failure(t *testing.T) {
	password := "SecurePassword123!"
	wrongPassword := "WrongPassword456!"
	hash, _ := HashPassword(password)

	err := ComparePassword(hash, wrongPassword)
	assert.Error(t, err)
}

func TestComparePassword_InvalidHash(t *testing.T) {
	password := "SecurePassword123!"
	invalidHash := "not-a-valid-hash"

	err := ComparePassword(invalidHash, password)
	assert.Error(t, err)
}
