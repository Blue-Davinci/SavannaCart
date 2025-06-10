package main

import (
	"expvar"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/justinas/alice"
)

// routes() is a method that returns a http.Handler that contains all the routes for the application
func (app *application) routes() http.Handler {
	router := chi.NewRouter()
	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   app.config.cors.trustedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "PATCH"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"link"},
		AllowCredentials: false,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	})) // Make our categorized routes
	//Use alice to make a global middleware chain.
	globalMiddleware := alice.New(app.metrics, app.recoverPanic, app.rateLimit, app.authenticate).Then

	// dynamic protected middleware
	dynamicMiddleware := alice.New(app.requireAuthenticatedUser, app.requireActivatedUser)

	// Apply the global middleware to the router
	router.Use(globalMiddleware)

	v1Router := chi.NewRouter()

	v1Router.Mount("/", app.generalRoutes())
	v1Router.Mount("/api", app.apiKeyRoutes(&dynamicMiddleware))

	// Mount the v1Router to the main base router
	router.Mount("/v1", v1Router)
	return router
}

// generalRoutes() is a method that returns a chi.Router that contains all the general routes
func (app *application) generalRoutes() chi.Router {
	router := chi.NewRouter()

	router.Get("/debug/vars", func(w http.ResponseWriter, r *http.Request) {
		expvar.Handler().ServeHTTP(w, r)
	})
	return router
}

func (app *application) apiKeyRoutes(dynamicMiddleware *alice.Chain) chi.Router {
	apiKeyRoutes := chi.NewRouter()
	// OAuth callback endpoint - must be GET since Google redirects with GET
	apiKeyRoutes.Get("/authentication", app.createAuthenticationApiKeyHandler)
	apiKeyRoutes.Put("/activation", app.activateUserHandler)

	// lougput route only applies to people who are registered
	apiKeyRoutes.With(dynamicMiddleware.Then).Post("/logout", app.logoutUserHandler)
	return apiKeyRoutes
}
