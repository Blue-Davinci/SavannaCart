package main

import (
	"net/http"

	"github.com/Blue-Davinci/SavannaCart/internal/data"
	"github.com/Blue-Davinci/SavannaCart/internal/validator"
	"github.com/shopspring/decimal"
)

func (app *application) getAllProductsHandler(w http.ResponseWriter, r *http.Request) {
	// make a struct to hold what we would want from the queries
	var input struct {
		Name string
		data.Filters
	}
	v := validator.New()
	// Call r.URL.Query() to get the url.Values map containing the query string data.
	qs := r.URL.Query()
	// get our parameters
	input.Name = app.readString(qs, "name", "")
	//get the page & pagesizes as ints and set to the embedded struct
	input.Filters.Page = app.readInt(qs, "page", 1, v)
	input.Filters.PageSize = app.readInt(qs, "page_size", 20, v)
	// We don't use any sort for this endpoint
	input.Filters.Sort = app.readString(qs, "", "")
	// None of the sort values are supported for this endpoint
	input.Filters.SortSafelist = []string{"", ""}
	// Perform validation
	if data.ValidateFilters(v, input.Filters); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}
	// now get our actual products
	products, metadata, err := app.models.Products.GetAllProducts(input.Name, input.Filters)
	if err != nil {
		switch {
		case err == data.ErrGeneralRecordNotFound:
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}
	// Return the products as a JSON response.
	err = app.writeJSON(w, http.StatusOK, envelope{"products": products, "metadata": metadata}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

}

// createNewProductsHandler handles the request to create a new product.
// It expects a JSON body with the product details, validates the input,
// and creates the product in the database. If successful, it returns the created product as a JSON response.
// If there are validation errors, it returns a 422 Unprocessable Entity response with the validation errors.
func (app *application) createNewProductsHandler(w http.ResponseWriter, r *http.Request) {
	// we expect  a couple of values
	var input struct {
		Name          string          `json:"name"`
		PriceKES      decimal.Decimal `json:"price_kes"`
		CategoryID    int32           `json:"category_id"`
		Description   string          `json:"description,omitempty"`
		StockQuantity int32           `json:"stock_quantity"`
	}
	// read our json
	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}
	// create a new product
	product := &data.Product{
		Name:          input.Name,
		PriceKES:      input.PriceKES,
		CategoryID:    input.CategoryID,
		Description:   input.Description,
		StockQuantity: input.StockQuantity,
	}
	// validate the input
	v := validator.New()
	if data.ValidateProduct(v, product); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}
	// create our product in the database
	err = app.models.Products.CreateNewProducts(product)
	if err != nil {
		switch {
		case err == data.ErrDuplicateProductName:
			v.AddError("name", "a product with this name already exists")
			app.failedValidationResponse(w, r, v.Errors)
		case err == data.ErrInvalidCategoryID:
			v.AddError("category_id", "the provided category ID is invalid")
			app.failedValidationResponse(w, r, v.Errors)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}
	// Return the created product as a JSON response.
	err = app.writeJSON(w, http.StatusCreated, envelope{"product": product}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

}
