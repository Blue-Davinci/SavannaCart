package data

import (
	"time"

	"github.com/Blue-Davinci/SavannaCart/internal/database"
)

// Define the TokenModel type.
type CategoryModel struct {
	DB *database.Queries
}

type Category struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	ParentId  int64     `json:"parent_id"`
	Version   int64     `json:"version"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Timeout constants for our module
const (
	DefaultCategoryDBContextTimeout = 5 * time.Second
)
