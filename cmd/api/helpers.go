package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Blue-Davinci/SavannaCart/internal/data"
	"github.com/Blue-Davinci/SavannaCart/internal/validator"
	"github.com/go-chi/chi/v5"
)

var (
	ErrInvalidAuthentication = errors.New("invalid authentication token format")
	ErrNoDataFoundInRedis    = errors.New("no data found in Redis")
)

// Define an envelope type.
type envelope map[string]any

// Define a writeJSON() helper for sending responses. This takes the destination
// http.ResponseWriter, the HTTP status code to send, the data to encode to JSON, and a
// header map containing any additional HTTP headers we want to include in the response.
func (app *application) writeJSON(w http.ResponseWriter, status int, data envelope, headers http.Header) error {
	// Encode the data to JSON, returning the error if there was one.
	js, err := json.MarshalIndent(data, "", "\t")
	if err != nil {
		return err
	}
	// Append a newline to make it easier to view in terminal applications.
	js = append(js, '\n')
	// At this point, we know that we won't encounter any more errors before writing the
	// response, so it's safe to add any headers that we want to include.
	for key, value := range headers {
		w.Header()[key] = value
	}
	// Add the "Content-Type: application/json" header, then write the status code
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(js)
	return nil
}

func (app *application) readJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	// Use http.MaxBytesReader() to limit the size of the request body to 1MB.
	maxBytes := 1_048_576
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes))
	// Initialize the json.Decoder, and call the DisallowUnknownFields() method on it
	// before decoding. This means that if the JSON from the client now includes any
	// field which cannot be mapped to the target destination, the decoder will return
	// an error instead of just ignoring the field.
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	// Decode the request body to the destination.
	err := dec.Decode(dst)
	err = app.jsonReadAndHandleError(err)
	if err != nil {
		return err
	}
	// Call Decode() again, using a pointer to an empty anonymous struct as the
	// destination. If the request body only contained a single JSON value this will
	// return an io.EOF error. So if we get anything else, we know that there is
	// additional data in the request body and we return our own custom error message.
	err = dec.Decode(&struct{}{})
	if err != io.EOF {
		return errors.New("body must only contain a single JSON value")
	}
	return nil
}

// Retrieve the "id" URL parameter from the current request context, then convert it to
// an integer and return it. If the operation isn't successful, return a nil UUID and an error.
func (app *application) readIDParam(r *http.Request, parameterName string) (int64, error) {
	// We use chi's URLParam method to get our ID parameter from the URL.
	params := chi.URLParam(r, parameterName)
	id, err := strconv.ParseInt(params, 10, 64)
	if err != nil || id < 1 {
		return 0, errors.New("invalid i-id parameter")
	}
	return id, nil
}

// jsonReadAndHandleError() is a helper function that takes an error as a parameter and
// returns a cleaned-up error message. This is used to provide more information in the
// event of a JSON decoding error.
func (app *application) jsonReadAndHandleError(err error) error {
	if err != nil {
		// Vars to carry our errors
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError
		var invalidUnmarshalError *json.InvalidUnmarshalError
		// Add a new maxBytesError variable.
		var maxBytesError *http.MaxBytesError
		switch {
		case errors.As(err, &syntaxError):
			return fmt.Errorf("body contains badly-formed JSON (at character %d)", syntaxError.Offset)
		case errors.Is(err, io.ErrUnexpectedEOF):
			return errors.New("body contains badly-formed JSON")
		case errors.As(err, &unmarshalTypeError):
			if unmarshalTypeError.Field != "" {
				return fmt.Errorf("body contains incorrect JSON type for field %q", unmarshalTypeError.Field)
			}
			return fmt.Errorf("body contains incorrect JSON type (at character %d)", unmarshalTypeError.Offset)
		case errors.Is(err, io.EOF):
			return errors.New("body must not be empty")
		// If the JSON contains a field which cannot be mapped to the target destination
		// then Decode() will now return an error message in the format "json: unknown
		// field "<name>"". We check for this, extract the field name from the error,
		// and interpolate it into our custom error message. Note that there's an open
		// issue at https://github.com/golang/go/issues/29035 regarding turning this
		// into a distinct error type in the future.
		case strings.HasPrefix(err.Error(), "json: unknown field "):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
			return fmt.Errorf("body contains unknown key %s", fieldName)
		// Use the errors.As() function to check whether the error has the type
		// *http.MaxBytesError. If it does, then it means the request body exceeded our
		// size limit of 1MB and we return a clear error message.
		case errors.As(err, &maxBytesError):
			return fmt.Errorf("body must not be larger than %d bytes", maxBytesError.Limit)
		case errors.As(err, &invalidUnmarshalError):
			panic(err)
		default:
			return err
		}
	}
	return nil
}

// The background() helper accepts an arbitrary function as a parameter.
// It launches a background goroutine to execute the function.
// The done() method of the WaitGroup is called when the goroutine completes.
func (app *application) background(fn func()) {
	app.wg.Add(1)
	// Launch a background goroutine.
	go func() {
		//defer our done()
		defer app.wg.Done()
		// Recover any panic.
		defer func() {
			if err := recover(); err != nil {
				app.logger.Error(fmt.Sprintf("%s", err))
			}
		}()
		// Execute the arbitrary function that we passed as the parameter.
		fn()
	}()
}

// The readString() helper returns a string value from the query string, or the provided
// default value if no matching key could be found.
func (app *application) readString(qs url.Values, key string, defaultValue string) string {
	// Extract the value for a given key from the query string. If no key exists this
	// will return the empty string "".
	s := qs.Get(key)
	// If no key exists (or the value is empty) then return the default value.
	if s == "" {
		return defaultValue
	}
	// Otherwise return the string.
	return s
}

// The readInt() helper reads a string value from the query string and converts it to an
// integer before returning. If no matching key could be found it returns the provided
// default value. If the value couldn't be converted to an integer, then we record an
// error message in the provided Validator instance.
func (app *application) readInt(qs url.Values, key string, defaultValue int, v *validator.Validator) int {
	// Extract the value from the query string.
	s := qs.Get(key)
	// If no key exists (or the value is empty) then return the default value.
	if s == "" {
		return defaultValue
	}
	// Try to convert the value to an int. If this fails, add an error message to the
	// validator instance and return the default value.
	i, err := strconv.Atoi(s)
	if err != nil {
		v.AddError(key, "must be an integer value")
		return defaultValue
	}
	// Otherwise, return the converted integer value.
	return i
}

// readFloat() reads a string value from the query string and converts it to a float64
// before returning. If no matching key could be found it returns the provided default value.
// If the value couldn't be converted to a float64, then we record an error message in the
// provided Validator instance.
func (app *application) readFloat64(qs url.Values, key string, defaultValue float64, v *validator.Validator) float64 {
	// Extract the value from the query string.
	s := qs.Get(key)
	// If no key exists (or the value is empty) then return the default value.
	if s == "" {
		return defaultValue
	}
	// Try to convert the value to a float64. If this fails, add an error message to the
	// validator instance and return the default value.
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		v.AddError(key, "must be a float value")
		return defaultValue
	}
	// Otherwise, return the converted float64 value.
	return f
}

// readBoolean() reads a string value from the query string and converts it to a boolean
// before returning. If no matching key could be found it returns the provided default value.
// If the value couldn't be converted to a boolean, then we record an error message in the
// provided Validator instance.
func (app *application) readBoolean(qs url.Values, key string, defaultValue bool, v *validator.Validator) bool {
	// Extract the value from the query string.
	s := qs.Get(key)
	// If no key exists (or the value is empty) then return the default value.
	if s == "" {
		return defaultValue
	}
	// Try to convert the value to a boolean. If this fails, add an error message to the
	// validator instance and return the default value.
	b, err := strconv.ParseBool(s)
	if err != nil {
		v.AddError(key, "must be a boolean value")
		return defaultValue
	}
	// Otherwise, return the converted boolean value.
	return b
}

// readDate() reads a string value from the query string and converts it to a time.Time
// before returning. If no matching key could be found it returns the provided default value.
// If the value couldn't be converted to a time.Time, then we record an error message in the
// provided Validator instance.
func (app *application) readDate(qs url.Values, key string, defaultValue time.Time, v *validator.Validator) time.Time {
	// Extract the value from the query string.
	s := qs.Get(key)
	// If no key exists (or the value is empty) then return the default value.
	if s == "" {
		return defaultValue
	}
	// Try to convert the value to a time.Time. If this fails, add an error message to the
	// validator instance and return the default value.
	d, err := time.Parse("2006-01-02", s)
	if err != nil {
		v.AddError(key, "must be a date in the format YYYY-MM-DD")
		return defaultValue
	}
	// Otherwise, return the converted time.Time value.
	return d
}

// isLastDayOfMonth() checks if the given time is the last day of the month.
func (app *application) isLastDayOfMonth(t time.Time) bool {
	nextDay := t.AddDate(0, 0, 1) // Add one day
	return nextDay.Day() == 1     // If the next day is the first day of the month
}

// validateURL() checks if the input string is a valid URL
func validateURL(input string) error {
	parsedURL, err := url.ParseRequestURI(input)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	// Further validate URL components
	if parsedURL.Scheme == "" || parsedURL.Host == "" {
		return fmt.Errorf("URL must contain both scheme and host")
	}

	return nil
}

// aunthenticatorHelper() is a helper function for the authentication middleware
// It takes in a request and returns a user and an error
// It retrieves the value of the Authorization header from the request. This will
// return the empty string "" if there is no such header found.
func (app *application) aunthenticatorHelper(r *http.Request) (*data.User, error) {
	// Retrieve the value of the Authorization header from the request. This will
	// return the empty string "" if there is no such header found.
	authorizationHeader := r.Header.Get("Authorization")
	// If there is no Authorization header found, use the contextSetUser() helper to
	// add the AnonymousUser to the request context. Then we
	// call the next handler in the chain and return without executing any of the
	// code below.
	if authorizationHeader == "" {
		return data.AnonymousUser, nil
	}
	// Otherwise, we expect the value of the Authorization header to be in the format
	// "Bearer <token>". We try to split this into its constituent parts, and if the
	// header isn't in the expected format we return a 401 Unauthorized response
	// using the invalidAuthenticationTokenResponse() helper
	headerParts := strings.Split(authorizationHeader, " ")
	if len(headerParts) != 2 || headerParts[0] != "Bearer" {
		return nil, ErrInvalidAuthentication
	}
	//app.logger.Info("Authentication header found", zap.String("header", authorizationHeader))
	// Extract the actual authentication token from the header parts.
	token := headerParts[1]
	//app.logger.Info("User id Connected", zap.String("Connected ID", token))
	// Validate the token to make sure it is in a sensible format.
	v := validator.New()
	// If the token isn't valid, use the invalidAuthenticationTokenResponse()
	// helper to send a response, rather than the failedValidationResponse() helper
	// that we'd normally use.
	if data.ValidateTokenPlaintext(v, token); !v.Valid() {
		return nil, ErrInvalidAuthentication
	}
	//app.logger.Info("Authentication token validated", zap.String("token", token))
	// Retrieve the details of the user associated with the authentication token,
	// again calling the invalidAuthenticationTokenResponse() helper if no
	// matching record was found. IMPORTANT: Notice that we are using
	// ScopeAuthentication as the first parameter here.
	user, err := app.models.Users.GetForToken(data.ScopeAuthentication, token)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrGeneralRecordNotFound):
			return nil, ErrInvalidAuthentication
		default:
			return nil, ErrInvalidAuthentication
		}
	}
	//app.logger.Info("[Auth Helper] User authenticated", zap.Int64("user_id", user.ID), zap.String("email", user.Email))
	return user, nil
}
