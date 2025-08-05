package data

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Blue-Davinci/SavannaCart/internal/database"
	"github.com/Blue-Davinci/SavannaCart/internal/validator"
	"github.com/shopspring/decimal"
)

var (
	ErrDuplicateCategoryName = errors.New("category with this name already exists, please choose a different name")
)

// Define the TokenModel type.
type CategoryModel struct {
	DB *database.Queries
}

type Category struct {
	ID        int32     `json:"id"`
	Name      string    `json:"name"`
	ParentId  int32     `json:"parent_id"`
	Version   int32     `json:"version"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CategoryAveragePrice struct {
	CategoryID   int32           `json:"category_id"`
	AveragePrice decimal.Decimal `json:"average_price"`
	ProductCount int32           `json:"product_count"`
	Currency     string          `json:"currency"`
}

// Timeout constants for our module
const (
	DefaultCategoryDBContextTimeout = 10 * time.Second
)

func ValidateURLID(v *validator.Validator, stockID int64, fieldName string) {
	v.Check(stockID > 0, fieldName, "must be a valid ID")
}

func ValidateUpdatedCategory(v *validator.Validator, category *Category) {
	// Validate the category name
	v.Check(category.Name != "", "name", "must be provided")
	// Validate the version
	v.Check(category.Version > 0, "version", "must be greater than 0")
}

func (m CategoryModel) GetCategoryByID(categoryID, categoryVersion int32) (*Category, error) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultCategoryDBContextTimeout)
	defer cancel()

	categoryRow, err := m.DB.GetCategoryById(ctx, database.GetCategoryByIdParams{
		ID:      categoryID,
		Version: categoryVersion,
	})
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrGeneralRecordNotFound
		default:
			return nil, err
		}
	}
	// Populate the Category struct with the data from the database row
	populatedCategory := populateCategories(categoryRow)

	return populatedCategory, nil
}

// GetAllCategories() retrieves all categories from the database.
// It accepts a name filter and pagination filters, returning a slice of Category pointers,
// metadata for pagination, and an error if any occurs.
// If no categories are found, it returns ErrGeneralRecordNotFound.
func (m CategoryModel) GetAllCategories(name string, filters Filters) ([]*Category, Metadata, error) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultCategoryDBContextTimeout)
	defer cancel()

	categoryRows, err := m.DB.GetAllCategories(ctx, database.GetAllCategoriesParams{
		Column1: name,
		Limit:   int32(filters.limit()),
		Offset:  int32(filters.offset()),
	})
	if err != nil {
		return nil, Metadata{}, err
	}
	//  check if there are no posts
	if len(categoryRows) == 0 {
		return nil, Metadata{}, ErrGeneralRecordNotFound
	}
	categories := []*Category{}
	totalCategories := 0
	for _, categoryRow := range categoryRows {
		totalCategories = int(categoryRow.TotalCount)
		categories = append(categories, populateCategories(categoryRow))

	}
	// make metadata struct
	metadata := calculateMetadata(totalCategories, filters.Page, filters.PageSize)

	return categories, metadata, nil
}

func (m CategoryModel) CreateNewCategory(category *Category) error {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultCategoryDBContextTimeout)
	defer cancel()

	// Convert 0 or invalid parent_id to NULL for root categories
	parentID := convertValueToNullInt32(category.ParentId)

	// Call the database query to create a new category
	createdCategory, err := m.DB.CreateCategory(ctx, database.CreateCategoryParams{
		Name: category.Name,
		// use nullable int32 for ParentID
		ParentID: parentID,
	})
	if err != nil {
		switch {
		case strings.Contains(err.Error(), "ux_categories_name_parent"):
			return ErrDuplicateCategoryName
		case strings.Contains(err.Error(), "duplicate key"):
			return ErrDuplicateCategoryName
		default:
			return err
		}
	}
	// fill the category struct with the created category data
	category.ID = createdCategory.ID
	category.Version = createdCategory.Version
	category.CreatedAt = createdCategory.CreatedAt
	category.UpdatedAt = createdCategory.UpdatedAt
	// If the category was created successfully, return nil error
	return nil

}

// UpdateCategory() updates an existing category in the database.
// It accepts a pointer to a Category struct, which contains the updated data.
// It returns an error if the update fails, including specific errors for duplicate names.
// If the update is successful, it updates the category struct with the new data.
func (m CategoryModel) UpdateCategory(category *Category) error {
	ctx, cancel := contextGenerator(context.Background(), DefaultCategoryDBContextTimeout)
	defer cancel()
	// Convert 0 or invalid parent_id to NULL for root categories
	parentID := convertValueToNullInt32(category.ParentId)
	// Call the database query to update the category
	updatedCategory, err := m.DB.UpdateCategory(ctx, database.UpdateCategoryParams{
		ID:       category.ID,
		Name:     category.Name,
		ParentID: parentID,
		Version:  category.Version,
	})
	if err != nil {
		switch {
		case strings.Contains(err.Error(), "ux_categories_name_parent"):
			return ErrDuplicateCategoryName
		case strings.Contains(err.Error(), "duplicate key"):
			return ErrDuplicateCategoryName
		default:
			return err
		}
	}
	// fill the category struct with the updated category data
	category.ParentId = updatedCategory.ParentID.Int32
	category.Version = updatedCategory.Version
	category.UpdatedAt = updatedCategory.UpdatedAt
	// If the category was updated successfully, return nil error
	return nil

}

func (m CategoryModel) DeleteCategoryByID(categoryID int32) error {
	ctx, cancel := contextGenerator(context.Background(), DefaultCategoryDBContextTimeout)
	defer cancel()
	// delete
	_, err := m.DB.DeleteCategory(ctx, categoryID)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return ErrGeneralRecordNotFound
		default:
			return err
		}
	}
	// done
	return nil
}

// GetCategoryAveragePrice calculates the average price of all products in a category and its children.
// It takes a category ID and returns a CategoryAveragePrice struct with the results and any error that occurs.
func (m CategoryModel) GetCategoryAveragePrice(categoryID int32) (*CategoryAveragePrice, error) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultCategoryDBContextTimeout)
	defer cancel()
	fmt.Println("Getting category average price for category ID:", categoryID)
	result, err := m.DB.GetCategoryAveragePrice(ctx, categoryID)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrGeneralRecordNotFound
		default:
			fmt.Println("Error getting category average price:", err)
			return nil, err
		}
	}
	// Convert the average price from string to decimal.Decimal
	// PostgreSQL NUMERIC is always returned as string
	averagePrice, err := decimal.NewFromString(result.AveragePrice)
	if err != nil {
		return nil, err
	}

	// Convert product count (it's already int64, just convert to int32)
	productCount := int32(result.ProductCount)

	categoryAverage := &CategoryAveragePrice{
		CategoryID:   categoryID,
		AveragePrice: averagePrice,
		ProductCount: productCount,
		Currency:     "KES",
	}

	return categoryAverage, nil
}

func populateCategories(categoryRow any) *Category {
	switch category := categoryRow.(type) {
	case database.GetAllCategoriesRow:
		return &Category{
			ID:        category.ID,
			Name:      category.Name,
			ParentId:  category.ParentID.Int32,
			Version:   category.Version,
			CreatedAt: category.CreatedAt,
			UpdatedAt: category.UpdatedAt,
		}
	case database.Category:
		return &Category{
			ID:        category.ID,
			Name:      category.Name,
			ParentId:  category.ParentID.Int32,
			Version:   category.Version,
			CreatedAt: category.CreatedAt,
			UpdatedAt: category.UpdatedAt,
		}
	default:
		return nil
	}
}
