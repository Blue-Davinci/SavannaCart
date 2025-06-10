package data

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/Blue-Davinci/SavannaCart/internal/database"
	"github.com/Blue-Davinci/SavannaCart/internal/validator"
	"github.com/shopspring/decimal"
)

var (
	ErrDuplicateProductName = errors.New("product with this name already exists, please choose a different name")
	ErrInvalidCategoryID    = errors.New("invalid category ID provided")
)

// Define the TokenModel type.
type ProductModel struct {
	DB *database.Queries
}

type Product struct {
	ID            int32           `json:"id"`
	Name          string          `json:"name"`
	PriceKES      decimal.Decimal `json:"price_kes"`
	CategoryID    int32           `json:"category_id"`
	Category      *CategoryInfo   `json:"category"` // Category details
	Description   string          `json:"description"`
	StockQuantity int32           `json:"stock_quantity"`
	StockStatus   string          `json:"stock_status"` // "in_stock", "low_stock", "out_of_stock"
	Version       int32           `json:"version"`
	CreatedAt     string          `json:"created_at"`
	UpdatedAt     string          `json:"updated_at"`
}
type CategoryInfo struct {
	ID       int32  `json:"id"`                  // Category's own ID
	Name     string `json:"name"`                // Category's name
	ParentID *int32 `json:"parent_id,omitempty"` // Category's parent ID (can be null for root)
}

// Timeout constants for our module
const (
	DefaultProductDBContextTimeout = 5 * time.Second
)

// Stock status constants
const (
	StockStatusInStock    = "in_stock"
	StockStatusLowStock   = "low_stock"
	StockStatusOutOfStock = "out_of_stock"
	LowStockThreshold     = 10 // Below this is considered low stock
)

// generateStockStatus determines the stock status based on quantity
func generateStockStatus(quantity int32) string {
	switch {
	case quantity == 0:
		return StockStatusOutOfStock
	case quantity <= LowStockThreshold:
		return StockStatusLowStock
	default:
		return StockStatusInStock
	}
}

func ValidateProduct(v *validator.Validator, product *Product) {
	// Validate the product name
	v.Check(product.Name != "", "name", "must be provided")
	// Validate the price
	v.Check(product.PriceKES.GreaterThanOrEqual(decimal.NewFromInt(0)), "price_kes", "must be greater than or equal to 0")
	// Validate the category ID
	v.Check(product.CategoryID > 0, "category_id", "must be a valid ID")
	// Validate the stock quantity
	v.Check(product.StockQuantity >= 0, "stock_quantity", "must be greater than or equal to 0")
}

// GetAllProducts() is a method that retrieves all products from the database.
// It takes a category name and filters as parameters and returns a slice of Product pointers,
// metadata for pagination, and an error if any.
func (m ProductModel) GetAllProducts(name string, filters Filters) ([]*Product, Metadata, error) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultProductDBContextTimeout)
	defer cancel()
	// Get all products from the database
	products, err := m.DB.GetAllProductsWithCategory(ctx, database.GetAllProductsWithCategoryParams{
		Column1: name,
		Limit:   int32(filters.limit()),
		Offset:  int32(filters.offset()),
	})
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, Metadata{}, ErrGeneralRecordNotFound // No products found
		default:
			return nil, Metadata{}, err // Some other error occurred
		}
	}
	// check length of products
	if len(products) == 0 {
		return nil, Metadata{}, ErrGeneralRecordNotFound // No products found
	}
	// Populate the products slice with the data from the database rows
	populatedProducts := []*Product{}
	totalProducts := 0
	for _, productRow := range products {
		totalProducts = int(productRow.TotalCount)
		populatedProducts = append(populatedProducts, populateProducts(productRow))
	}
	// Create metadata for pagination
	metadata := calculateMetadata(totalProducts, filters.Page, filters.PageSize)
	return populatedProducts, metadata, nil
}

// CreateNewProducts() is a method that creates a new product in the database.
// It takes a pointer to a Product struct and returns an error if any.
func (m ProductModel) CreateNewProducts(product *Product) error {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultCategoryDBContextTimeout)
	defer cancel()

	// CREATE new category in the database
	newProduct, err := m.DB.CreateNewProducts(ctx, database.CreateNewProductsParams{
		Name:          product.Name,
		PriceKes:      product.PriceKES.String(),
		CategoryID:    product.CategoryID,
		Description:   sql.NullString{String: product.Description, Valid: true},
		StockQuantity: product.StockQuantity,
	})
	if err != nil {
		switch {
		case strings.Contains(err.Error(), "ux_products_name_cat"):
			return ErrDuplicateProductName // Fix: should be product, not category
		case strings.Contains(err.Error(), "products_category_id_fkey"):
			return ErrInvalidCategoryID // New: handle invalid category
		case strings.Contains(err.Error(), "duplicate key"):
			return ErrDuplicateProductName
		default:
			return err
		}
	} // fill in the ID, CreatedAt, and UpdatedAt fields
	product.ID = newProduct.ID
	product.Version = newProduct.Version
	product.StockStatus = generateStockStatus(product.StockQuantity) // Generate stock status
	product.CreatedAt = newProduct.CreatedAt.Format(time.RFC3339)
	product.UpdatedAt = newProduct.UpdatedAt.Format(time.RFC3339)
	// return nil to indicate success
	return nil
}

// populateProducts converts a database row into a Product struct.
func populateProducts(productRow any) *Product {
	switch product := productRow.(type) {
	case database.GetAllProductsWithCategoryRow:
		// Handle nullable parent_id properly

		return &Product{
			ID:         product.ID,
			Name:       product.Name,
			PriceKES:   decimal.RequireFromString(product.PriceKes),
			CategoryID: product.CategoryID,
			Category: &CategoryInfo{
				ID:       product.CategoryIDInfo.Int32, // Category's own ID
				Name:     product.CategoryName.String,
				ParentID: &product.CategoryParentID.Int32,
			},
			Description:   product.Description.String,
			StockQuantity: product.StockQuantity,
			StockStatus:   generateStockStatus(product.StockQuantity), // Generate stock status
			Version:       product.Version,
			CreatedAt:     product.CreatedAt.Format(time.RFC3339),
			UpdatedAt:     product.UpdatedAt.Format(time.RFC3339),
		}
	default:
		return nil // Return nil if the type does not match
	}
}
