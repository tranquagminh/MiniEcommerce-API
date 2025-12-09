package auth

import (
	"fmt"
	"regexp"
	"unicode"
)

// PasswordValidator validates password strength
type PasswordValidator struct {
	MinLength          int
	RequireUppercase   bool
	RequireLowercase   bool
	RequireNumbers     bool
	RequireSpecialChar bool
}

// DefaultPasswordValidator returns a validator with secure defaults
func DefaultPasswordValidator() *PasswordValidator {
	return &PasswordValidator{
		MinLength:          8,
		RequireUppercase:   true,
		RequireLowercase:   true,
		RequireNumbers:     true,
		RequireSpecialChar: true,
	}
}

// Validate checks if a password meets the strength requirements
func (v *PasswordValidator) Validate(password string) error {
	if len(password) < v.MinLength {
		return fmt.Errorf("password must be at least %d characters long", v.MinLength)
	}

	if v.RequireUppercase && !hasUppercase(password) {
		return fmt.Errorf("password must contain at least one uppercase letter")
	}

	if v.RequireLowercase && !hasLowercase(password) {
		return fmt.Errorf("password must contain at least one lowercase letter")
	}

	if v.RequireNumbers && !hasNumber(password) {
		return fmt.Errorf("password must contain at least one number")
	}

	if v.RequireSpecialChar && !hasSpecialChar(password) {
		return fmt.Errorf("password must contain at least one special character")
	}

	// Check for common weak passwords
	if isCommonPassword(password) {
		return fmt.Errorf("password is too common, please choose a stronger password")
	}

	return nil
}

func hasUppercase(s string) bool {
	for _, r := range s {
		if unicode.IsUpper(r) {
			return true
		}
	}
	return false
}

func hasLowercase(s string) bool {
	for _, r := range s {
		if unicode.IsLower(r) {
			return true
		}
	}
	return false
}

func hasNumber(s string) bool {
	for _, r := range s {
		if unicode.IsDigit(r) {
			return true
		}
	}
	return false
}

func hasSpecialChar(s string) bool {
	// Check for common special characters
	specialChars := regexp.MustCompile(`[!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?~` + "`" + `]`)
	return specialChars.MatchString(s)
}

// Common weak passwords list (add more as needed)
var commonPasswords = map[string]bool{
	"password":   true,
	"12345678":   true,
	"123456789":  true,
	"qwerty":     true,
	"abc123":     true,
	"password1":  true,
	"12341234":   true,
	"qwerty123":  true,
	"1q2w3e4r":   true,
	"admin":      true,
	"letmein":    true,
	"welcome":    true,
	"monkey":     true,
	"1234567890": true,
}

func isCommonPassword(password string) bool {
	// Convert to lowercase for comparison
	lowerPassword := ""
	for _, r := range password {
		lowerPassword += string(unicode.ToLower(r))
	}
	return commonPasswords[lowerPassword]
}
