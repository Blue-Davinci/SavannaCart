package data

import (
	"testing"

	"github.com/Blue-Davinci/SavannaCart/internal/validator"
)

func TestValidateFilters(t *testing.T) {
	tests := []struct {
		name           string
		filters        Filters
		expectedErrors []string
	}{
		{
			name: "valid filters",
			filters: Filters{
				Page:         1,
				PageSize:     20,
				Sort:         "name",
				SortSafelist: []string{"name", "price", "created_at"},
			},
			expectedErrors: []string{},
		},
		{
			name: "page too small",
			filters: Filters{
				Page:         0,
				PageSize:     20,
				Sort:         "name",
				SortSafelist: []string{"name", "price", "created_at"},
			},
			expectedErrors: []string{"page"},
		},
		{
			name: "page too large",
			filters: Filters{
				Page:         10_000_001,
				PageSize:     20,
				Sort:         "name",
				SortSafelist: []string{"name", "price", "created_at"},
			},
			expectedErrors: []string{"page"},
		},
		{
			name: "page size too small",
			filters: Filters{
				Page:         1,
				PageSize:     0,
				Sort:         "name",
				SortSafelist: []string{"name", "price", "created_at"},
			},
			expectedErrors: []string{"page_size"},
		},
		{
			name: "page size too large",
			filters: Filters{
				Page:         1,
				PageSize:     101,
				Sort:         "name",
				SortSafelist: []string{"name", "price", "created_at"},
			},
			expectedErrors: []string{"page_size"},
		},
		{
			name: "invalid sort field",
			filters: Filters{
				Page:         1,
				PageSize:     20,
				Sort:         "invalid_field",
				SortSafelist: []string{"name", "price", "created_at"},
			},
			expectedErrors: []string{"sort"},
		},
		{
			name: "multiple errors",
			filters: Filters{
				Page:         0,
				PageSize:     101,
				Sort:         "invalid_field",
				SortSafelist: []string{"name", "price", "created_at"},
			},
			expectedErrors: []string{"page", "page_size", "sort"},
		},
		{
			name: "empty sort with empty safelist",
			filters: Filters{
				Page:         1,
				PageSize:     20,
				Sort:         "",
				SortSafelist: []string{},
			},
			expectedErrors: []string{"sort"},
		},
		{
			name: "valid descending sort",
			filters: Filters{
				Page:         1,
				PageSize:     20,
				Sort:         "-name",
				SortSafelist: []string{"name", "-name", "price"},
			},
			expectedErrors: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := validator.New()
			ValidateFilters(v, tt.filters)

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

func TestFiltersSortColumn(t *testing.T) {
	tests := []struct {
		name         string
		sort         string
		sortSafelist []string
		expected     string
	}{
		{
			name:         "simple ascending sort",
			sort:         "name",
			sortSafelist: []string{"name", "price"},
			expected:     "name",
		},
		{
			name:         "descending sort",
			sort:         "-name",
			sortSafelist: []string{"name", "-name"},
			expected:     "name",
		},
		{
			name:         "complex field name",
			sort:         "created_at",
			sortSafelist: []string{"created_at", "updated_at"},
			expected:     "created_at",
		},
		{
			name:         "descending complex field",
			sort:         "-created_at",
			sortSafelist: []string{"created_at", "-created_at"},
			expected:     "created_at",
		}, {
			name:         "empty sort",
			sort:         "",
			sortSafelist: []string{"", "name"},
			expected:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filters := Filters{
				Sort:         tt.sort,
				SortSafelist: tt.sortSafelist,
			}

			result := filters.sortColumn()
			if result != tt.expected {
				t.Errorf("sortColumn() = %s, want %s", result, tt.expected)
			}
		})
	}
}

func TestFiltersSortDirection(t *testing.T) {
	tests := []struct {
		name         string
		sort         string
		sortSafelist []string
		expected     string
	}{
		{
			name:         "ascending sort",
			sort:         "name",
			sortSafelist: []string{"name", "-name"},
			expected:     "ASC",
		},
		{
			name:         "descending sort",
			sort:         "-name",
			sortSafelist: []string{"name", "-name"},
			expected:     "DESC",
		},
		{
			name:         "descending complex field",
			sort:         "-created_at",
			sortSafelist: []string{"created_at", "-created_at"},
			expected:     "DESC",
		},
		{
			name:         "ascending complex field",
			sort:         "created_at",
			sortSafelist: []string{"created_at", "-created_at"},
			expected:     "ASC",
		},
		{
			name:         "empty sort",
			sort:         "",
			sortSafelist: []string{"name"},
			expected:     "ASC",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filters := Filters{
				Sort:         tt.sort,
				SortSafelist: tt.sortSafelist,
			}

			result := filters.sortDirection()
			if result != tt.expected {
				t.Errorf("sortDirection() = %s, want %s", result, tt.expected)
			}
		})
	}
}

func TestCalculateMetadata(t *testing.T) {
	tests := []struct {
		name         string
		totalRecords int
		page         int
		pageSize     int
		expected     Metadata
	}{
		{
			name:         "first page with full results",
			totalRecords: 100,
			page:         1,
			pageSize:     20,
			expected: Metadata{
				CurrentPage:  1,
				PageSize:     20,
				FirstPage:    1,
				LastPage:     5,
				TotalRecords: 100,
			},
		},
		{
			name:         "middle page",
			totalRecords: 100,
			page:         3,
			pageSize:     20,
			expected: Metadata{
				CurrentPage:  3,
				PageSize:     20,
				FirstPage:    1,
				LastPage:     5,
				TotalRecords: 100,
			},
		},
		{
			name:         "last page",
			totalRecords: 100,
			page:         5,
			pageSize:     20,
			expected: Metadata{
				CurrentPage:  5,
				PageSize:     20,
				FirstPage:    1,
				LastPage:     5,
				TotalRecords: 100,
			},
		},
		{
			name:         "partial last page",
			totalRecords: 95,
			page:         5,
			pageSize:     20,
			expected: Metadata{
				CurrentPage:  5,
				PageSize:     20,
				FirstPage:    1,
				LastPage:     5,
				TotalRecords: 95,
			},
		},
		{
			name:         "exactly one page",
			totalRecords: 20,
			page:         1,
			pageSize:     20,
			expected: Metadata{
				CurrentPage:  1,
				PageSize:     20,
				FirstPage:    1,
				LastPage:     1,
				TotalRecords: 20,
			},
		}, {
			name:         "no records",
			totalRecords: 0,
			page:         1,
			pageSize:     20,
			expected: Metadata{
				CurrentPage:  0,
				PageSize:     0,
				FirstPage:    0,
				LastPage:     0,
				TotalRecords: 0,
			},
		},
		{
			name:         "single record",
			totalRecords: 1,
			page:         1,
			pageSize:     20,
			expected: Metadata{
				CurrentPage:  1,
				PageSize:     20,
				FirstPage:    1,
				LastPage:     1,
				TotalRecords: 1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateMetadata(tt.totalRecords, tt.page, tt.pageSize)

			if result.CurrentPage != tt.expected.CurrentPage {
				t.Errorf("CurrentPage = %d, want %d", result.CurrentPage, tt.expected.CurrentPage)
			}
			if result.PageSize != tt.expected.PageSize {
				t.Errorf("PageSize = %d, want %d", result.PageSize, tt.expected.PageSize)
			}
			if result.FirstPage != tt.expected.FirstPage {
				t.Errorf("FirstPage = %d, want %d", result.FirstPage, tt.expected.FirstPage)
			}
			if result.LastPage != tt.expected.LastPage {
				t.Errorf("LastPage = %d, want %d", result.LastPage, tt.expected.LastPage)
			}
			if result.TotalRecords != tt.expected.TotalRecords {
				t.Errorf("TotalRecords = %d, want %d", result.TotalRecords, tt.expected.TotalRecords)
			}
		})
	}
}

func TestFiltersLimit(t *testing.T) {
	tests := []struct {
		name     string
		pageSize int
		expected int
	}{
		{
			name:     "normal page size",
			pageSize: 20,
			expected: 20,
		},
		{
			name:     "small page size",
			pageSize: 5,
			expected: 5,
		},
		{
			name:     "maximum page size",
			pageSize: 100,
			expected: 100,
		},
		{
			name:     "minimum page size",
			pageSize: 1,
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filters := Filters{PageSize: tt.pageSize}
			result := filters.limit()
			if result != tt.expected {
				t.Errorf("limit() = %d, want %d", result, tt.expected)
			}
		})
	}
}

func TestFiltersOffset(t *testing.T) {
	tests := []struct {
		name     string
		page     int
		pageSize int
		expected int
	}{
		{
			name:     "first page",
			page:     1,
			pageSize: 20,
			expected: 0,
		},
		{
			name:     "second page",
			page:     2,
			pageSize: 20,
			expected: 20,
		},
		{
			name:     "third page",
			page:     3,
			pageSize: 20,
			expected: 40,
		},
		{
			name:     "small page size",
			page:     3,
			pageSize: 5,
			expected: 10,
		},
		{
			name:     "large page number",
			page:     100,
			pageSize: 10,
			expected: 990,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filters := Filters{
				Page:     tt.page,
				PageSize: tt.pageSize,
			}
			result := filters.offset()
			if result != tt.expected {
				t.Errorf("offset() = %d, want %d", result, tt.expected)
			}
		})
	}
}
