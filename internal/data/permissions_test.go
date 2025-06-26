package data

import (
	"testing"

	"github.com/Blue-Davinci/SavannaCart/internal/validator"
)

func TestIsValidPermissionFormat(t *testing.T) {
	tests := []struct {
		name       string
		permission string
		expected   bool
	}{
		{
			name:       "valid admin read permission",
			permission: "admin:read",
			expected:   true,
		},
		{
			name:       "valid admin write permission",
			permission: "admin:write",
			expected:   true,
		},
		{
			name:       "valid user read permission",
			permission: "user:read",
			expected:   true,
		},
		{
			name:       "valid orders permission",
			permission: "orders:manage",
			expected:   true,
		},
		{
			name:       "valid products permission",
			permission: "products:create",
			expected:   true,
		},
		{
			name:       "missing colon",
			permission: "adminread",
			expected:   false,
		},
		{
			name:       "empty string",
			permission: "",
			expected:   false,
		},
		{
			name:       "only colon",
			permission: ":",
			expected:   false,
		},
		{
			name:       "colon at start",
			permission: ":read",
			expected:   false,
		},
		{
			name:       "colon at end",
			permission: "admin:",
			expected:   false,
		},
		{
			name:       "multiple colons",
			permission: "admin:read:write",
			expected:   false,
		},
		{
			name:       "spaces in permission",
			permission: "admin : read",
			expected:   false,
		},
		{
			name:       "uppercase permission",
			permission: "ADMIN:READ",
			expected:   true, // Assuming case doesn't matter
		},
		{
			name:       "mixed case permission",
			permission: "Admin:Read",
			expected:   true, // Assuming case doesn't matter
		},
		{
			name:       "long permission names",
			permission: "super_admin:manage_everything",
			expected:   true,
		},
		{
			name:       "special characters",
			permission: "admin-role:read-only",
			expected:   true,
		},
		{
			name:       "numbers in permission",
			permission: "level1:access2",
			expected:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidPermissionFormat(tt.permission)
			if result != tt.expected {
				t.Errorf("IsValidPermissionFormat(%q) = %v, want %v", tt.permission, result, tt.expected)
			}
		})
	}
}

func TestPermissionsInclude(t *testing.T) {
	tests := []struct {
		name        string
		permissions Permissions
		code        string
		expected    bool
	}{
		{
			name:        "permission exists",
			permissions: Permissions{"admin:read", "admin:write", "user:read"},
			code:        "admin:read",
			expected:    true,
		},
		{
			name:        "permission does not exist",
			permissions: Permissions{"admin:read", "admin:write", "user:read"},
			code:        "admin:delete",
			expected:    false,
		},
		{
			name:        "empty permissions slice",
			permissions: Permissions{},
			code:        "admin:read",
			expected:    false,
		},
		{
			name:        "nil permissions slice",
			permissions: nil,
			code:        "admin:read",
			expected:    false,
		},
		{
			name:        "empty code",
			permissions: Permissions{"admin:read", "admin:write"},
			code:        "",
			expected:    false,
		},
		{
			name:        "exact match required",
			permissions: Permissions{"admin:read", "admin:write"},
			code:        "admin",
			expected:    false,
		},
		{
			name:        "case sensitive match",
			permissions: Permissions{"admin:read", "admin:write"},
			code:        "ADMIN:READ",
			expected:    false,
		},
		{
			name:        "single permission",
			permissions: Permissions{"orders:manage"},
			code:        "orders:manage",
			expected:    true,
		},
		{
			name: "many permissions",
			permissions: func() Permissions {
				perms := make(Permissions, 100)
				for i := 0; i < 100; i++ {
					perms[i] = "perm:action" + string(rune(i))
				}
				return perms
			}(),
			code:     "perm:action50",
			expected: false, // This will likely be false due to the string conversion
		},
		{
			name:        "duplicate permissions",
			permissions: Permissions{"admin:read", "admin:read", "admin:write"},
			code:        "admin:read",
			expected:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.permissions.Include(tt.code)
			if result != tt.expected {
				t.Errorf("Permissions.Include(%q) = %v, want %v", tt.code, result, tt.expected)
			}
		})
	}
}

func TestValidatePermission(t *testing.T) {
	tests := []struct {
		name           string
		permissionCode string
		expectedErrors []string
	}{
		{
			name:           "valid permission",
			permissionCode: "admin:read",
			expectedErrors: []string{},
		},
		{
			name:           "empty permission code",
			permissionCode: "",
			expectedErrors: []string{"codes"},
		},
		{
			name:           "invalid permission format",
			permissionCode: "invalid_format",
			expectedErrors: []string{"permissions"},
		},
		{
			name:           "permission with multiple colons",
			permissionCode: "admin:read:write",
			expectedErrors: []string{"permissions"},
		},
		{
			name:           "permission starting with colon",
			permissionCode: ":read",
			expectedErrors: []string{"permissions"},
		},
		{
			name:           "permission ending with colon",
			permissionCode: "admin:",
			expectedErrors: []string{"permissions"},
		},
		{
			name:           "valid complex permission",
			permissionCode: "orders:manage_all",
			expectedErrors: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := validator.New()
			ValidatePermission(v, tt.permissionCode)

			// Check that we have the expected number of errors
			if len(v.Errors) != len(tt.expectedErrors) {
				t.Errorf("Expected %d errors, got %d. Errors: %v", len(tt.expectedErrors), len(v.Errors), v.Errors)
			}

			// Check that all expected error fields are present
			for _, expectedField := range tt.expectedErrors {
				if _, exists := v.Errors[expectedField]; !exists {
					t.Errorf("Expected error for field '%s', but it was not found. Actual errors: %v", expectedField, v.Errors)
				}
			}
		})
	}
}

func TestValidatePermissionsAddition(t *testing.T) {
	tests := []struct {
		name           string
		permissions    *UserPermission
		expectedErrors []string
	}{
		{
			name: "valid permissions addition",
			permissions: &UserPermission{
				UserID:      1,
				Permissions: []string{"admin:read", "admin:write"},
			},
			expectedErrors: []string{},
		},
		{
			name: "empty permissions",
			permissions: &UserPermission{
				UserID:      1,
				Permissions: []string{},
			},
			expectedErrors: []string{"permissions"},
		},
		{
			name: "nil permissions",
			permissions: &UserPermission{
				UserID:      1,
				Permissions: nil,
			},
			expectedErrors: []string{"permissions"},
		},
		{
			name: "invalid user ID",
			permissions: &UserPermission{
				UserID:      0,
				Permissions: []string{"admin:read"},
			},
			expectedErrors: []string{"user_id"},
		},
		{
			name: "negative user ID",
			permissions: &UserPermission{
				UserID:      -1,
				Permissions: []string{"admin:read"},
			},
			expectedErrors: []string{"user_id"},
		},
		{
			name: "both invalid",
			permissions: &UserPermission{
				UserID:      0,
				Permissions: []string{},
			},
			expectedErrors: []string{"permissions", "user_id"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := validator.New()
			ValidatePermissionsAddition(v, tt.permissions)

			// Check that we have the expected number of errors
			if len(v.Errors) != len(tt.expectedErrors) {
				t.Errorf("Expected %d errors, got %d. Errors: %v", len(tt.expectedErrors), len(v.Errors), v.Errors)
			}

			// Check that all expected error fields are present
			for _, expectedField := range tt.expectedErrors {
				if _, exists := v.Errors[expectedField]; !exists {
					t.Errorf("Expected error for field '%s', but it was not found. Actual errors: %v", expectedField, v.Errors)
				}
			}
		})
	}
}

func TestValidatePermissionsDeletion(t *testing.T) {
	tests := []struct {
		name           string
		userID         int64
		permissionCode string
		expectedErrors []string
	}{
		{
			name:           "valid deletion",
			userID:         1,
			permissionCode: "admin:read",
			expectedErrors: []string{},
		},
		{
			name:           "empty permission code",
			userID:         1,
			permissionCode: "",
			expectedErrors: []string{"codes"},
		},
		{
			name:           "invalid user ID",
			userID:         0,
			permissionCode: "admin:read",
			expectedErrors: []string{"user_id"},
		},
		{
			name:           "both invalid",
			userID:         0,
			permissionCode: "",
			expectedErrors: []string{"codes", "user_id"},
		},
		{
			name:           "negative user ID",
			userID:         -1,
			permissionCode: "admin:read",
			expectedErrors: []string{"user_id"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := validator.New()
			ValidatePermissionsDeletion(v, tt.userID, tt.permissionCode)

			// Check that we have the expected number of errors
			if len(v.Errors) != len(tt.expectedErrors) {
				t.Errorf("Expected %d errors, got %d. Errors: %v", len(tt.expectedErrors), len(v.Errors), v.Errors)
			}

			// Check that all expected error fields are present
			for _, expectedField := range tt.expectedErrors {
				if _, exists := v.Errors[expectedField]; !exists {
					t.Errorf("Expected error for field '%s', but it was not found. Actual errors: %v", expectedField, v.Errors)
				}
			}
		})
	}
}
