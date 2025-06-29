package repo

import (
	"gorm.io/gorm"
)

type PSQLRepository struct {
	db *gorm.DB
}

func NewPSQLRepository(db *gorm.DB) *PSQLRepository {
	return &PSQLRepository{db: db}
}