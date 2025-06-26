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
	// Permission Middleware, this will apply to specific routes that are capped by the permissions
	adminPermissionMiddleware := alice.New(app.requirePermission("admin:write"))

	// Apply the global middleware to the router
	router.Use(globalMiddleware)

	v1Router := chi.NewRouter()

	v1Router.Mount("/", app.generalRoutes())
	v1Router.Mount("/api", app.apiKeyRoutes(&dynamicMiddleware))
	// this are hybrid routes
	v1Router.With(dynamicMiddleware.Then).Mount("/categories", app.categoryRoutes(&adminPermissionMiddleware))
	v1Router.With(dynamicMiddleware.Then).Mount("/products", app.productRoutes(&adminPermissionMiddleware))
	v1Router.With(dynamicMiddleware.Then).Mount("/orders", app.orderRoutes(&dynamicMiddleware))

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
	apiKeyRoutes.Get("/healthcheck", app.healthCheckHandler)

	// updateUserInfo
	apiKeyRoutes.With(dynamicMiddleware.Then).Patch("/user", app.updateUserInfo)
	// lougput route only applies to people who are registered
	apiKeyRoutes.With(dynamicMiddleware.Then).Post("/logout", app.logoutUserHandler)
	return apiKeyRoutes
}

// category routes
func (app *application) categoryRoutes(adminMIddleware *alice.Chain) chi.Router {
	categoryRoutes := chi.NewRouter()
	// Get all categories, open to everyone who is authenticated
	categoryRoutes.Get("/", app.getAllCategoriesHandler)

	// Get category average price, open to everyone who is authenticated
	categoryRoutes.Get("/{categoryID:[0-9]+}", app.getCategoryAveragePriceHandler)

	// Admin only routes
	categoryRoutes.With(adminMIddleware.Then).Post("/", app.createNewCategoryHandler)
	categoryRoutes.With(adminMIddleware.Then).Patch("/{categoryID:[0-9]+}/{versionID:[0-9]+}", app.updateCategoryHandler)
	categoryRoutes.With(adminMIddleware.Then).Delete("/{categoryID:[0-9]+}", app.deleteCategoryByIDHandler)

	return categoryRoutes
}

// productRoutes() is a method that returns a chi.Router that contains all the product routes
func (app *application) productRoutes(adminMIddleware *alice.Chain) chi.Router {
	productRoutes := chi.NewRouter()
	// get all products, open to everyone who is authenticated
	productRoutes.Get("/", app.getAllProductsHandler)

	// Create a new product, open to everyone who is authenticated
	productRoutes.With(adminMIddleware.Then).Post("/", app.createNewProductsHandler)

	return productRoutes
}

func (app *application) orderRoutes(adminPermissionMiddleware *alice.Chain) chi.Router {
	orderRoutes := chi.NewRouter()
	// Create a new order, open to everyone who is authenticated
	orderRoutes.Post("/", app.createOrderHandler)
	// Get all orders, open to everyone who is authenticated
	orderRoutes.Get("/", app.getUserOrdersHandler)

	// admin only routes
	orderRoutes.With(adminPermissionMiddleware.Then).Get("/admin", app.getAllOrdersHandler)
	orderRoutes.With(adminPermissionMiddleware.Then).Get("/statistics", app.getOrderStatisticsHandler)
	orderRoutes.With(adminPermissionMiddleware.Then).Patch("/{orderID:[0-9]+}", app.updateOrderStatusHandler)

	return orderRoutes
}
