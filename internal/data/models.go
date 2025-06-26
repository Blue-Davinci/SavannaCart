package data

import (
	"errors"

	"github.com/Blue-Davinci/SavannaCart/internal/database"
)

var (
	ErrGeneralRecordNotFound = errors.New("record not found")
	ErrEditConflict          = errors.New("edit conflict")
)

type Models struct {
	Users       UserModel
	Tokens      TokenModel
	Permissions PermissionModel
	Categories  CategoryModel
	Products    ProductModel
	Orders      OrderModel
}

func NewModels(db *database.Queries) Models {
	return Models{
		Users:       UserModel{DB: db},
		Tokens:      TokenModel{DB: db},
		Permissions: PermissionModel{DB: db},
		Categories:  CategoryModel{DB: db},
		Products:    ProductModel{DB: db},
		Orders:      OrderModel{DB: db},
	}
}
