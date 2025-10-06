package service

import (
	"context"
	"fmt"
	"log"

	"github.com/lijuuu/ChallengeWssManagerService/internal/wss/broadcasts"
	challengePb "github.com/lijuuu/GlobalProtoXcode/ChallengeService"
)

func (s *ChallengeService) AbandonChallenge(ctx context.Context, req *challengePb.AbandonChallengeRequest) (*challengePb.AbandonChallengeResponse, error) {
	// Fetch the challenge to verify the creator using Redis repository
	challenge, err := s.GlobalState.Redis.GetChallengeByID(ctx, req.ChallengeId)
	if err != nil {
		return &challengePb.AbandonChallengeResponse{Success: false, Message: "Challenge not found", ErrorType: "CHALLENGENOTFOUND"}, err
	}

	if challenge.CreatorID != req.CreatorId {
		return &challengePb.AbandonChallengeResponse{Success: false, Message: "Only the creator can abandon the challenge", ErrorType: "NOTCREATOR"}, nil
	}

	if err := s.GlobalState.Redis.AbandonChallenge(ctx, req.CreatorId, req.ChallengeId); err != nil {
		return &challengePb.AbandonChallengeResponse{Success: false, Message: err.Error(), ErrorType: "CHALLENGEABANDONFAILED"}, err
	}

	// Clean up leaderboard for abandoned challenge
	if err := s.GlobalState.LeaderboardManager.CleanupLeaderboard(req.ChallengeId); err != nil {
		log.Printf("[AbandonChallenge] Warning: Failed to cleanup leaderboard for challenge %s: %v", req.ChallengeId, err)
	}

	// Trigger MongoDB persistence for ABANDONED challenge
	if err := s.persistChallengeToMongoDB(ctx, req.ChallengeId); err != nil {
		fmt.Printf("Warning: Failed to persist abandoned challenge %s to MongoDB: %v\n", req.ChallengeId, err)
	}

	// Broadcast the abandon event using the new method with clients
	if s.GlobalState != nil && s.GlobalState.LocalState != nil {
		wsClients := s.GlobalState.LocalState.GetAllWSClients(challenge.ChallengeID)
		if len(wsClients) > 0 {
			broadcasts.BroadcastChallengeAbandonWithClients(wsClients, challenge.ChallengeID, challenge.CreatorID)
		}
	}

	return &challengePb.AbandonChallengeResponse{Success: true}, nil
}
