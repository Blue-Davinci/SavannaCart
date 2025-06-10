package data

import (
	"context"
	"time"

	"github.com/Blue-Davinci/SavannaCart/internal/database"
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

// Timeout constants for our module
const (
	DefaultCategoryDBContextTimeout = 5 * time.Second
)

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
	default:
		return nil
	}
}
