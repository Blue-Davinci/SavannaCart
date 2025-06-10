package data

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"time"

	"github.com/Blue-Davinci/SavannaCart/internal/database"
	"golang.org/x/crypto/bcrypt"
)

// constants for general module usage
const (
	DefaultUserDBContextTimeout = 5 * time.Second
	// DefaulRedistUserMFATTLS     = 5 * time.Minute
)

// Define a custom ErrDuplicateEmail error.
var (
	ErrDuplicateEmail = errors.New("duplicate email")
	ErrEditConflict   = errors.New("edit conflict")
)

/*
	constants for tags to be used during REDIS operations

const (

	RedisMFASetupPendingPrefix         = "mfa_setup_pending"
	RedisMFALoginPendingPrefix         = "mfa_login_pending"
	RedisMFAResetPasswordPendingPrefix = "mfa_reset_password_pending"

)
*/
type UserModel struct {
	DB *database.Queries
}

type User struct {
	ID               int64     `json:"id"`
	FirstName        string    `json:"first_name"`
	LastName         string    `json:"last_name"`
	Email            string    `json:"email"`
	ProfileAvatarURL string    `json:"profile_avatar_url"`
	Password         password  `json:"-"`
	OIDCSubject      string    `json:"-"`
	RoleLevel        string    `json:"role_level"`
	Activated        bool      `json:"activated"`
	Version          int32     `json:"version"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	LastLogin        time.Time `json:"last_login"`
}

// Create a custom password type which is a struct containing the plaintext and hashed
// versions of the password for a user.
type password struct {
	plaintext *string
	hash      []byte
}

// set() calculates the bcrypt hash of a plaintext password, and stores both
// the hash and the plaintext versions in the struct.
func (p *password) Set(plaintextPassword string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(plaintextPassword), 12)
	if err != nil {
		return err
	}
	p.plaintext = &plaintextPassword
	p.hash = hash
	return nil
}

// The Matches() method checks whether the provided plaintext password matches the
// hashed password stored in the struct, returning true if it matches and false
// otherwise.
func (p *password) Matches(plaintextPassword string) (bool, error) {
	err := bcrypt.CompareHashAndPassword(p.hash, []byte(plaintextPassword))
	if err != nil {
		//fmt.Printf(">>>>> Plain text: %s\nHash: %v\n", plaintextPassword, p.hash)
		switch {
		case errors.Is(err, bcrypt.ErrMismatchedHashAndPassword):
			return false, nil
		default:
			return false, err
		}
	}
	return true, nil
}

// Declare a new AnonymousUser variable.
var AnonymousUser = &User{}

// Check if a User instance is the AnonymousUser.
func (u *User) IsAnonymous() bool {
	return u == AnonymousUser
}

func (m UserModel) CreateNewUser(User *User) error {
	ctx, cancel := contextGenerator(context.Background(), DefaultUserDBContextTimeout)
	defer cancel()

	createdUser, err := m.DB.CreateNewUser(ctx, database.CreateNewUserParams{
		FirstName:        User.FirstName,
		LastName:         User.LastName,
		Email:            User.Email,
		ProfileAvatarUrl: User.ProfileAvatarURL,
		Password:         User.Password.hash,
		OidcSub:          User.OIDCSubject, // prolly encrypt this for security
		RoleLevel:        User.RoleLevel,
		Activated:        User.Activated,
	})
	if err != nil {
		// check if user already exists
		switch {
		case err.Error() == `pq: duplicate key value violates unique constraint "users_email_key"`:
			return ErrDuplicateEmail
		default:
			return err
		}
	}
	// update the User struct with the created user data
	User.ID = createdUser.ID
	User.Version = createdUser.Version
	User.CreatedAt = createdUser.CreatedAt
	User.UpdatedAt = createdUser.UpdatedAt
	User.LastLogin = createdUser.LastLogin

	// return nil if no error
	return nil
}

// GetForToken() retrieves the details of a user based on a token, scope, and encryption key.
func (m UserModel) GetForToken(tokenScope, tokenPlaintext string) (*User, error) {
	// Calculate sha256 hash of plaintext
	tokenHash := sha256.Sum256([]byte(tokenPlaintext))
	ctx, cancel := contextGenerator(context.Background(), DefaultUserDBContextTimeout)
	defer cancel()
	// get the user
	user, err := m.DB.GetForToken(ctx, database.GetForTokenParams{
		Hash:   tokenHash[:],
		Scope:  tokenScope,
		Expiry: time.Now(),
	})
	// check for any error
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrGeneralRecordNotFound
		default:
			return nil, err
		}
	}
	// make a user
	tokenuser := populateUser(user)
	// fill in the user data
	return tokenuser, nil
}

// populateUser() is a helper function that takes a database.User struct and returns a
// pointer to a User struct containing the same data. It also decrypts the phone number
// using the provided encryption key.
// populateUser takes in a SQLC-generated row (userRow) and a decrypted phone number,
// and populates a User struct based on the type of userRow.
// The function currently supports two types: database.GetForTokenRow and database.User.
// If a new row type is introduced, this function can be extended to handle it.

func populateUser(userRow interface{}) *User {
	switch user := userRow.(type) {
	// Case for database.GetForTokenRow: Populates a User object with fields specific to the GetForTokenRow
	// Case for database.User: Populates a User object with fields specific to the User type.
	case database.User:
		// Create a new password struct instance for the user.
		userPassword := password{
			hash: user.Password,
		}
		return &User{
			ID:               user.ID,
			FirstName:        user.FirstName,
			LastName:         user.LastName,
			Email:            user.Email,
			ProfileAvatarURL: user.ProfileAvatarUrl,
			Password:         userPassword,
			OIDCSubject:      user.OidcSub,
			RoleLevel:        user.RoleLevel,
			Activated:        user.Activated,
			Version:          user.Version,
			CreatedAt:        user.CreatedAt,
			UpdatedAt:        user.UpdatedAt,
			LastLogin:        user.LastLogin,
		}

	// Default case: Returns nil if the input type does not match any supported types.
	default:
		return nil
	}
}
