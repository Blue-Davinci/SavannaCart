package data

import (
	"context"
	"database/sql"
	"testing"
	"time"
)

func TestContextGenerator(t *testing.T) {
	tests := []struct {
		name    string
		timeout time.Duration
	}{
		{
			name:    "5 second timeout",
			timeout: 5 * time.Second,
		},
		{
			name:    "1 minute timeout",
			timeout: time.Minute,
		},
		{
			name:    "100 millisecond timeout",
			timeout: 100 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			timeoutCtx, cancel := contextGenerator(ctx, tt.timeout)
			defer cancel()

			if timeoutCtx == nil {
				t.Fatal("Expected non-nil context")
			}

			// Check that the context has a deadline
			deadline, ok := timeoutCtx.Deadline()
			if !ok {
				t.Fatal("Expected context to have a deadline")
			}

			// Check that the deadline is approximately correct (within 1 second tolerance)
			expectedDeadline := time.Now().Add(tt.timeout)
			timeDiff := deadline.Sub(expectedDeadline)
			if timeDiff < -time.Second || timeDiff > time.Second {
				t.Errorf("Deadline is too far from expected. Expected around %v, got %v", expectedDeadline, deadline)
			}
		})
	}
}

func TestContextGeneratorCancellation(t *testing.T) {
	ctx := context.Background()
	timeoutCtx, cancel := contextGenerator(ctx, 100*time.Millisecond)

	// Cancel immediately
	cancel()

	select {
	case <-timeoutCtx.Done():
		// Context was cancelled as expected
		if timeoutCtx.Err() != context.Canceled {
			t.Errorf("Expected context.Canceled, got %v", timeoutCtx.Err())
		}
	case <-time.After(50 * time.Millisecond):
		t.Error("Context should have been cancelled immediately")
	}
}

func TestConvertValueToNullInt32(t *testing.T) {
	tests := []struct {
		name     string
		value    int32
		expected sql.NullInt32
	}{
		{
			name:  "positive value",
			value: 42,
			expected: sql.NullInt32{
				Int32: 42,
				Valid: true,
			},
		},
		{
			name:  "zero value",
			value: 0,
			expected: sql.NullInt32{
				Valid: false,
			},
		},
		{
			name:  "negative value",
			value: -1,
			expected: sql.NullInt32{
				Valid: false,
			},
		},
		{
			name:  "large positive value",
			value: 2147483647, // max int32
			expected: sql.NullInt32{
				Int32: 2147483647,
				Valid: true,
			},
		},
		{
			name:  "small positive value",
			value: 1,
			expected: sql.NullInt32{
				Int32: 1,
				Valid: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertValueToNullInt32(tt.value)

			if result.Valid != tt.expected.Valid {
				t.Errorf("Valid flag mismatch. Expected %v, got %v", tt.expected.Valid, result.Valid)
			}

			if result.Valid && result.Int32 != tt.expected.Int32 {
				t.Errorf("Value mismatch. Expected %d, got %d", tt.expected.Int32, result.Int32)
			}
		})
	}
}

func TestConvertValueToNullInt32BoundaryValues(t *testing.T) {
	// Test boundary values specifically
	boundaryTests := []struct {
		name     string
		value    int32
		expected bool // Expected validity
	}{
		{"exactly zero", 0, false},
		{"one above zero", 1, true},
		{"one below zero", -1, false},
		{"very negative", -1000, false},
		{"very positive", 1000, true},
	}

	for _, tt := range boundaryTests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertValueToNullInt32(tt.value)
			if result.Valid != tt.expected {
				t.Errorf("Expected Valid=%v for value %d, got Valid=%v", tt.expected, tt.value, result.Valid)
			}
		})
	}
}
