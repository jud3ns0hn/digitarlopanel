package api

import (
	"errors"
	"unicode"

	"github.com/pquerna/otp/totp"
)

// verifyTOTP validates a 6-digit code against the given base32 secret.
func verifyTOTP(secret, code string) bool {
	if secret == "" || code == "" {
		return false
	}
	return totp.Validate(code, secret)
}

// validatePasswordStrength enforces a minimum password policy.
func validatePasswordStrength(pw string) error {
	if len(pw) < 10 {
		return errors.New("password must be at least 10 characters")
	}
	var hasLetter, hasDigit bool
	for _, r := range pw {
		switch {
		case unicode.IsLetter(r):
			hasLetter = true
		case unicode.IsDigit(r):
			hasDigit = true
		}
	}
	if !hasLetter || !hasDigit {
		return errors.New("password must contain both letters and digits")
	}
	return nil
}
