package data

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"time"

	"github.com/Blue-Davinci/SavannaCart/internal/database"
	"github.com/Blue-Davinci/SavannaCart/internal/validator"
)

var (
	ErrDuplicatePermission = errors.New("duplicate permission")
	ErrPermissionNotFound  = errors.New("permission not found")
)
var (
	PermissionAdminWrite = "admin:write"
	PermissionAdminRead  = "admin:read"
)

// Define the PermissionModel type.
type PermissionModel struct {
	DB *database.Queries
}

type UserPermission struct {
	PermissionID int64    `json:"permission_id"`
	UserID       int64    `json:"user_id"`
	Permissions  []string `json:"permissions"`
}

type SuperUsersWithPermissions struct {
	UserID        int64  `json:"user_id"`
	UserFirstName string `json:"user_first_name"`
	UserLastName  string `json:"user_last_name"`
	UserEmail     string `json:"user_email"`
}

func ValidatePermissionsAddition(v *validator.Validator, permissions *UserPermission) {
	v.Check(len(permissions.Permissions) != 0, "permissions", "must be provided")
	//v.Check()
	v.Check(permissions.UserID > 0, "user_id", "must be provided")
}
func ValidatePermissionsDeletion(v *validator.Validator, userID int64, permissionCode string) {
	v.Check(permissionCode != "", "codes", "must be provided")
	//v.Check()
	v.Check(userID > 0, "user_id", "must be provided")
}

func ValidatePermission(v *validator.Validator, permissionCode string) {
	v.Check(permissionCode != "", "codes", "must be provided")
	//check permission validity only if code is not empty
	if permissionCode != "" {
		v.Check(IsValidPermissionFormat(permissionCode), "permissions", "must be in the format 'permission:code'")
	}
}

// Function to check if a permission matches the format "permission:code"
func IsValidPermissionFormat(permission string) bool {
	// Compile the regular expression to allow letters, numbers, underscores, and hyphens
	re := regexp.MustCompile(`^[a-zA-Z0-9_-]+:[a-zA-Z0-9_-]+$`)
	// Check if the permission matches the format
	return re.MatchString(permission)
}

// Make a slice to hold the the permission codes (like
// "admin:read" and "admin:write") for an admin user.
type Permissions []string

// Add a helper method to check whether the Permissions slice contains a specific
// permission code.
func (p Permissions) Include(code string) bool {
	for i := range p {
		if code == p[i] {
			return true
		}
	}
	return false
}

// GetAllSuperUsersWithPermissions() is a method that retrieves all super users with their permissions
// from the database. It returns a slice of UserPermission pointers and an error if any occurs.
func (m PermissionModel) GetAllSuperUsersWithPermissions() ([]*SuperUsersWithPermissions, error) {
	// set up context
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	// create our super users with permissions
	var superUsersWithPermissions []*SuperUsersWithPermissions
	// call the database method
	dbSuperUsers, err := m.DB.GetAllSuperUsersWithPermissions(ctx)
	if err != nil {
		return nil, err
	}
	for _, user := range dbSuperUsers {
		superUsersWithPermissions = append(superUsersWithPermissions, &SuperUsersWithPermissions{
			UserID:        user.UserID,
			UserFirstName: user.FirstName,
			UserLastName:  user.LastName,
			UserEmail:     user.Email,
		})
	}
	return superUsersWithPermissions, nil
}

// GetAllPermissions() just returns all available permissions currently in the system.
func (m PermissionModel) GetAllPermissions() ([]*UserPermission, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	permissions, err := m.DB.GetAllPermissions(ctx)
	if err != nil {
		return nil, err
	}
	var allPermissions []*UserPermission
	for _, permission := range permissions {
		allPermissions = append(allPermissions, &UserPermission{
			PermissionID: permission.ID,
			Permissions:  []string{permission.Code},
		})
	}

	return allPermissions, nil
}

// GetAllPermissionsForUser() is a method that retrieves all permissions for a specific user
// from the database. It expects the user's ID as input and returns a slice of permission codes.
func (m PermissionModel) GetAllPermissionsForUser(userID int64) (Permissions, error) {
	// set up context
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	// create our permissions
	var permissions Permissions
	// call the database method
	dbPermissions, err := m.DB.GetAllPermissionsForUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	for _, permission := range dbPermissions {
		permissions = append(permissions, permission)
	}
	// return permissions
	return permissions, nil
}

// AddPermissionsForUser() is an admin method that adds permissions for a specific user
// in the database. It expects the user's ID and a slice of permission codes as input.
func (m PermissionModel) AddPermissionsForUser(userID int64, codes ...string) (*UserPermission, error) {
	// setup our context timeout
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	// insert our permissions
	queryResult, err := m.DB.AddPermissionsForUser(ctx, database.AddPermissionsForUserParams{
		UserID:  userID,
		Column2: codes,
	})
	if err != nil {
		switch {
		case err.Error() == `pq: duplicate key value violates unique constraint "users_permissions_pkey"`:
			return nil, ErrDuplicatePermission
		default:
			return nil, err
		}
	}
	// create our permissions
	userPermission := &UserPermission{
		PermissionID: queryResult.PermissionID,
		UserID:       queryResult.UserID,
		Permissions:  codes,
	}
	// return the permissions
	return userPermission, nil
}

// DeletePermissionsForUser() is an admin method that deletes permissions for a specific user
// in the database. It expects the user's ID and a permission code as input.
func (m PermissionModel) DeletePermissionsForUser(userID int64, permissionCode string) (int64, error) {
	// Setup our context timeout
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Execute the deletion query
	permissionID, err := m.DB.DeletePermissionsForUser(ctx, database.DeletePermissionsForUserParams{
		UserID: userID,
		Code:   permissionCode,
	})
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return 0, ErrPermissionNotFound
		default:
			return 0, err
		}
	}

	return permissionID, nil
}
