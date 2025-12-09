package security

import (
	"testing"
)

func TestSanitizer_SanitizeString(t *testing.T) {
	sanitizer := NewSanitizer()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "normal string",
			input: "Hello World",
			want:  "Hello World",
		},
		{
			name:  "with HTML tags",
			input: "<script>alert('XSS')</script>",
			want:  "&lt;script&gt;alert(&#39;XSS&#39;)&lt;/script&gt;",
		},
		{
			name:  "with quotes",
			input: `He said "Hello"`,
			want:  "He said &#34;Hello&#34;",
		},
		{
			name:  "with trailing spaces",
			input: "  Hello World  ",
			want:  "Hello World",
		},
		{
			name:  "with null bytes",
			input: "Hello\x00World",
			want:  "HelloWorld",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sanitizer.SanitizeString(tt.input); got != tt.want {
				t.Errorf("SanitizeString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSanitizer_SanitizeEmail(t *testing.T) {
	sanitizer := NewSanitizer()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "normal email",
			input: "user@example.com",
			want:  "user@example.com",
		},
		{
			name:  "uppercase email",
			input: "USER@EXAMPLE.COM",
			want:  "user@example.com",
		},
		{
			name:  "email with spaces",
			input: "  user@example.com  ",
			want:  "user@example.com",
		},
		{
			name:  "email with HTML",
			input: "<script>user@example.com</script>",
			want:  "&lt;script&gt;user@example.com&lt;/script&gt;",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sanitizer.SanitizeEmail(tt.input); got != tt.want {
				t.Errorf("SanitizeEmail() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSanitizer_SanitizeUsername(t *testing.T) {
	sanitizer := NewSanitizer()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "normal username",
			input: "john_doe",
			want:  "john_doe",
		},
		{
			name:  "username with hyphen",
			input: "john-doe",
			want:  "john-doe",
		},
		{
			name:  "username with spaces",
			input: "john doe",
			want:  "johndoe",
		},
		{
			name:  "username with special chars",
			input: "john@doe!",
			want:  "johndoe",
		},
		{
			name:  "alphanumeric username",
			input: "john123",
			want:  "john123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sanitizer.SanitizeUsername(tt.input); got != tt.want {
				t.Errorf("SanitizeUsername() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSanitizer_SanitizePhone(t *testing.T) {
	sanitizer := NewSanitizer()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "normal phone",
			input: "+1234567890",
			want:  "+1234567890",
		},
		{
			name:  "phone with spaces",
			input: "+1 234 567 890",
			want:  "+1 234 567 890",
		},
		{
			name:  "phone with dashes",
			input: "+1-234-567-890",
			want:  "+1234567890",
		},
		{
			name:  "phone with parentheses",
			input: "+1 (234) 567-890",
			want:  "+1 234 567890",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sanitizer.SanitizePhone(tt.input); got != tt.want {
				t.Errorf("SanitizePhone() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSanitizer_RemoveScriptTags(t *testing.T) {
	sanitizer := NewSanitizer()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "with script tag",
			input: "<script>alert('XSS')</script>Hello",
			want:  "Hello",
		},
		{
			name:  "with uppercase script tag",
			input: "<SCRIPT>alert('XSS')</SCRIPT>Hello",
			want:  "Hello",
		},
		{
			name:  "with onclick event",
			input: `<div onclick="alert('XSS')">Hello</div>`,
			want:  "<div>Hello</div>",
		},
		{
			name:  "normal text",
			input: "Hello World",
			want:  "Hello World",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sanitizer.RemoveScriptTags(tt.input); got != tt.want {
				t.Errorf("RemoveScriptTags() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSanitizer_DetectSQLInjection(t *testing.T) {
	sanitizer := NewSanitizer()

	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{
			name:  "normal input",
			input: "john_doe",
			want:  true,
		},
		{
			name:  "SQL injection - union select",
			input: "' UNION SELECT * FROM users--",
			want:  false,
		},
		{
			name:  "SQL injection - drop table",
			input: "'; DROP TABLE users;--",
			want:  false,
		},
		{
			name:  "SQL injection - comment",
			input: "admin'--",
			want:  false,
		},
		{
			name:  "SQL injection - or 1=1",
			input: "admin' OR '1'='1",
			want:  false,
		},
		{
			name:  "safe email",
			input: "user@example.com",
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sanitizer.DetectSQLInjection(tt.input); got != tt.want {
				t.Errorf("DetectSQLInjection() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSanitizer_ValidateNoPath(t *testing.T) {
	sanitizer := NewSanitizer()

	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{
			name:  "normal path",
			input: "files/document.pdf",
			want:  true,
		},
		{
			name:  "path traversal - double dot",
			input: "../../../etc/passwd",
			want:  false,
		},
		{
			name:  "path traversal - encoded",
			input: "..%2f..%2f..%2fetc%2fpasswd",
			want:  false,
		},
		{
			name:  "windows path traversal",
			input: "..\\..\\windows\\system32",
			want:  false,
		},
		{
			name:  "safe filename",
			input: "my_document.pdf",
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sanitizer.ValidateNoPath(tt.input); got != tt.want {
				t.Errorf("ValidateNoPath() = %v, want %v", got, tt.want)
			}
		})
	}
}
