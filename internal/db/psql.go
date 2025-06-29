package db

import (
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	configs "github.com/lijuuu/ChallengeWssManagerService/internal/config"
)

func InitDB(cfg *configs.Config) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(cfg.PsqlURL), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to open db: %w", err)
	}
	return db, nil
}
