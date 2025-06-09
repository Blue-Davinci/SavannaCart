package data

import "github.com/Blue-Davinci/SavannaCart/internal/database"

type Models struct {
	Users UserModel
}

func NewModels(db *database.Queries) Models {
	return Models{
		Users: UserModel{DB: db},
	}
}
