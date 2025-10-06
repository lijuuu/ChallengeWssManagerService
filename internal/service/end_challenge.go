package service

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/lijuuu/ChallengeWssManagerService/internal/model"
)

func (s *ChallengeService) EndChallenge(ctx context.Context, challengeID, creatorID string) error {
	// Fetch the challenge to verify the creator using Redis repository
	challenge, err := s.GlobalState.Redis.GetChallengeByID(ctx, challengeID)
	if err != nil {
		return fmt.Errorf("challenge not found: %w", err)
	}
	if challenge.CreatorID != creatorID {
		return errors.New("only the creator can end the challenge")
	}
	if err := s.GlobalState.LeaderboardManager.CleanupLeaderboard(challengeID); err != nil {
		log.Printf("[EndChallenge] Warning: Failed to cleanup leaderboard for challenge %s: %v", challengeID, err)
	}
	if err := s.updateChallengeStatus(ctx, challengeID, model.ChallengeEnded); err != nil {
		return fmt.Errorf("failed to end challenge: %w", err)
	}
	return nil
}
