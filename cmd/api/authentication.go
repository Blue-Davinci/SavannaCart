package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Blue-Davinci/SavannaCart/internal/data"
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
	} // Exchange the authorization code for tokens
	exchange_token, err := app.config.authenticators.oauthConfig.Exchange(context.Background(), input.AuthorizationCode)
	if err != nil {
		app.logger.Error("Error exchanging authorization code for tokens", zap.Error(err))
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
	var claims struct {
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		Name          string `json:"name"`
		GivenName     string `json:"given_name"`
		FamilyName    string `json:"family_name"`
		Picture       string `json:"picture"`
		Subject       string `json:"sub"`
	}

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
		// User doesn't exist, create a new one
		// TODO: Implement user creation and email verification
		app.logger.Info("User not found, would create new user", zap.String("email", claims.Email))

		// For now, return success with user info
		response := envelope{
			"message": "New user detected - account creation flow needed",
			"user_info": envelope{
				"email":    claims.Email,
				"name":     claims.Name,
				"picture":  claims.Picture,
				"verified": claims.EmailVerified,
			},
		}

		err = app.writeJSON(w, http.StatusOK, response, nil)
		if err != nil {
			app.serverErrorResponse(w, r, err)
		}
		return
	}
	// User exists, generate an API key
	// Delete any existing authentication tokens for this user
	err = app.models.Tokens.DeleteAllForUser(data.ScopeAuthentication, user.ID)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// Generate a new authentication token with 72 hour expiry
	token, err := app.models.Tokens.New(user.ID, data.DefaultTokenExpiryTime, data.ScopeAuthentication)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	app.logger.Info("Existing user authenticated", zap.String("email", user.Email), zap.Int64("user_id", user.ID))

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
		app.serverErrorResponse(w, r, err)
		return
	}
}
