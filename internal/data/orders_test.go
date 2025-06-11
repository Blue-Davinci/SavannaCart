package data

import (
	"testing"

	"github.com/Blue-Davinci/SavannaCart/internal/validator"
)

func TestValidateCreateOrderRequest(t *testing.T) {
	tests := []struct {
		name           string
		request        *CreateOrderRequest
		expectedErrors []string
	}{
		{
			name: "valid order request",
			request: &CreateOrderRequest{
				UserID: 1,
				Items: []*CreateOrderItemRequest{
					{ProductID: 1, Quantity: 2},
					{ProductID: 2, Quantity: 1},
				},
			},
			expectedErrors: []string{},
		},
		{
			name: "invalid user ID",
			request: &CreateOrderRequest{
				UserID: 0,
				Items: []*CreateOrderItemRequest{
					{ProductID: 1, Quantity: 2},
				},
			},
			expectedErrors: []string{"user_id"},
		},
		{
			name: "negative user ID",
			request: &CreateOrderRequest{
				UserID: -1,
				Items: []*CreateOrderItemRequest{
					{ProductID: 1, Quantity: 2},
				},
			},
			expectedErrors: []string{"user_id"},
		},
		{
			name: "empty items",
			request: &CreateOrderRequest{
				UserID: 1,
				Items:  []*CreateOrderItemRequest{},
			},
			expectedErrors: []string{"items"},
		},
		{
			name: "nil items",
			request: &CreateOrderRequest{
				UserID: 1,
				Items:  nil,
			},
			expectedErrors: []string{"items"},
		},
		{
			name: "invalid product ID in item",
			request: &CreateOrderRequest{
				UserID: 1,
				Items: []*CreateOrderItemRequest{
					{ProductID: 0, Quantity: 2},
				},
			},
			expectedErrors: []string{"items[0].product_id"},
		},
		{
			name: "invalid quantity in item",
			request: &CreateOrderRequest{
				UserID: 1,
				Items: []*CreateOrderItemRequest{
					{ProductID: 1, Quantity: 0},
				},
			},
			expectedErrors: []string{"items[0].quantity"},
		},
		{
			name: "negative quantity in item",
			request: &CreateOrderRequest{
				UserID: 1,
				Items: []*CreateOrderItemRequest{
					{ProductID: 1, Quantity: -1},
				},
			},
			expectedErrors: []string{"items[0].quantity"},
		},
		{
			name: "multiple invalid items",
			request: &CreateOrderRequest{
				UserID: 1,
				Items: []*CreateOrderItemRequest{
					{ProductID: 0, Quantity: 0},
					{ProductID: -1, Quantity: -5},
				},
			},
			expectedErrors: []string{
				"items[0].product_id",
				"items[0].quantity",
				"items[1].product_id",
				"items[1].quantity",
			},
		},
		{
			name: "all validation errors",
			request: &CreateOrderRequest{
				UserID: 0,
				Items: []*CreateOrderItemRequest{
					{ProductID: 0, Quantity: 0},
				},
			},
			expectedErrors: []string{
				"user_id",
				"items[0].product_id",
				"items[0].quantity",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := validator.New()
			ValidateCreateOrderRequest(v, tt.request)

			// Check that we have the expected number of errors
			if len(v.Errors) != len(tt.expectedErrors) {
				t.Errorf("Expected %d errors, got %d. Errors: %v", len(tt.expectedErrors), len(v.Errors), v.Errors)
			}

			// Check that all expected error fields are present
			for _, expectedField := range tt.expectedErrors {
				if _, exists := v.Errors[expectedField]; !exists {
					t.Errorf("Expected error for field '%s', but it was not found. Actual errors: %v", expectedField, v.Errors)
				}
			}
		})
	}
}

func TestValidateOrderStatus(t *testing.T) {
	tests := []struct {
		name           string
		status         string
		expectedErrors []string
	}{
		{
			name:           "valid status - PLACED",
			status:         OrderStatusPlaced,
			expectedErrors: []string{},
		},
		{
			name:           "valid status - PROCESSING",
			status:         OrderStatusProcessing,
			expectedErrors: []string{},
		},
		{
			name:           "valid status - SHIPPED",
			status:         OrderStatusShipped,
			expectedErrors: []string{},
		},
		{
			name:           "valid status - DELIVERED",
			status:         OrderStatusDelivered,
			expectedErrors: []string{},
		},
		{
			name:           "valid status - CANCELLED",
			status:         OrderStatusCancelled,
			expectedErrors: []string{},
		},
		{
			name:           "empty status",
			status:         "",
			expectedErrors: []string{"status"},
		},
		{
			name:           "invalid status",
			status:         "INVALID_STATUS",
			expectedErrors: []string{"status"},
		},
		{
			name:           "lowercase valid status",
			status:         "placed",
			expectedErrors: []string{"status"},
		},
		{
			name:           "mixed case valid status",
			status:         "Placed",
			expectedErrors: []string{"status"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := validator.New()
			ValidateOrderStatus(v, tt.status)

			// Check that we have the expected number of errors
			if len(v.Errors) != len(tt.expectedErrors) {
				t.Errorf("Expected %d errors, got %d. Errors: %v", len(tt.expectedErrors), len(v.Errors), v.Errors)
			}

			// Check that all expected error fields are present
			for _, expectedField := range tt.expectedErrors {
				if _, exists := v.Errors[expectedField]; !exists {
					t.Errorf("Expected error for field '%s', but it was not found. Actual errors: %v", expectedField, v.Errors)
				}
			}
		})
	}
}

func TestOrderStatusConstants(t *testing.T) {
	// We'll validate through the ValidateOrderStatus function
	validStatuses := []string{
		OrderStatusPlaced,
		OrderStatusProcessing,
		OrderStatusShipped,
		OrderStatusDelivered,
		OrderStatusCancelled,
	}

	for _, status := range validStatuses {
		t.Run("status_"+status, func(t *testing.T) {
			v := validator.New()
			ValidateOrderStatus(v, status)

			if !v.Valid() {
				t.Errorf("Status %s should be valid, but got errors: %v", status, v.Errors)
			}
		})
	}

	// Test that we have exactly 5 valid statuses
	if len(validStatuses) != 5 {
		t.Errorf("Expected 5 order statuses, got %d", len(validStatuses))
	}

	// Check for expected values
	statusMap := map[string]bool{
		"PLACED":     false,
		"PROCESSING": false,
		"SHIPPED":    false,
		"DELIVERED":  false,
		"CANCELLED":  false,
	}

	for _, status := range validStatuses {
		if _, exists := statusMap[status]; exists {
			statusMap[status] = true
		} else {
			t.Errorf("Unexpected status constant: %s", status)
		}
	}

	// Check that all expected statuses were found
	for expectedStatus, found := range statusMap {
		if !found {
			t.Errorf("Expected status %s not found in constants", expectedStatus)
		}
	}
}

func TestValidateCreateOrderRequestEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		request *CreateOrderRequest
		valid   bool
	}{
		{
			name: "very large user ID",
			request: &CreateOrderRequest{
				UserID: 2147483647, // max int32
				Items: []*CreateOrderItemRequest{
					{ProductID: 1, Quantity: 1},
				},
			},
			valid: true,
		},
		{
			name: "many items",
			request: &CreateOrderRequest{
				UserID: 1,
				Items: func() []*CreateOrderItemRequest {
					items := make([]*CreateOrderItemRequest, 100)
					for i := range items {
						items[i] = &CreateOrderItemRequest{
							ProductID: int32(i + 1),
							Quantity:  1,
						}
					}
					return items
				}(),
			},
			valid: true,
		},
		{
			name: "very large quantities",
			request: &CreateOrderRequest{
				UserID: 1,
				Items: []*CreateOrderItemRequest{
					{ProductID: 1, Quantity: 2147483647}, // max int32
				},
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := validator.New()
			ValidateCreateOrderRequest(v, tt.request)

			isValid := v.Valid()
			if isValid != tt.valid {
				t.Errorf("Expected valid=%v, got valid=%v, errors: %v", tt.valid, isValid, v.Errors)
			}
		})
	}
}
