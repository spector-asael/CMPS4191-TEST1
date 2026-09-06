package data

import (
	"database/sql"
)

type Models struct {
	Jobs   JobModel
	Images ImageModel
}

func NewModels(db *sql.DB) Models {
	return Models{
		Jobs:   JobModel{DB: db},
		Images: ImageModel{DB: db},
	}
}
