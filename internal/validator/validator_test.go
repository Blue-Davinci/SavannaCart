package validator

import (
	"regexp"
	"testing"
)

func TestNew(t *testing.T) {
	v := New()

	if v == nil {
		t.Fatal("Expected non-nil validator")
	}

	if v.Errors == nil {
		t.Fatal("Expected non-nil errors map")
	}

	if len(v.Errors) != 0 {
		t.Errorf("Expected empty errors map, got %d errors", len(v.Errors))
	}
}

func TestValidator_Valid(t *testing.T) {
	tests := []struct {
		name     string
		errors   map[string]string
		expected bool
	}{
		{
			name:     "empty errors map should be valid",
			errors:   make(map[string]string),
			expected: true,
		},
		{
			name:     "nil errors map should be valid",
			errors:   nil,
			expected: true,
		},
		{
			name: "errors map with entries should be invalid",
			errors: map[string]string{
				"field": "error message",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &Validator{Errors: tt.errors}
			if got := v.Valid(); got != tt.expected {
				t.Errorf("Valid() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestValidator_AddError(t *testing.T) {
	v := New()

	// Test adding first error
	v.AddError("field1", "error message 1")
	if len(v.Errors) != 1 {
		t.Errorf("Expected 1 error, got %d", len(v.Errors))
	}
	if v.Errors["field1"] != "error message 1" {
		t.Errorf("Expected 'error message 1', got '%s'", v.Errors["field1"])
	}

	// Test adding second error
	v.AddError("field2", "error message 2")
	if len(v.Errors) != 2 {
		t.Errorf("Expected 2 errors, got %d", len(v.Errors))
	}

	// Test that duplicate key doesn't overwrite
	v.AddError("field1", "new error message")
	if v.Errors["field1"] != "error message 1" {
		t.Errorf("Expected original message, got '%s'", v.Errors["field1"])
	}
}

func TestValidator_Check(t *testing.T) {
	tests := []struct {
		name        string
		ok          bool
		key         string
		message     string
		expectError bool
	}{
		{
			name:        "valid check should not add error",
			ok:          true,
			key:         "field",
			message:     "error message",
			expectError: false,
		},
		{
			name:        "invalid check should add error",
			ok:          false,
			key:         "field",
			message:     "error message",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := New()
			v.Check(tt.ok, tt.key, tt.message)

			hasError := len(v.Errors) > 0
			if hasError != tt.expectError {
				t.Errorf("Expected error: %v, got: %v", tt.expectError, hasError)
			}

			if tt.expectError && v.Errors[tt.key] != tt.message {
				t.Errorf("Expected message '%s', got '%s'", tt.message, v.Errors[tt.key])
			}
		})
	}
}

func TestPermittedValue(t *testing.T) {
	tests := []struct {
		name            string
		value           string
		permittedValues []string
		expected        bool
	}{
		{
			name:            "value in permitted list",
			value:           "apple",
			permittedValues: []string{"apple", "banana", "cherry"},
			expected:        true,
		},
		{
			name:            "value not in permitted list",
			value:           "grape",
			permittedValues: []string{"apple", "banana", "cherry"},
			expected:        false,
		},
		{
			name:            "empty permitted list",
			value:           "apple",
			permittedValues: []string{},
			expected:        false,
		},
		{
			name:            "exact match",
			value:           "test",
			permittedValues: []string{"test"},
			expected:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := PermittedValue(tt.value, tt.permittedValues...)
			if result != tt.expected {
				t.Errorf("PermittedValue() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestPermittedValueInt(t *testing.T) {
	tests := []struct {
		name            string
		value           int
		permittedValues []int
		expected        bool
	}{
		{
			name:            "integer value in permitted list",
			value:           2,
			permittedValues: []int{1, 2, 3},
			expected:        true,
		},
		{
			name:            "integer value not in permitted list",
			value:           5,
			permittedValues: []int{1, 2, 3},
			expected:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := PermittedValue(tt.value, tt.permittedValues...)
			if result != tt.expected {
				t.Errorf("PermittedValue() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestMatches(t *testing.T) {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

	tests := []struct {
		name     string
		value    string
		regex    *regexp.Regexp
		expected bool
	}{
		{
			name:     "valid email",
			value:    "test@example.com",
			regex:    emailRegex,
			expected: true,
		},
		{
			name:     "invalid email",
			value:    "invalid-email",
			regex:    emailRegex,
			expected: false,
		},
		{
			name:     "empty string",
			value:    "",
			regex:    emailRegex,
			expected: false,
		},
		{
			name:     "partial email match",
			value:    "test@",
			regex:    emailRegex,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Matches(tt.value, tt.regex)
			if result != tt.expected {
				t.Errorf("Matches() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestUnique(t *testing.T) {
	tests := []struct {
		name     string
		values   []string
		expected bool
	}{
		{
			name:     "all unique values",
			values:   []string{"a", "b", "c"},
			expected: true,
		},
		{
			name:     "duplicate values",
			values:   []string{"a", "b", "a"},
			expected: false,
		},
		{
			name:     "empty slice",
			values:   []string{},
			expected: true,
		},
		{
			name:     "single value",
			values:   []string{"a"},
			expected: true,
		},
		{
			name:     "all same values",
			values:   []string{"a", "a", "a"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Unique(tt.values)
			if result != tt.expected {
				t.Errorf("Unique() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestUniqueInt(t *testing.T) {
	tests := []struct {
		name     string
		values   []int
		expected bool
	}{
		{
			name:     "unique integers",
			values:   []int{1, 2, 3},
			expected: true,
		},
		{
			name:     "duplicate integers",
			values:   []int{1, 2, 1},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Unique(tt.values)
			if result != tt.expected {
				t.Errorf("Unique() = %v, want %v", result, tt.expected)
			}
		})
	}
}
