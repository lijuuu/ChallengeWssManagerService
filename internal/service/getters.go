package service

import (
	"context"
)

func (s *ChallengeService) LeaveChallenge(ctx context.Context, challengeId, userId string) bool {
	challenge, err := s.GlobalState.Redis.GetChallengeByID(ctx, challengeId)
	if err != nil {
		return false
	}
	if challenge.CreatorID != userId {
		return false
	}
	if err := s.GlobalState.Redis.RemoveParticipantInJoinPhase(ctx, challengeId, userId); err != nil {
		return false
	}
	return true
}
