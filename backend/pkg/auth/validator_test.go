package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidatePassword_Success(t *testing.T) {
	password := "SecurePass123!"
	err := ValidatePassword(password)
	assert.NoError(t, err)
}

func TestValidatePassword_TooShort(t *testing.T) {
	password := "Short1!"
	err := ValidatePassword(password)
	assert.Equal(t, ErrPasswordTooShort, err)
}

func TestValidatePassword_NoUppercase(t *testing.T) {
	password := "lowercase123!"
	err := ValidatePassword(password)
	assert.Equal(t, ErrPasswordNoUppercase, err)
}

func TestValidatePassword_NoLowercase(t *testing.T) {
	password := "UPPERCASE123!"
	err := ValidatePassword(password)
	assert.Equal(t, ErrPasswordNoLowercase, err)
}

func TestValidatePassword_NoNumber(t *testing.T) {
	password := "NoNumbers!"
	err := ValidatePassword(password)
	assert.Equal(t, ErrPasswordNoNumber, err)
}

func TestValidatePassword_NoSpecial(t *testing.T) {
	password := "NoSpecial123"
	err := ValidatePassword(password)
	assert.Equal(t, ErrPasswordNoSpecial, err)
}

func TestValidatePassword_AllRequirements(t *testing.T) {
	testCases := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{"Valid password", "MyPass123!", false},
		{"Valid with special chars", "Test@Pass123", false},
		{"Valid with symbols", "Secure#Pass99", false},
		{"Too short", "Abc1!", true},
		{"No uppercase", "mypass123!", true},
		{"No lowercase", "MYPASS123!", true},
		{"No number", "MyPassword!", true},
		{"No special", "MyPassword123", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidatePassword(tc.password)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateEmail_Success(t *testing.T) {
	validEmails := []string{
		"user@example.com",
		"john.doe@company.co.uk",
		"test+tag@domain.com",
		"user_name@sub.domain.com",
	}

	for _, email := range validEmails {
		t.Run(email, func(t *testing.T) {
			err := ValidateEmail(email)
			assert.NoError(t, err)
		})
	}
}

func TestValidateEmail_Invalid(t *testing.T) {
	invalidEmails := []string{
		"notanemail",
		"@example.com",
		"user@",
		"user@.com",
		"user @example.com",
		"user@example",
		"",
	}

	for _, email := range invalidEmails {
		t.Run(email, func(t *testing.T) {
			err := ValidateEmail(email)
			assert.Equal(t, ErrInvalidEmailFormat, err)
		})
	}
}
