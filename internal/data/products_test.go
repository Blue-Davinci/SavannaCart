package data

import (
	"testing"

	"github.com/Blue-Davinci/SavannaCart/internal/validator"
	"github.com/shopspring/decimal"
)

func TestGenerateStockStatus(t *testing.T) {
	tests := []struct {
		name     string
		quantity int32
		expected string
	}{
		{
			name:     "zero quantity - out of stock",
			quantity: 0,
			expected: StockStatusOutOfStock,
		},
		{
			name:     "low stock - exactly at threshold",
			quantity: LowStockThreshold,
			expected: StockStatusLowStock,
		},
		{
			name:     "low stock - below threshold",
			quantity: LowStockThreshold - 1,
			expected: StockStatusLowStock,
		},
		{
			name:     "in stock - above threshold",
			quantity: LowStockThreshold + 1,
			expected: StockStatusInStock,
		},
		{
			name:     "high stock",
			quantity: 100,
			expected: StockStatusInStock,
		},
		{
			name:     "very low stock",
			quantity: 1,
			expected: StockStatusLowStock,
		},
		{
			name:     "medium stock",
			quantity: 50,
			expected: StockStatusInStock,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateStockStatus(tt.quantity)
			if result != tt.expected {
				t.Errorf("generateStockStatus(%d) = %s, want %s", tt.quantity, result, tt.expected)
			}
		})
	}
}

func TestGenerateStockStatusConstants(t *testing.T) {
	// Test that our constants are what we expect
	expectedConstants := map[string]string{
		"StockStatusInStock":    "in_stock",
		"StockStatusLowStock":   "low_stock",
		"StockStatusOutOfStock": "out_of_stock",
	}

	if StockStatusInStock != expectedConstants["StockStatusInStock"] {
		t.Errorf("StockStatusInStock = %s, want %s", StockStatusInStock, expectedConstants["StockStatusInStock"])
	}
	if StockStatusLowStock != expectedConstants["StockStatusLowStock"] {
		t.Errorf("StockStatusLowStock = %s, want %s", StockStatusLowStock, expectedConstants["StockStatusLowStock"])
	}
	if StockStatusOutOfStock != expectedConstants["StockStatusOutOfStock"] {
		t.Errorf("StockStatusOutOfStock = %s, want %s", StockStatusOutOfStock, expectedConstants["StockStatusOutOfStock"])
	}

	// Test low stock threshold
	if LowStockThreshold != 10 {
		t.Errorf("LowStockThreshold = %d, want 10", LowStockThreshold)
	}
}

func TestValidateProduct(t *testing.T) {
	tests := []struct {
		name           string
		product        *Product
		expectedErrors []string
	}{
		{
			name: "valid product",
			product: &Product{
				Name:          "Valid Product",
				PriceKES:      decimal.NewFromFloat(100.50),
				CategoryID:    1,
				StockQuantity: 50,
			},
			expectedErrors: []string{},
		},
		{
			name: "empty name",
			product: &Product{
				Name:          "",
				PriceKES:      decimal.NewFromFloat(100.50),
				CategoryID:    1,
				StockQuantity: 50,
			},
			expectedErrors: []string{"name"},
		},
		{
			name: "negative price",
			product: &Product{
				Name:          "Test Product",
				PriceKES:      decimal.NewFromFloat(-10.00),
				CategoryID:    1,
				StockQuantity: 50,
			},
			expectedErrors: []string{"price_kes"},
		},
		{
			name: "zero category ID",
			product: &Product{
				Name:          "Test Product",
				PriceKES:      decimal.NewFromFloat(100.50),
				CategoryID:    0,
				StockQuantity: 50,
			},
			expectedErrors: []string{"category_id"},
		},
		{
			name: "negative stock quantity",
			product: &Product{
				Name:          "Test Product",
				PriceKES:      decimal.NewFromFloat(100.50),
				CategoryID:    1,
				StockQuantity: -5,
			},
			expectedErrors: []string{"stock_quantity"},
		},
		{
			name: "multiple validation errors",
			product: &Product{
				Name:          "",
				PriceKES:      decimal.NewFromFloat(-10.00),
				CategoryID:    0,
				StockQuantity: -5,
			},
			expectedErrors: []string{"name", "price_kes", "category_id", "stock_quantity"},
		},
		{
			name: "zero price (valid)",
			product: &Product{
				Name:          "Free Product",
				PriceKES:      decimal.NewFromFloat(0.00),
				CategoryID:    1,
				StockQuantity: 10,
			},
			expectedErrors: []string{},
		},
		{
			name: "zero stock quantity (valid)",
			product: &Product{
				Name:          "Out of Stock Product",
				PriceKES:      decimal.NewFromFloat(100.50),
				CategoryID:    1,
				StockQuantity: 0,
			},
			expectedErrors: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := validator.New()
			ValidateProduct(v, tt.product)

			// Check that we have the expected number of errors
			if len(v.Errors) != len(tt.expectedErrors) {
				t.Errorf("Expected %d errors, got %d", len(tt.expectedErrors), len(v.Errors))
			}

			// Check that all expected error fields are present
			for _, expectedField := range tt.expectedErrors {
				if _, exists := v.Errors[expectedField]; !exists {
					t.Errorf("Expected error for field '%s', but it was not found", expectedField)
				}
			}

			// Check that no unexpected errors are present
			for field := range v.Errors {
				found := false
				for _, expectedField := range tt.expectedErrors {
					if field == expectedField {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Unexpected error for field '%s': %s", field, v.Errors[field])
				}
			}
		})
	}
}

func TestValidateProductEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		product *Product
		valid   bool
	}{
		{
			name: "very long product name",
			product: &Product{
				Name:          "This is a very long product name that might be too long for some systems to handle properly",
				PriceKES:      decimal.NewFromFloat(100.50),
				CategoryID:    1,
				StockQuantity: 50,
			},
			valid: true, // Should be valid unless we add length validation
		},
		{
			name: "very large price",
			product: &Product{
				Name:          "Expensive Product",
				PriceKES:      decimal.NewFromFloat(999999.99),
				CategoryID:    1,
				StockQuantity: 1,
			},
			valid: true,
		},
		{
			name: "very large stock quantity",
			product: &Product{
				Name:          "High Volume Product",
				PriceKES:      decimal.NewFromFloat(10.00),
				CategoryID:    1,
				StockQuantity: 2147483647, // max int32
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := validator.New()
			ValidateProduct(v, tt.product)

			isValid := v.Valid()
			if isValid != tt.valid {
				t.Errorf("Expected valid=%v, got valid=%v, errors: %v", tt.valid, isValid, v.Errors)
			}
		})
	}
}
