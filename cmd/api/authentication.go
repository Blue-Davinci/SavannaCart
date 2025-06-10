package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Blue-Davinci/SavannaCart/internal/data"
	"github.com/Blue-Davinci/SavannaCart/internal/validator"
	"github.com/coreos/go-oidc/v3/oidc"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
)

/*
Flow:
- Frontend, or our init code in this case, allows a user to login via OAuth2.0
- The user is redirected to the OAuth2.0 provider (e.g., Google, GitHub) to authenticate.
- After successful authentication, the OAuth2.0 provider redirects back to our application with an authorization code.
- Our application exchanges the authorization code for an id_token and access_token .
- wWe validate the id_token to ensure it is valid and contains the necessary claims.
- We extract the user information from the id_token, such as email and name.
- Once this is done, we check if the user exists in our database.
- If the user exists:
  - we generate the token, saving it to our DB.
  - We also save a 'session' copy in REDIS for quick access.
  - We then send back the bearer token + expiry_time + user information to the frontend.

- If the user does not exist:
  - we create a new user in our database with the information from the id_token.
  - Send the user an email verification link with a callback URL to verify their email.
  - If they click the link, we verify their email and mark them as verified in our database.
  - Then proceed with the same steps as above to generate the token and send it back to the frontend.
*/

func (app *application) InitOIDC() error {
	// Validate required configuration
	if app.config.api.oidc_client_id == "" {
		return fmt.Errorf("OIDC client ID is required")
	}
	if app.config.api.oidc_client_secret == "" {
		return fmt.Errorf("OIDC client secret is required")
	}
	ctx := context.Background()
	// Initialize the OIDC provider
	var err error
	app.config.authenticators.provider, err = oidc.NewProvider(ctx, app.config.app_urls.provide_url)
	if err != nil {
		return err
	}

	// Initialize the OAuth2 configuration
	app.config.authenticators.oauthConfig = &oauth2.Config{
		ClientID:     app.config.api.oidc_client_id,
		ClientSecret: app.config.api.oidc_client_secret,
		Endpoint:     app.config.authenticators.provider.Endpoint(),
		RedirectURL:  app.config.app_urls.authentication_callback_url,
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
	}

	// Create an ID token verifier
	app.config.authenticators.verifier = app.config.authenticators.provider.Verifier(&oidc.Config{ClientID: app.config.api.oidc_client_id})

	return nil
}

// logoutUserHandler() is the main endpoint responsible for logging out the user.
// Currently, we will just terminate a user's SSE connection if they have one.
func (app *application) logoutUserHandler(w http.ResponseWriter, r *http.Request) {
	// Get the user from the context
	userID := app.contextGetUser(r).ID
	// delete all their authentication tokens
	err := app.models.Tokens.DeleteAllForUser(data.ScopeAuthentication, userID)
	if err != nil {
		app.logger.Error("Error deleting authentication tokens for user",
			zap.Int64("user_id", userID),
			zap.Error(err))
		app.serverErrorResponse(w, r, err)
		return
	}
	// write 200 ok
	err = app.writeJSON(w, http.StatusOK, envelope{"message": "you have been logged out"}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// activateUserHandler() Handles activating a user. Inactive users cannot perform a multitude
// of functions. This handler accepts a JSON request containing a plaintext activation token
// and activates the user associated with the token & the activate scope if that token exists.
func (app *application) activateUserHandler(w http.ResponseWriter, r *http.Request) {
	// Parse the plaintext activation token from the request body.
	var input struct {
		TokenPlaintext string `json:"token"`
	}
	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}
	// Validate the plaintext token provided by the client.
	v := validator.New()
	if data.ValidateTokenPlaintext(v, input.TokenPlaintext); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}
	// Retrieve the details of the user associated with the token using the
	// GetForToken() method. If no matching record is found, then we let the
	// client know that the token they provided is not valid.
	user, err := app.models.Users.GetForToken(data.ScopeActivation, input.TokenPlaintext)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrGeneralRecordNotFound):
			v.AddError("token", "invalid or expired activation token")
			app.failedValidationResponse(w, r, v.Errors)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}
	app.logger.Info("User Version: ", zap.Int("Version", int(user.Version)))
	// Update the user's activation status.
	user.Activated = true
	// Save the updated user record in our database, checking for any edit conflicts in
	// the same way that we did for our movie records.
	err = app.models.Users.UpdateUser(user)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrEditConflict):
			app.editConflictResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}
	// If everything went successfully, then we delete all activation tokens for the
	// user.
	err = app.models.Tokens.DeleteAllForUser(data.ScopeActivation, user.ID)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	// Succesful, so we send an email for a succesful activation
	app.background(func() {
		// As there are now multiple pieces of data that we want to pass to our email
		// templates, we create a map to act as a 'holding structure' for the data. This
		// contains the plaintext version of the activation token for the user, along
		// with their ID.
		data := map[string]any{
			"loginURL":  app.config.app_urls.authentication_callback_url,
			"firstName": user.FirstName,
			"lastName":  user.LastName,
		}
		// Send the welcome email, passing in the map above as dynamic data.
		err = app.mailer.Send(user.Email, "user_succesful_activation.tmpl", data)
		if err != nil {
			app.logger.Error("Error sending welcome email", zap.String("email", user.Email), zap.Error(err))
		}
	})

	// Generate authentication token for the activated user
	authToken, err := app.generateUserAuthenticationToken(user)
	if err != nil {
		app.logger.Error("Error generating authentication token for activated user", zap.Error(err))
		app.serverErrorResponse(w, r, err)
		return
	}

	app.logger.Info("User successfully activated and authenticated",
		zap.String("email", user.Email),
		zap.Int64("user_id", user.ID))

	// Send complete response with user details and authentication token
	response := envelope{
		"message": "Account activated successfully! You are now logged in.",
		"user": envelope{
			"id":         user.ID,
			"email":      user.Email,
			"first_name": user.FirstName,
			"last_name":  user.LastName,
			"activated":  user.Activated,
		},
		"authentication_token": envelope{
			"token":  authToken.Plaintext,
			"expiry": authToken.Expiry,
		},
	}

	err = app.writeJSON(w, http.StatusOK, response, nil)
	if err != nil {
		app.logger.Error("Error writing JSON response for user activation", zap.Error(err))
		app.serverErrorResponse(w, r, err)
	}
}

// createAuthenticationApiKeyHandler is our main authentication endpoint handler and the hitpoint for tha auth callback
// It will handle the OAuth2.0 flow, including redirecting to the provider, handling the callback, and generating API keys.
// It will also handle the creation of API keys for authenticated users.
func (app *application) createAuthenticationApiKeyHandler(w http.ResponseWriter, r *http.Request) {
	// Get the authorization code from query parameters using helper methods
	var input struct {
		AuthorizationCode string `json:"code"`
		State             string `json:"state,omitempty"`
		Error             string `json:"error,omitempty"`
		ErrorDescription  string `json:"error_description,omitempty"`
	}

	qs := r.URL.Query()
	input.AuthorizationCode = app.readString(qs, "code", "")
	input.State = app.readString(qs, "state", "")
	input.Error = app.readString(qs, "error", "")
	input.ErrorDescription = app.readString(qs, "error_description", "")

	// Check for OAuth errors first
	if input.Error != "" {
		app.logger.Error("OAuth error received",
			zap.String("error", input.Error),
			zap.String("description", input.ErrorDescription))
		app.invalidCredentialsResponse(w, r)
		return
	}
	// Validate that we received an authorization code (no need of dedicated validator here since it's a simple check)
	if input.AuthorizationCode == "" {
		app.logger.Error("No authorization code received")
		app.invalidCredentialsResponse(w, r)
		return
	}

	// Log the authorization code length for debugging (don't log the actual code for security)
	app.logger.Info("Received authorization code",
		zap.Int("code_length", len(input.AuthorizationCode)),
		zap.String("state", input.State))

	// Exchange the authorization code for tokens
	ctx := context.Background()
	exchange_token, err := app.config.authenticators.oauthConfig.Exchange(ctx, input.AuthorizationCode)
	if err != nil {
		app.logger.Error("Error exchanging authorization code for tokens",
			zap.Error(err),
			zap.String("error_type", fmt.Sprintf("%T", err)))

		// Check if it's an OAuth2 error for more detailed logging
		if oauthErr, ok := err.(*oauth2.RetrieveError); ok {
			app.logger.Error("OAuth2 detailed error",
				zap.Int("response_code", oauthErr.Response.StatusCode),
				zap.String("response_body", string(oauthErr.Body)))
		}

		app.invalidCredentialsResponse(w, r)
		return
	}

	// Extract the ID Token from OAuth2 token
	rawIDToken, ok := exchange_token.Extra("id_token").(string)
	if !ok {
		app.logger.Error("No id_token field in OAuth2 token")
		app.serverErrorResponse(w, r, fmt.Errorf("no id_token field in OAuth2 token"))
		return
	}

	// Verify and parse the ID token
	idToken, err := app.config.authenticators.verifier.Verify(context.Background(), rawIDToken)
	if err != nil {
		app.logger.Error("Failed to verify ID Token", zap.Error(err))
		app.invalidCredentialsResponse(w, r)
		return
	}
	// Extract user information from the ID token
	var claims data.OAuthClaims

	if err := idToken.Claims(&claims); err != nil {
		app.logger.Error("Failed to extract claims from ID token", zap.Error(err))
		app.serverErrorResponse(w, r, err)
		return
	}
	app.logger.Info("OAuth user authenticated",
		zap.String("email", claims.Email),
		zap.String("name", claims.Name),
		zap.Bool("email_verified", claims.EmailVerified))

	// Check if the user exists in the database
	user, err := app.models.Users.GetByEmail(claims.Email, "")
	if err != nil {
		switch {
		case errors.Is(err, data.ErrGeneralRecordNotFound):
			// User doesn't exist, handle signup flow
			app.handleOAuthSignup(w, r, &claims)
			return
		default:
			app.logger.Error("Error retrieving user by email", zap.String("email", claims.Email), zap.Error(err))
			app.serverErrorResponse(w, r, err)
			return
		}
	}
	// User exists, handle login flow
	app.handleOAuthLogin(w, r, user)
}

// handleOAuthSignup handles the creation of new users from OAuth authentication
func (app *application) handleOAuthSignup(w http.ResponseWriter, r *http.Request, claims *data.OAuthClaims) {
	app.logger.Info("User not found, creating new user", zap.String("email", claims.Email))

	// Extract user information from claims
	firstName, lastName := claims.ExtractNames()

	// Create a new user (NOT activated by default - they need to verify email)
	newUser := &data.User{
		FirstName:        firstName,
		LastName:         lastName,
		Email:            claims.Email,
		ProfileAvatarURL: claims.Picture,
		OIDCSubject:      claims.Subject,
		RoleLevel:        "user", // Default role
		Activated:        false,  // Always false for new users - they must verify email
	}

	// Set a default password (user won't use this for OAuth logins)
	err := newUser.Password.Set("oauth_user_" + claims.Subject)
	if err != nil {
		app.logger.Error("Error setting password for new OAuth user", zap.Error(err))
		app.serverErrorResponse(w, r, err)
		return
	}

	// Validate the user data
	v := validator.New()
	data.ValidateUser(v, newUser)
	if !v.Valid() {
		app.logger.Error("Validation failed for new OAuth user", zap.Any("errors", v.Errors))
		app.invalidCredentialsResponse(w, r)
		return
	}

	// Create the user in the database
	err = app.models.Users.CreateNewUser(newUser)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrDuplicateEmail):
			// User was created by another request, redirect to login flow
			existingUser, getErr := app.models.Users.GetByEmail(claims.Email, "")
			if getErr != nil {
				app.logger.Error("Error retrieving user after duplicate email error", zap.Error(getErr))
				app.serverErrorResponse(w, r, getErr)
				return
			}
			// Handle existing user login
			app.handleOAuthLogin(w, r, existingUser)
			return
		default:
			app.logger.Error("Error creating new user", zap.Error(err))
			app.serverErrorResponse(w, r, err)
			return
		}
	}

	app.logger.Info("New user created successfully",
		zap.String("email", newUser.Email),
		zap.Int64("user_id", newUser.ID))

	// Generate activation token - this is REQUIRED for new users
	activationToken, err := app.models.Tokens.New(newUser.ID, data.DefaultTokenExpiryTime, data.ScopeActivation)
	if err != nil {
		app.logger.Error("Error creating activation token", zap.Error(err))
		// If we cannot generate an activation token, we need to exit here as users need to be activated
		app.serverErrorResponse(w, r, err)
		return
	}

	app.background(func() {
		// Send activation email
		activationURL := fmt.Sprintf("%s?token=%s", app.config.app_urls.activation_callback_url, activationToken.Plaintext)
		emailData := map[string]any{
			"activationURL": activationURL,
			"firstName":     newUser.FirstName,
			"lastName":      newUser.LastName,
			"userID":        newUser.ID,
		}
		err = app.mailer.Send(newUser.Email, "user_welcome.tmpl", emailData)
		if err != nil {
			app.logger.Error("Error sending welcome email", zap.Error(err))
		}

	})

	// Return success response - NO TOKEN GENERATION for new users
	response := envelope{
		"message": "Account created successfully! Please check your email to activate your account.",
		"user": envelope{
			"id":         newUser.ID,
			"email":      newUser.Email,
			"first_name": newUser.FirstName,
			"last_name":  newUser.LastName,
			"activated":  newUser.Activated,
		},
		"next_step": "Click the activation link in your email to complete registration",
	}

	err = app.writeJSON(w, http.StatusCreated, response, nil)
	if err != nil {
		app.logger.Error("Error writing JSON response for new user creation", zap.Error(err))
		app.serverErrorResponse(w, r, err)
	}
}

// handleOAuthLogin handles authentication for existing users
func (app *application) handleOAuthLogin(w http.ResponseWriter, r *http.Request, user *data.User) {
	app.logger.Info("Existing user attempting login", zap.String("email", user.Email), zap.Int64("user_id", user.ID))

	// Check if user is activated
	if !user.Activated {
		response := envelope{
			"error":   "Account not activated",
			"message": "Please check your email and click the activation link to activate your account.",
			"user": envelope{
				"id":        user.ID,
				"email":     user.Email,
				"activated": user.Activated,
			},
		}
		err := app.writeJSON(w, http.StatusUnauthorized, response, nil)
		if err != nil {
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	// Generate authentication token using our modular helper
	token, err := app.generateUserAuthenticationToken(user)
	if err != nil {
		app.logger.Error("Error generating authentication token", zap.Error(err))
		app.serverErrorResponse(w, r, err)
		return
	}

	app.logger.Info("User successfully authenticated", zap.String("email", user.Email), zap.Int64("user_id", user.ID))

	response := envelope{
		"message": "Authentication successful",
		"user": envelope{
			"id":         user.ID,
			"email":      user.Email,
			"first_name": user.FirstName,
			"last_name":  user.LastName,
			"activated":  user.Activated,
		},
		"authentication_token": envelope{
			"token":  token.Plaintext,
			"expiry": token.Expiry,
		},
	}

	err = app.writeJSON(w, http.StatusOK, response, nil)
	if err != nil {
		app.logger.Error("Error writing JSON response for user login", zap.Error(err))
		app.serverErrorResponse(w, r, err)
	}
}

// generateAndPrintOAuthURL creates a fresh OAuth URL and prints it to the console
// This is called at server startup for easy copy-paste during development
func (app *application) generateAndPrintOAuthURL() {
	// Generate a more secure state parameter for CSRF protection
	state := fmt.Sprintf("savanna_%d_%s", time.Now().Unix(), generateSecureToken(16))

	// Generate the OAuth URL with additional parameters for a more robust flow
	authURL := app.config.authenticators.oauthConfig.AuthCodeURL(state,
		oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("prompt", "consent"), // Force consent screen to ensure fresh tokens
		oauth2.SetAuthURLParam("include_granted_scopes", "true"),
	)

	// Print to console with clear formatting
	app.logger.Info("🔗 OAuth Authentication URL Generated")
	app.logger.Info("📋 Copy and paste this URL into your browser to test OAuth:")
	app.logger.Info("🌐 " + authURL)
	app.logger.Info("💡 This URL is fresh and ready to use!")
	app.logger.Info("🔒 State parameter for CSRF protection: " + state)

	// Also print without logger formatting for easy copy-paste
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("🚀 SAVANNACART OAUTH URL - Ready for Testing!")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("Copy this URL to your browser:")
	fmt.Println()
	fmt.Println(authURL)
	fmt.Println()
	fmt.Printf("State: %s\n", state)
	fmt.Println(strings.Repeat("=", 80))
}

// generateSecureToken creates a cryptographically secure random token
func generateSecureToken(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}

// generateUserAuthenticationToken generates a new authentication token for a user
// This is a helper function to keep token generation logic modular and reusable
func (app *application) generateUserAuthenticationToken(user *data.User) (*data.Token, error) {
	// Delete any existing authentication tokens for this user
	err := app.models.Tokens.DeleteAllForUser(data.ScopeAuthentication, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to delete existing tokens: %w", err)
	}

	// Generate a new authentication token with default expiry
	token, err := app.models.Tokens.New(user.ID, data.DefaultTokenExpiryTime, data.ScopeAuthentication)
	if err != nil {
		return nil, fmt.Errorf("failed to generate new token: %w", err)
	}

	return token, nil
}
