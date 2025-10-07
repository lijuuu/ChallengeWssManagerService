package service

import (
	"context"
	"log"

	"github.com/lijuuu/ChallengeWssManagerService/internal/constants"
	"github.com/lijuuu/ChallengeWssManagerService/internal/model"
	challengePb "github.com/lijuuu/GlobalProtoXcode/ChallengeService"
)

func (s *ChallengeService) GetOwnersActiveChallenges(ctx context.Context, req *challengePb.GetOwnersActiveChallengesRequest) (*challengePb.ChallengeListResponse, error) {
	log.Printf("[GetOwnersActiveChallenges] Fetching active challenges for user %v", req)
	challengeIDs, err := s.GlobalState.Redis.GetActiveChallenges(ctx)
	if err != nil {
		log.Printf("[GetOwnersActiveChallenges] Error fetching active challenges: %v", err)
		return nil, err
	}
	log.Printf("[GetOwnersActiveChallenges] Retrieved %d active challenge IDs", len(challengeIDs))
	var challenges []model.ChallengeDocument
	for _, id := range challengeIDs {
		challenge, err := s.GlobalState.Redis.GetChallengeByID(ctx, id)
		if err != nil {
			log.Printf("[GetOwnersActiveChallenges] Skipping challenge %s due to fetch error: %v", id, err)
			continue
		}
		// Owner can always see; participants can see to allow reconnects
		if (challenge.CreatorID == req.UserId || challenge.Participants[req.UserId] != nil) &&
			(challenge.Status == constants.CHALLENGE_OPEN || challenge.Status == constants.CHALLENGE_STARTED) {
			log.Printf("[GetOwnersActiveChallenges] Challenge %s visible to user %s", id, req.UserId)
			challenges = append(challenges, challenge)
		}
	}
	return &challengePb.ChallengeListResponse{Challenges: ChallengesToProto(ToPtrSlice(challenges), false)}, nil
}
