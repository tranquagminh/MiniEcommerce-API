package auth

import (
	"testing"
)

func TestPasswordValidator_Validate(t *testing.T) {
	validator := DefaultPasswordValidator()

	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{
			name:     "valid strong password",
			password: "SecureP@ss123",
			wantErr:  false,
		},
		{
			name:     "too short",
			password: "Short1!",
			wantErr:  true,
		},
		{
			name:     "no uppercase",
			password: "lowercase123!",
			wantErr:  true,
		},
		{
			name:     "no lowercase",
			password: "UPPERCASE123!",
			wantErr:  true,
		},
		{
			name:     "no numbers",
			password: "NoNumbers!",
			wantErr:  true,
		},
		{
			name:     "no special characters",
			password: "NoSpecial123",
			wantErr:  true,
		},
		{
			name:     "common password - password",
			password: "Password123!",
			wantErr:  true,
		},
		{
			name:     "common password - 12345678",
			password: "12345678",
			wantErr:  true,
		},
		{
			name:     "common password - qwerty",
			password: "Qwerty123!",
			wantErr:  true,
		},
		{
			name:     "valid complex password",
			password: "MyV3ry$ecureP@ssw0rd!",
			wantErr:  false,
		},
		{
			name:     "minimum valid length",
			password: "Abcd123!",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("PasswordValidator.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPasswordValidator_CustomRules(t *testing.T) {
	// Test with custom validator
	validator := &PasswordValidator{
		MinLength:          6,
		RequireUppercase:   false,
		RequireLowercase:   true,
		RequireNumbers:     true,
		RequireSpecialChar: false,
	}

	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{
			name:     "valid with custom rules",
			password: "simple123",
			wantErr:  false,
		},
		{
			name:     "too short for custom rules",
			password: "abc12",
			wantErr:  true,
		},
		{
			name:     "no lowercase",
			password: "UPPER123",
			wantErr:  true,
		},
		{
			name:     "no numbers",
			password: "nonnumbers",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("PasswordValidator.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHasUppercase(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"Hello", true},
		{"hello", false},
		{"HELLO", true},
		{"HeLLo", true},
		{"123", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := hasUppercase(tt.input); got != tt.want {
				t.Errorf("hasUppercase(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestHasLowercase(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"Hello", true},
		{"HELLO", false},
		{"hello", true},
		{"HeLLo", true},
		{"123", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := hasLowercase(tt.input); got != tt.want {
				t.Errorf("hasLowercase(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestHasNumber(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"Hello123", true},
		{"Hello", false},
		{"12345", true},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := hasNumber(tt.input); got != tt.want {
				t.Errorf("hasNumber(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestHasSpecialChar(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"Hello!", true},
		{"Hello@World", true},
		{"Hello#123", true},
		{"HelloWorld", false},
		{"", false},
		{"Hello$", true},
		{"Pass&word", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := hasSpecialChar(tt.input); got != tt.want {
				t.Errorf("hasSpecialChar(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsCommonPassword(t *testing.T) {
	tests := []struct {
		password string
		want     bool
	}{
		{"password", true},
		{"Password", true},
		{"PASSWORD", true},
		{"12345678", true},
		{"qwerty", true},
		{"MySecurePassword123!", false},
		{"admin", true},
		{"letmein", true},
	}

	for _, tt := range tests {
		t.Run(tt.password, func(t *testing.T) {
			if got := isCommonPassword(tt.password); got != tt.want {
				t.Errorf("isCommonPassword(%q) = %v, want %v", tt.password, got, tt.want)
			}
		})
	}
}
