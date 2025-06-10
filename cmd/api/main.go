package main

import (
	"database/sql"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Blue-Davinci/SavannaCart/internal/data"
	"github.com/Blue-Davinci/SavannaCart/internal/database"
	"github.com/Blue-Davinci/SavannaCart/internal/logger"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
)

var (
	version = "0.1.0"
)

type config struct {
	port int
	env  string
	api  struct {
		name               string
		author             string
		version            string
		oidc_client_id     string
		oidc_client_secret string
	}
	db struct {
		dsn          string
		maxOpenConns int
		maxIdleConns int
		maxIdleTime  string
	}
	cors struct {
		trustedOrigins []string
	}
	app_urls struct {
		authentication_callback_url string
		activation_callback_url     string
		provide_url                 string
	}
	authenticators struct {
		provider    *oidc.Provider
		verifier    *oidc.IDTokenVerifier
		oauthConfig *oauth2.Config
	}
	smtp struct {
		host     string
		port     int
		username string
		password string
		sender   string
	}
}

// app struct for dependency injection
type application struct {
	config config
	logger *zap.Logger
	models data.Models
	wg     sync.WaitGroup
}

func main() {
	// initialize logger
	logger, err := logger.InitJSONLogger()
	if err != nil {
		fmt.Printf("Error initializing logger: %s. Version:  %s", err, version)
		return
	}
	// config
	var cfg config
	// Load the environment variables from the .env file
	getCurrentPath(logger)
	// Load our configurations from the Flags
	// Port & env
	flag.IntVar(&cfg.port, "port", 4000, "API server port")
	flag.StringVar(&cfg.env, "env", "development", "Environment (development|staging|production)")
	// Database configuration
	flag.StringVar(&cfg.db.dsn, "db-dsn", os.Getenv("SAVANNACART_DB_DSN"), "PostgreSQL DSN")
	flag.IntVar(&cfg.db.maxOpenConns, "db-max-open-conns", 25, "PostgreSQL max open connections")
	flag.IntVar(&cfg.db.maxIdleConns, "db-max-idle-conns", 25, "PostgreSQL max idle connections")
	flag.StringVar(&cfg.db.maxIdleTime, "db-max-idle-time", "15m", "PostgreSQL max connection idle time")
	// api configuration
	flag.StringVar(&cfg.api.name, "api-name", "SavannaCart", "API Name")
	flag.StringVar(&cfg.api.author, "api-author", "Blue-Davinci", "API Author")
	flag.StringVar(&cfg.api.oidc_client_id, "oidc-client-id", os.Getenv("SAVANNACART_OIDC_CLIENT_ID"), "OIDC Client ID")
	flag.StringVar(&cfg.api.oidc_client_secret, "oidc-client-secret", os.Getenv("SAVANNACART_OIDC_CLIENT_SECRET"), "OIDC Client Secret")
	// urls configuration
	flag.StringVar(&cfg.app_urls.authentication_callback_url, "authentication-callback-url", "http://localhost:4000/v1/api/authentication", "Authentication Callback URL")
	flag.StringVar(&cfg.app_urls.activation_callback_url, "activation-callback-url", "http://localhost:4000/v1/api/activation", "Activation Callback URL")
	flag.StringVar(&cfg.app_urls.provide_url, "provide-url", "https://accounts.google.com", "Provide URL")
	// SMTP configuration
	flag.StringVar(&cfg.smtp.host, "smtp-host", os.Getenv("SAVANNACART_SMTP_HOST"), "SMTP server hostname")
	flag.IntVar(&cfg.smtp.port, "smtp-port", 587, "SMTP server port")
	flag.StringVar(&cfg.smtp.username, "smtp-username", os.Getenv("SAVANNACART_SMTP_USERNAME"), "SMTP server username")
	flag.StringVar(&cfg.smtp.password, "smtp-password", os.Getenv("SAVANNACART_SMTP_PASSWORD"), "SMTP server password")
	flag.StringVar(&cfg.smtp.sender, "smtp-sender", os.Getenv("SAVANNACART_SMTP_SENDER"), "SMTP sender email address")
	// CORS configuration
	flag.Func("cors-trusted-origins", "Trusted CORS origins (space separated)", func(val string) error {
		cfg.cors.trustedOrigins = strings.Fields(val)
		return nil

	})

	// Parse the flags
	flag.Parse()

	// Load additional configuration from environment variables
	loadConfig(&cfg)

	// create our connection pull
	db, err := openDB(cfg)
	if err != nil {
		logger.Fatal(err.Error(), zap.String("dsn", cfg.db.dsn))
	}

	app := &application{
		config: cfg,
		logger: logger,
		models: data.NewModels(db),
	}
	// Initialize OIDC at startup
	err = app.InitOIDC()
	if err != nil {
		logger.Fatal("Failed to initialize OIDC", zap.Error(err))
	}
	logger.Info("OIDC initialized successfully")
	// Initialize the server
	logger.Info("Loaded Cors Origins", zap.Strings("origins", cfg.cors.trustedOrigins))
	err = app.server()
	if err != nil {
		logger.Fatal("Error while starting server.", zap.String("error", err.Error()))
	}

}

// getCurrentPath invokes getEnvPath to get the path to the .env file based on the current working directory.
// After that it loads the .env file using godotenv.Load to be used by the initFlags() function
func getCurrentPath(logger *zap.Logger) string {
	currentpath := getEnvPath(logger)
	if currentpath != "" {
		err := godotenv.Load(currentpath)
		if err != nil {
			logger.Fatal(err.Error(), zap.String("path", currentpath))
		}
	} else {

		logger.Error("Path Error", zap.String("path", currentpath), zap.String("error", "unable to load .env file"))
	}
	logger.Info("Loading Environment Variables", zap.String("path", currentpath))
	return currentpath
}

// getEnvPath returns the path to the .env file based on the current working directory.
func getEnvPath(logger *zap.Logger) string {
	dir, err := os.Getwd()
	if err != nil {
		logger.Fatal(err.Error(), zap.String("path", dir))
		return ""
	}
	if strings.Contains(dir, "cmd/api") || strings.Contains(dir, "cmd") {
		return ".env"
	}
	return filepath.Join("cmd", "api", ".env")
}

// openDB() opens a new database connection using the provided configuration.
// It returns a pointer to the sql.DB connection pool and an error value.
func openDB(cfg config) (*database.Queries, error) {
	db, err := sql.Open("postgres", cfg.db.dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(cfg.db.maxOpenConns)
	db.SetMaxIdleConns(cfg.db.maxIdleConns)
	duration, err := time.ParseDuration(cfg.db.maxIdleTime)
	if err != nil {
		return nil, err
	}
	db.SetConnMaxIdleTime(duration)
	// Use ping to establish new conncetions
	err = db.Ping()
	if err != nil {
		return nil, err
	}
	queries := database.New(db)
	return queries, nil
}

// loadConfig loads additional configuration values from environment variables
func loadConfig(cfg *config) {
	// Set API configuration
	cfg.api.name = getEnvDefault("SAVANNACART_API_NAME", "SavannaCart API")
	cfg.api.author = getEnvDefault("SAVANNACART_API_AUTHOR", "Blue-Davinci")
	cfg.api.version = version

	// Set CORS trusted origins (comma-separated)
	originsStr := getEnvDefault("SAVANNACART_CORS_TRUSTED_ORIGINS", "http://localhost:3000,http://localhost:8080")
	if originsStr != "" {
		cfg.cors.trustedOrigins = strings.Split(originsStr, ",")
		// Trim spaces from each origin
		for i, origin := range cfg.cors.trustedOrigins {
			cfg.cors.trustedOrigins[i] = strings.TrimSpace(origin)
		}
	}
}

// getEnvDefault gets an environment variable with a default fallback
func getEnvDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
