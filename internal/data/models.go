package data

import "github.com/Blue-Davinci/SavannaCart/internal/database"

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
