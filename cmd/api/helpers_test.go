package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/Blue-Davinci/SavannaCart/internal/validator"
	"github.com/go-chi/chi/v5"
)

// TestReadIDParam tests the readIDParam helper function
func TestReadIDParam(t *testing.T) {
	app := &application{}

	tests := []struct {
		name          string
		paramValue    string
		paramName     string
		expectedID    int64
		expectedError bool
	}{
		{
			name:          "valid ID",
			paramValue:    "123",
			paramName:     "id",
			expectedID:    123,
			expectedError: false,
		},
		{
			name:          "valid large ID",
			paramValue:    "9223372036854775807", // max int64
			paramName:     "id",
			expectedID:    9223372036854775807,
			expectedError: false,
		},
		{
			name:          "invalid ID - zero",
			paramValue:    "0",
			paramName:     "id",
			expectedID:    0,
			expectedError: true,
		},
		{
			name:          "invalid ID - negative",
			paramValue:    "-1",
			paramName:     "id",
			expectedID:    0,
			expectedError: true,
		},
		{
			name:          "invalid ID - not a number",
			paramValue:    "abc",
			paramName:     "id",
			expectedID:    0,
			expectedError: true,
		},
		{
			name:          "invalid ID - empty",
			paramValue:    "",
			paramName:     "id",
			expectedID:    0,
			expectedError: true,
		},
		{
			name:          "valid ID with different param name",
			paramValue:    "456",
			paramName:     "product_id",
			expectedID:    456,
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new chi router and route
			r := chi.NewRouter()
			r.Get("/{"+tt.paramName+"}", func(w http.ResponseWriter, r *http.Request) {
				id, err := app.readIDParam(r, tt.paramName)

				if tt.expectedError {
					if err == nil {
						t.Errorf("Expected error but got none. ID: %d", id)
					}
				} else {
					if err != nil {
						t.Errorf("Expected no error but got: %v", err)
					}
					if id != tt.expectedID {
						t.Errorf("Expected ID %d, got %d", tt.expectedID, id)
					}
				}
			})

			// Create request with the param value
			req := httptest.NewRequest("GET", "/"+tt.paramValue, nil)
			w := httptest.NewRecorder()

			// Serve the request
			r.ServeHTTP(w, req)
		})
	}
}

// TestReadString tests the readString helper function
func TestReadString(t *testing.T) {
	app := &application{}

	tests := []struct {
		name         string
		queryParams  string
		key          string
		defaultValue string
		expected     string
	}{
		{
			name:         "key exists",
			queryParams:  "name=john&age=25",
			key:          "name",
			defaultValue: "default",
			expected:     "john",
		},
		{
			name:         "key does not exist",
			queryParams:  "name=john&age=25",
			key:          "email",
			defaultValue: "default@test.com",
			expected:     "default@test.com",
		},
		{
			name:         "empty query string",
			queryParams:  "",
			key:          "name",
			defaultValue: "default",
			expected:     "default",
		},
		{
			name:         "empty value",
			queryParams:  "name=&age=25",
			key:          "name",
			defaultValue: "default",
			expected:     "default",
		},
		{
			name:         "special characters",
			queryParams:  "search=hello%20world&category=test",
			key:          "search",
			defaultValue: "default",
			expected:     "hello world",
		},
		{
			name:         "multiple values for same key",
			queryParams:  "tag=go&tag=testing&tag=api",
			key:          "tag",
			defaultValue: "default",
			expected:     "go", // url.Values.Get() returns first value
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse query string
			qs, err := url.ParseQuery(tt.queryParams)
			if err != nil {
				t.Fatalf("Failed to parse query string: %v", err)
			}

			// Test the function
			result := app.readString(qs, tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestReadInt tests the readInt helper function
func TestReadInt(t *testing.T) {
	app := &application{}

	tests := []struct {
		name         string
		queryParams  string
		key          string
		defaultValue int
		expected     int
		expectError  bool
	}{
		{
			name:         "valid integer",
			queryParams:  "page=5&limit=10",
			key:          "page",
			defaultValue: 1,
			expected:     5,
			expectError:  false,
		},
		{
			name:         "key does not exist",
			queryParams:  "page=5&limit=10",
			key:          "offset",
			defaultValue: 0,
			expected:     0,
			expectError:  false,
		},
		{
			name:         "empty value",
			queryParams:  "page=&limit=10",
			key:          "page",
			defaultValue: 1,
			expected:     1,
			expectError:  false,
		},
		{
			name:         "invalid integer",
			queryParams:  "page=abc&limit=10",
			key:          "page",
			defaultValue: 1,
			expected:     1,
			expectError:  true,
		},
		{
			name:         "negative integer",
			queryParams:  "page=-5&limit=10",
			key:          "page",
			defaultValue: 1,
			expected:     -5,
			expectError:  false,
		},
		{
			name:         "zero value",
			queryParams:  "page=0&limit=10",
			key:          "page",
			defaultValue: 1,
			expected:     0,
			expectError:  false,
		},
		{
			name:         "large integer",
			queryParams:  "page=2147483647&limit=10", // max int32
			key:          "page",
			defaultValue: 1,
			expected:     2147483647,
			expectError:  false,
		},
		{
			name:         "float value",
			queryParams:  "page=5.5&limit=10",
			key:          "page",
			defaultValue: 1,
			expected:     1,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse query string
			qs, err := url.ParseQuery(tt.queryParams)
			if err != nil {
				t.Fatalf("Failed to parse query string: %v", err)
			}

			// Create validator
			v := validator.New()

			// Test the function
			result := app.readInt(qs, tt.key, tt.defaultValue, v)

			// Check result
			if result != tt.expected {
				t.Errorf("Expected %d, got %d", tt.expected, result)
			}

			// Check error expectation
			hasError := !v.Valid()
			if hasError != tt.expectError {
				if tt.expectError {
					t.Errorf("Expected validation error but got none")
				} else {
					t.Errorf("Expected no validation error but got: %v", v.Errors)
				}
			}
		})
	}
}

// TestValidateURL tests the validateURL helper function
func TestValidateURL(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
	}{
		{
			name:        "valid HTTP URL",
			input:       "http://example.com",
			expectError: false,
		},
		{
			name:        "valid HTTPS URL",
			input:       "https://example.com",
			expectError: false,
		},
		{
			name:        "valid URL with path",
			input:       "https://example.com/path/to/resource",
			expectError: false,
		},
		{
			name:        "valid URL with query params",
			input:       "https://example.com/search?q=test&category=api",
			expectError: false,
		},
		{
			name:        "valid URL with port",
			input:       "http://localhost:8080",
			expectError: false,
		},
		{
			name:        "invalid URL - no scheme",
			input:       "example.com",
			expectError: true,
		},
		{
			name:        "invalid URL - empty string",
			input:       "",
			expectError: true,
		},
		{
			name:        "invalid URL - malformed",
			input:       "ht!tp://example.com",
			expectError: true,
		},
		{
			name:        "invalid URL - spaces",
			input:       "http://example .com",
			expectError: true,
		},
		{
			name:        "valid FTP URL",
			input:       "ftp://files.example.com",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateURL(tt.input)

			if tt.expectError && err == nil {
				t.Errorf("Expected error for URL %q but got none", tt.input)
			}

			if !tt.expectError && err != nil {
				t.Errorf("Expected no error for URL %q but got: %v", tt.input, err)
			}
		})
	}
}
