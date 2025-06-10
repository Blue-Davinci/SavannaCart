package data

import (
	"errors"

	"github.com/Blue-Davinci/SavannaCart/internal/database"
)

var (
	ErrGeneralRecordNotFound = errors.New("record not found")
)

type Models struct {
	Users  UserModel
	Tokens TokenModel
}

func NewModels(db *database.Queries) Models {
	return Models{
		Users:  UserModel{DB: db},
		Tokens: TokenModel{DB: db},
	}
}
