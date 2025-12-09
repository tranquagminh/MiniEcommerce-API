package auth

import (
	"fmt"
	"regexp"
	"strings"
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

// Common weak passwords list - Top 50 most common passwords
// Source: Compiled from common password breach databases
var commonPasswords = map[string]bool{
	// Numeric sequences
	"123456":     true,
	"12345678":   true,
	"123456789":  true,
	"1234567890": true,
	"12341234":   true,
	"11111111":   true,
	"00000000":   true,
	"87654321":   true,

	// Keyboard patterns
	"qwerty":     true,
	"qwertyuiop": true,
	"asdfghjkl":  true,
	"zxcvbnm":    true,
	"qwerty123":  true,
	"1q2w3e4r":   true,
	"1q2w3e4r5t": true,
	"zaq12wsx":   true,

	// Dictionary words
	"password":      true,
	"password1":     true,
	"password123":   true,
	"passw0rd":      true,
	"admin":         true,
	"admin123":      true,
	"administrator": true,
	"root":          true,
	"letmein":       true,
	"welcome":       true,
	"welcome1":      true,
	"monkey":        true,
	"dragon":        true,
	"master":        true,
	"sunshine":      true,
	"princess":      true,
	"football":      true,
	"baseball":      true,
	"superman":      true,
	"batman":        true,

	// Common patterns
	"abc123":    true,
	"abc12345":  true,
	"iloveyou":  true,
	"trustno1":  true,
	"login":     true,
	"Password1": true,
	"qazwsx":    true,
	"starwars":  true,

	// Years and dates
	"20202020": true,
	"20212021": true,
	"20222022": true,
	"20232023": true,
	"20242024": true,
}

// Weak patterns that should be detected even as substrings
// Only very weak patterns like pure numeric sequences and keyboard patterns
var weakSubstringPatterns = []string{
	"123456789",
	"12345678",
	"1234567890",
	"qwertyuiop",
	"asdfghjkl",
}

func isCommonPassword(password string) bool {
	// Convert to lowercase for comparison
	lowerPassword := ""
	for _, r := range password {
		lowerPassword += string(unicode.ToLower(r))
	}

	// Check exact match against common passwords
	if commonPasswords[lowerPassword] {
		return true
	}

	// Check if password starts with a common weak password followed by simple additions
	// This catches "Password123!" but not "MySecurePassword123!"
	for commonPwd := range commonPasswords {
		if len(commonPwd) >= 6 {
			// Check if password starts with common password (case-insensitive)
			if strings.HasPrefix(lowerPassword, commonPwd) {
				return true
			}
		}
	}

	// Check for very weak substring patterns (numeric sequences, keyboard patterns)
	for _, pattern := range weakSubstringPatterns {
		if strings.Contains(lowerPassword, pattern) {
			return true
		}
	}

	return false
}
