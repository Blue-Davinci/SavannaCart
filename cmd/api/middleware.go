package main

import (
	"errors"
	"expvar"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/Blue-Davinci/SavannaCart/internal/data"
	"github.com/felixge/httpsnoop"
	"github.com/tomasen/realip"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

func (app *application) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Create a deferred function (which will always be run in the event of a panic
		// as Go unwinds the stack).
		defer func() {
			// Use the builtin recover function to check if there has been a panic or
			// not.
			if err := recover(); err != nil {
				// If there was a panic, set a "Connection: close" header on the
				// response. This acts as a trigger to make Go's HTTP server
				// automatically close the current connection after a response has been
				// sent.
				w.Header().Set("Connection", "close")
				// The value returned by recover() has the type any, so we use
				// fmt.Errorf() to normalize it into an error and call our
				// serverErrorResponse() helper. In turn, this will log the error using
				// our custom Logger type at the ERROR level and send the client a 500
				// Internal Server Error response.
				app.serverErrorResponse(w, r, fmt.Errorf("%s", err))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func (app *application) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Add the "Vary: Authorization" header to the response. This indicates to any
		// caches that the response may vary based on the value of the Authorization
		// header in the request.
		w.Header().Add("Vary", "Authorization")
		// Retrieve the value of the Authorization header from the request. This will
		// return the empty string "" if there is no such header found.
		user, err := app.aunthenticatorHelper(r)
		if user == data.AnonymousUser {
			r = app.contextSetUser(r, data.AnonymousUser)
			next.ServeHTTP(w, r)
			return
		}
		if err != nil {
			switch {
			case errors.Is(err, ErrInvalidAuthentication):
				app.invalidAuthenticationTokenResponse(w, r)
			case errors.Is(err, data.ErrGeneralRecordNotFound):
				app.invalidAuthenticationTokenResponse(w, r)
			default:
				app.serverErrorResponse(w, r, err)
			}
			return
		}
		// Call the contextSetUser() helper to add the user information to the request
		// context.
		app.logger.Info("user authenticated", zap.String("name", user.FirstName), zap.String("email", user.Email))
		r = app.contextSetUser(r, user)
		// Call the next handler in the chain.
		next.ServeHTTP(w, r)
	})
}

// Create a new requireAuthenticatedUser() middleware to check that a user is not
// anonymous.
func (app *application) requireAuthenticatedUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Use the contextGetUser() helper to retrieve the user
		// information from the request context.
		app.logger.Debug("requireAuthenticatedUser middleware called")
		user := app.contextGetUser(r)
		// If the user is anonymous, then call the authenticationRequiredResponse() to
		// inform the client that they should authenticate before trying again.
		if user.IsAnonymous() {
			app.authenticationRequiredResponse(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Checks that a user is both authenticated and activated.
func (app *application) requireActivatedUser(next http.Handler) http.Handler {
	// Rather than returning this http.HandlerFunc we assign it to the variable fn.
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := app.contextGetUser(r)
		// If the user is not activated, use the inactiveAccountResponse() helper to
		// inform them that they need to activate their account.
		if !user.Activated {
			app.inactiveAccountResponse(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// requirePermission takes the first parameter for th,e permission code that
// we require the user to have. It then proceeds to read whether the user has that
// permission or not. If they do not, then it returns a 403 Forbidden response.
// If they do have the permission, then it calls the next handler in the chain.
func (app *application) requirePermission(code string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Retrieve the user from the request context.
			user := app.contextGetUser(r)
			// Get the slice of permissions for the user.
			permissions, err := app.models.Permissions.GetAllPermissionsForUser(user.ID)
			if err != nil {
				app.serverErrorResponse(w, r, err)
				return
			}
			// Check if the slice includes the required permission. If it doesn't, then
			// return a 403 Forbidden response.
			if !permissions.Include(code) {
				app.notPermittedResponse(w, r)
				return
			}
			// Otherwise they have the required permission so we call the next handler in
			// the chain.
			next.ServeHTTP(w, r)
		})
	}
}

// The rateLimit() middleware will be used to rate limit the number of requests that a
// client can make to certain routes within a given time window.
func (app *application) rateLimit(next http.Handler) http.Handler {
	// Define a client struct to hold the rate limiter and last seen time for each
	// client.
	type client struct {
		limiter  *rate.Limiter
		lastSeen time.Time
	}

	// Declare a mutex and a map to hold the clients' IP addresses and rate limiters.
	var (
		mu      sync.Mutex
		clients = make(map[string]*client)
	)
	// Launch a background goroutine which removes old entries from the clients map once
	// every minute.
	go func() {
		for {
			time.Sleep(time.Minute)
			// Lock the mutex to prevent any rate limiter checks from happening while
			// the cleanup is taking place.
			mu.Lock()
			// Loop through all clients. If they haven't been seen within the last three
			// minutes, delete the corresponding entry from the map.
			for ip, client := range clients {
				if time.Since(client.lastSeen) > 3*time.Minute {
					delete(clients, ip)
				}
			}
			// Importantly, unlock the mutex when the cleanup is complete.
			mu.Unlock()
		}
	}()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only carry out the check if rate limiting is enabled.
		if app.config.limiter.enabled {
			// Extract the client's IP address from the request.
			ip := realip.FromRequest(r)
			// Lock the mutex to prevent this code from being executed concurrently.
			mu.Lock()
			// Check to see if the IP address already exists in the map. If it doesn't, then
			// initialize a new rate limiter and add the IP address and limiter to the map.
			if _, found := clients[ip]; !found {
				clients[ip] = &client{
					// Use the requests-per-second and burst values from the config struct.
					limiter: rate.NewLimiter(rate.Limit(app.config.limiter.rps), app.config.limiter.burst),
				}
			}
			// Update the last seen time for the client.
			clients[ip].lastSeen = time.Now()

			// Call the Allow() method on the rate limiter for the current IP address. If
			// the request isn't allowed, unlock the mutex and send a 429 Too Many Requests
			// response, just like before.
			if !clients[ip].limiter.Allow() {
				mu.Unlock()
				app.rateLimitExceededResponse(w, r)
				return
			}
			// unlock the mutex before calling the next handler in the
			// chain
			mu.Unlock()
		}
		next.ServeHTTP(w, r)
	})
}
func (app *application) metrics(next http.Handler) http.Handler {
	// Initialize the new expvar variables when the middleware chain is first built.
	totalRequestsReceived := expvar.NewInt("total_requests_received")
	totalResponsesSent := expvar.NewInt("total_responses_sent")
	totalProcessingTimeMicroseconds := expvar.NewInt("total_processing_time_μs")
	totalResponsesSentByStatus := expvar.NewMap("total_responses_sent_by_status")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Increment the number of requests received by 1.
		totalRequestsReceived.Add(1)

		// Use httpsnoop to capture metrics while passing along the original response writer.
		metrics := httpsnoop.CaptureMetrics(next, w, r)

		// Increment the total responses sent.
		totalResponsesSent.Add(1)
		// Increment the processing time.
		totalProcessingTimeMicroseconds.Add(metrics.Duration.Microseconds())
		// Increment the count for the response status code.
		totalResponsesSentByStatus.Add(strconv.Itoa(metrics.Code), 1)
	})
}
