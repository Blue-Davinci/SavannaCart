package main

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"go.uber.org/zap"
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
	v1Router := chi.NewRouter()

	v1Router.Mount("/", app.generalRoutes())

	// Mount the v1Router to the main base router
	router.Mount("/v1", v1Router)
	return router
}

// generalRoutes() is a method that returns a chi.Router that contains all the general routes
func (app *application) generalRoutes() chi.Router {
	router := chi.NewRouter()

	router.Get("/", app.welcomeHandler)

	return router
}

// welcomeHandler handles the welcome endpoint
func (app *application) welcomeHandler(w http.ResponseWriter, r *http.Request) {
	app.logger.Info("welcomeHandler called", zap.String("method", r.Method), zap.String("url", r.URL.String()))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, `{"message": "Welcome to the SavannaCart API!"}`)
}
