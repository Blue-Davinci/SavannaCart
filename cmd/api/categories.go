package main

import (
	"errors"
	"net/http"

	"github.com/Blue-Davinci/SavannaCart/internal/data"
	"github.com/Blue-Davinci/SavannaCart/internal/validator"
	"go.uber.org/zap"
)

// getAllCategoriesHandler handles the request to get all categories.
// It should retrieve all categories from the database and return them as a JSON response.
// we however also allow pagination as well as searches by the category's name
func (app *application) getAllCategoriesHandler(w http.ResponseWriter, r *http.Request) {
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
	// now get our actual categories
	categories, metadata, err := app.models.Categories.GetAllCategories(input.Name, input.Filters)
	if err != nil {
		switch {
		case err == data.ErrGeneralRecordNotFound:
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}
	// Return the categories as a JSON response.
	err = app.writeJSON(w, http.StatusOK, envelope{"categories": categories, "metadata": metadata}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}

}

// createNewCategoryHandler() handles the request to create a new category.
func (app *application) createNewCategoryHandler(w http.ResponseWriter, r *http.Request) {
	// we expect a name and optional parent_id
	var input struct {
		Name     string `json:"name"`
		ParentId int32  `json:"parent_id,omitempty"`
	}
	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}
	// create a category
	category := &data.Category{
		Name:     input.Name,
		ParentId: input.ParentId,
	}
	// validate the category
	v := validator.New()
	if data.ValidateName(v, input.Name, "name"); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}
	// create our category in the database
	err = app.models.Categories.CreateNewCategory(category)
	if err != nil {
		switch {
		case err == data.ErrDuplicateCategoryName:
			v.AddError("name", "a category with this name already exists")
			app.failedValidationResponse(w, r, v.Errors)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}
	// Return the created category as a JSON response.
	err = app.writeJSON(w, http.StatusCreated, envelope{"category": category}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}

}

// deleteCategoryByIDHandler handles the request to delete a category by its ID.
// It expects the category ID to be provided in the URL as a path parameter.
func (app *application) deleteCategoryByIDHandler(w http.ResponseWriter, r *http.Request) {
	// get id from url
	categoryID, err := app.readIDParam(r, "categoryID")
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}
	// validate the category ID
	v := validator.New()
	if data.ValidateURLID(v, categoryID, "categoryID"); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}
	// delete the category from the database
	err = app.models.Categories.DeleteCategoryByID(int32(categoryID))
	if err != nil {
		switch {
		case errors.Is(err, data.ErrGeneralRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}
	// Return a 204 No Content response with a success message.
	err = app.writeJSON(w, http.StatusNoContent, envelope{"message": "category deleted successfully"}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

}

// updateCategoryHandler handles the request to update a category.
// It expects the category ID and version ID to be provided in the URL as path parameters.
// It also expects the updated category data to be provided in the request body as JSON.
// It validates the input and updates the category in the database.
// If the category does not exist, it returns a 404 Not Found response.
func (app *application) updateCategoryHandler(w http.ResponseWriter, r *http.Request) {
	// get the feed ID from the URL
	categoryID, err := app.readIDParam(r, "categoryID")
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}
	// get versionID from the URL
	versionID, err := app.readIDParam(r, "versionID")
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}
	v := validator.New()
	// validate the feed ID
	if data.ValidateURLID(v, categoryID, "categoryID"); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}
	// validate the version ID
	if data.ValidateURLID(v, versionID, "versionID"); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}
	// make a category struct to hold the data
	var input struct {
		Name     *string `json:"name"`
		ParentId *int32  `json:"parent_id,omitempty"`
	}
	// read the JSON from the request body
	err = app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}
	app.logger.Debug("updateCategoryHandler", zap.Int32("categoryID", int32(categoryID)), zap.Int32("versionID", int32(versionID)), zap.Any("input", input))
	// check if the exact category exists
	category, err := app.models.Categories.GetCategoryByID(int32(categoryID), int32(versionID))
	if err != nil {
		switch {
		case errors.Is(err, data.ErrGeneralRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}
	// check to see which fields we want to update
	if input.Name != nil {
		category.Name = *input.Name
	}
	if input.ParentId != nil {
		category.ParentId = *input.ParentId
	}

	// validate the input
	if data.ValidateUpdatedCategory(v, category); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}
	// update the category in the database
	err = app.models.Categories.UpdateCategory(category)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrDuplicateCategoryName):
			v.AddError("name", "a category with this name already exists")
			app.failedValidationResponse(w, r, v.Errors)
		case errors.Is(err, data.ErrGeneralRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}
	// Return the updated category as a JSON response.
	err = app.writeJSON(w, http.StatusOK, envelope{"category": category}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}

}

// getCategoryAveragePriceHandler handles the request to get the average price of products in a category and its children.
// It expects the category ID to be provided in the URL as a path parameter.
func (app *application) getCategoryAveragePriceHandler(w http.ResponseWriter, r *http.Request) {
	// get id from url
	categoryID, err := app.readIDParam(r, "categoryID")
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}
	// validate the category ID
	v := validator.New()
	if data.ValidateURLID(v, categoryID, "categoryID"); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}
	// get the average price from the database
	categoryAverage, err := app.models.Categories.GetCategoryAveragePrice(int32(categoryID))
	if err != nil {
		switch {
		case errors.Is(err, data.ErrGeneralRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}
	// Return the category average price as a JSON response.
	err = app.writeJSON(w, http.StatusOK, envelope{"category_average": categoryAverage}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
