package security

import (
	"html"
	"regexp"
	"strings"
)

// Sanitizer provides input sanitization methods
type Sanitizer struct{}

// NewSanitizer creates a new sanitizer instance
func NewSanitizer() *Sanitizer {
	return &Sanitizer{}
}

// SanitizeString removes potentially dangerous characters and HTML
func (s *Sanitizer) SanitizeString(input string) string {
	// Trim whitespace
	input = strings.TrimSpace(input)

	// HTML escape to prevent XSS
	input = html.EscapeString(input)

	// Remove null bytes
	input = strings.ReplaceAll(input, "\x00", "")

	return input
}

// SanitizeEmail sanitizes and validates email format
func (s *Sanitizer) SanitizeEmail(email string) string {
	// Trim and lowercase
	email = strings.ToLower(strings.TrimSpace(email))

	// HTML escape
	email = html.EscapeString(email)

	return email
}

// SanitizeUsername sanitizes username (alphanumeric, underscores, hyphens only)
func (s *Sanitizer) SanitizeUsername(username string) string {
	// Trim whitespace
	username = strings.TrimSpace(username)

	// HTML escape
	username = html.EscapeString(username)

	// Remove any characters that aren't alphanumeric, underscore, or hyphen
	reg := regexp.MustCompile(`[^a-zA-Z0-9_\-]`)
	username = reg.ReplaceAllString(username, "")

	return username
}

// SanitizePhone removes non-numeric characters except + and spaces
func (s *Sanitizer) SanitizePhone(phone string) string {
	// Remove all characters except digits, +, and spaces
	reg := regexp.MustCompile(`[^0-9+\s]`)
	phone = reg.ReplaceAllString(phone, "")

	return strings.TrimSpace(phone)
}

// RemoveScriptTags removes script tags from input
func (s *Sanitizer) RemoveScriptTags(input string) string {
	// Remove script tags
	scriptRegex := regexp.MustCompile(`(?i)<script[^>]*>.*?</script>`)
	input = scriptRegex.ReplaceAllString(input, "")

	// Remove on* event handlers
	eventRegex := regexp.MustCompile(`(?i)\s*on\w+\s*=\s*["'][^"']*["']`)
	input = eventRegex.ReplaceAllString(input, "")

	return input
}

// SanitizeHTML basic HTML sanitization (for user-generated content)
func (s *Sanitizer) SanitizeHTML(input string) string {
	// Remove script tags
	input = s.RemoveScriptTags(input)

	// Remove potentially dangerous tags
	dangerousTags := []string{"iframe", "object", "embed", "link", "style"}
	for _, tag := range dangerousTags {
		regex := regexp.MustCompile(`(?i)<` + tag + `[^>]*>.*?</` + tag + `>`)
		input = regex.ReplaceAllString(input, "")
	}

	return input
}

// ValidateSQLInjection checks for common SQL injection patterns
func (s *Sanitizer) ValidateSQLInjection(input string) bool {
	// Common SQL injection patterns
	sqlPatterns := []string{
		`(?i)\bunion\b.*\bselect\b`,
		`(?i)\bselect\b.*\bfrom\b`,
		`(?i)\binsert\b.*\binto\b`,
		`(?i)\bdelete\b.*\bfrom\b`,
		`(?i)\bdrop\b.*\btable\b`,
		`(?i)--`,
		`(?i)/\*.*\*/`,
		`(?i)\bor\b.*=.*`,
		`(?i)\band\b.*=.*`,
		`'.*--`,
		`'.*OR.*'.*'`,
	}

	for _, pattern := range sqlPatterns {
		matched, _ := regexp.MatchString(pattern, input)
		if matched {
			return false // Potential SQL injection detected
		}
	}

	return true // Input appears safe
}

// ValidateNoPath checks if input contains path traversal attempts
func (s *Sanitizer) ValidateNoPath(input string) bool {
	// Check for path traversal patterns
	pathPatterns := []string{
		`\.\.\/`,
		`\.\.\\`,
		`%2e%2e%2f`,
		`%2e%2e\/`,
		`\.\.%2f`,
	}

	for _, pattern := range pathPatterns {
		matched, _ := regexp.MatchString(pattern, strings.ToLower(input))
		if matched {
			return false
		}
	}

	return true
}
