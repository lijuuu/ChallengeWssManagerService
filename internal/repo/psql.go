package repo

import (
	"context"
	"errors"

	model "github.com/lijuuu/ChallengeWssManagerService/internal/model"
	"gorm.io/gorm"
)

type PSQLRepository struct {
	db *gorm.DB
}

func NewPSQLRepository(db *gorm.DB) *PSQLRepository {
	return &PSQLRepository{db: db}
}

// CreateChallenge inserts a minimal snapshot
func (r *PSQLRepository) CreateChallenge(ctx context.Context, c *model.Challenge) error {
	if c.ChallengeID == "" {
		return errors.New("challenge ID cannot be empty")
	}
	return r.db.WithContext(ctx).Create(&c).Error
}

// GetPublicChallenges returns open, public challenges paginated
func (r *PSQLRepository) GetPublicChallenges(ctx context.Context, page, pageSize int) ([]model.Challenge, error) {
	if page < 1 || pageSize < 1 {
		return nil, errors.New("page and pageSize must be positive integers")
	}
	var out []model.Challenge
	offset := (page - 1) * pageSize

	err := r.db.WithContext(ctx).
		Where("is_private = ?", false).
		Order("start_time DESC").
		Limit(pageSize).
		Offset(offset).
		Find(&out).Error

	if err != nil {
		return nil, err
	}
	return out, nil
}

// GetPrivateChallengesForUser returns private challenges where user is creator or participant
func (r *PSQLRepository) GetPrivateChallengesForUser(ctx context.Context, userID string, page, pageSize int) ([]model.Challenge, error) {
	if page < 1 || pageSize < 1 {
		return nil, errors.New("page and pageSize must be positive integers")
	}
	if userID == "" {
		return nil, errors.New("userID cannot be empty")
	}
	var out []model.Challenge
	offset := (page - 1) * pageSize

	err := r.db.WithContext(ctx).
		Where("is_private = ? AND (creator_id = ? OR id IN (SELECT challenge_id FROM challenge_participants WHERE user_id = ?))", true, userID, userID).
		Order("start_time DESC").
		Limit(pageSize).
		Offset(offset).
		Find(&out).Error

	if err != nil {
		return nil, err
	}
	return out, nil
}

// GetActiveChallenges returns all not-ended challenges
func (r *PSQLRepository) GetActiveChallenges(ctx context.Context, page, pageSize int) ([]model.Challenge, error) {
	if page < 1 || pageSize < 1 {
		return nil, errors.New("page and pageSize must be positive integers")
	}
	var out []model.Challenge
	offset := (page - 1) * pageSize

	err := r.db.WithContext(ctx).
		Where("status != ?", "ended").
		Order("start_time ASC").
		Limit(pageSize).
		Offset(offset).
		Find(&out).Error

	if err != nil {
		return nil, err
	}
	return out, nil
}

// GetUserChallenges returns all challenges created by a specific user
func (r *PSQLRepository) GetUserChallenges(ctx context.Context, userID string, page, pageSize int) ([]model.Challenge, error) {
	if page < 1 || pageSize < 1 {
		return nil, errors.New("page and pageSize must be positive integers")
	}
	if userID == "" {
		return nil, errors.New("userID cannot be empty")
	}
	var out []model.Challenge
	offset := (page - 1) * pageSize

	err := r.db.WithContext(ctx).
		Where("creator_id = ?", userID).
		Order("created_at DESC").
		Limit(pageSize).
		Offset(offset).
		Find(&out).Error

	if err != nil {
		return nil, err
	}
	return out, nil
}
