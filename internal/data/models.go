package data

import (
	"database/sql"
)

type Models struct {
	ImageJobs   ImageJobModel
	Images      ImageModel
	Consumers   ConsumerModel
	Reports     ReportModel
	ReportsJobs ReportJobModel
}

func NewModels(db *sql.DB) Models {
	return Models{
		ImageJobs:   ImageJobModel{DB: db},
		Images:      ImageModel{DB: db},
		Reports:     ReportModel{DB: db},
		ReportsJobs: ReportJobModel{DB: db},
		Consumers:   ConsumerModel{DB: db},
	}
}
