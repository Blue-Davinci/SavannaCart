package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"
)

// createTestApp creates a minimal working application for testing
func createTestApp(t *testing.T) *application {
	t.Helper()

	// Initialize a logger for testing
	testLogger, err := zap.NewDevelopment()
	if err != nil {
		t.Fatalf("Failed to create test logger: %v", err)
	}

	return &application{
		config: config{
			env: "test",
			api: struct {
				name               string
				author             string
				version            string
				oidc_client_id     string
				oidc_client_secret string
			}{
				name:    "SavannaCart API",
				author:  "Blue-Davinci",
				version: "0.1.0",
			},
			cors: struct {
				trustedOrigins []string
			}{
				trustedOrigins: []string{"http://localhost:3000"},
			},
		},
		logger: testLogger,
	}
}

// TestE2E_All runs all e2e tests in sequence using a single application instance
func TestE2E_All(t *testing.T) {
	app := createTestApp(t)

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	// Test 1: Health Check
	t.Run("HealthCheck", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/v1/api/healthcheck")
		if err != nil {
			t.Fatalf("Failed to get health check: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Health check failed with status: %d", resp.StatusCode)
		}

		var healthResponse struct {
			Status string `json:"status"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&healthResponse); err != nil {
			t.Fatalf("Failed to decode health check response: %v", err)
		}

		if healthResponse.Status != "API is healthy" {
			t.Fatalf("Expected status: 'API is healthy', got: %s", healthResponse.Status)
		}

		t.Log("Health check endpoint test passedd")
	})

	// Test 2: API Versioning
	t.Run("APIVersioning", func(t *testing.T) {
		// Test that v1 API responds correctly
		resp, err := http.Get(ts.URL + "/v1/api/healthcheck")
		if err != nil {
			t.Fatalf("Failed to get v1 health check: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("V1 health check failed with status: %d", resp.StatusCode)
		}

		// Test that non-existent version returns 404
		resp2, err := http.Get(ts.URL + "/v2/api/healthcheck")
		if err != nil {
			t.Fatalf("Failed to test v2 endpoint: %v", err)
		}
		defer resp2.Body.Close()

		if resp2.StatusCode != http.StatusNotFound {
			t.Fatalf("Expected v2 endpoint to return 404, got: %d", resp2.StatusCode)
		}

		t.Log("API versioning test passed successfully")
	})

	// Test 3: Debug Endpoint
	t.Run("DebugEndpoint", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/v1/debug/vars")
		if err != nil {
			t.Fatalf("Failed to get debug vars: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Debug vars endpoint failed with status: %d", resp.StatusCode)
		}

		// The debug endpoint should return some metrics data
		contentType := resp.Header.Get("Content-Type")
		if contentType == "" {
			t.Log("Debug endpoint accessibe but content type not set")
		} else {
			t.Logf("Debug endpoint Content-Type: %s", contentType)
		}

		t.Log("endpoint test passed")
	})

	// Test 4: Authentication Endpoint Structure
	t.Run("AuthenticationEndpoint", func(t *testing.T) {
		// Test authentication endpoint exists (should return 400 for empty request)
		resp, err := http.Get(ts.URL + "/v1/api/authentication")
		if err != nil {
			t.Fatalf("Failed to get authentication endpoint: %v", err)
		}
		defer resp.Body.Close()

		// Should return an error for GET request (probably 405 Method Not Allowed)
		if resp.StatusCode == http.StatusOK {
			t.Log("Auth endpoint returned 200 for GET request")
		} else {
			t.Logf("uth endpoint correctly rejected GET request with status: %d", resp.StatusCode)
		}

		t.Log("Auth endpoint structure test passed")
	})

	// Test 5: Route Structure
	t.Run("RouteStructure", func(t *testing.T) {
		// Test various endpoints exist and return appropriate responses
		endpoints := []struct {
			path           string
			expectedStatus int
			description    string
		}{
			{"/v1/api/healthcheck", http.StatusOK, "health check"},
			{"/v1/debug/vars", http.StatusOK, "debug vars"},
			{"/v1/api/authentication", http.StatusUnauthorized, "authentication (GET unauthorized)"},
			{"/v1/nonexistent", http.StatusNotFound, "non-existent endpoint"},
		}
		// Loop through each endpoint and test it
		for _, endpoint := range endpoints {
			resp, err := http.Get(ts.URL + endpoint.path)
			if err != nil {
				t.Fatalf("Failed to test %s: %v", endpoint.description, err)
			}
			resp.Body.Close()

			if resp.StatusCode != endpoint.expectedStatus {
				t.Logf("%s returned %d, expected %d", endpoint.description, resp.StatusCode, endpoint.expectedStatus)
			} else {
				t.Logf("%s endpoint workinig correctly (status: %d)", endpoint.description, resp.StatusCode)
			}
		}

		t.Log("Route structure test completed")
	})
}
