package service

import (
	"context"

	"github.com/lijuuu/ChallengeWssManagerService/internal/model"
	challengePb "github.com/lijuuu/GlobalProtoXcode/ChallengeService"
)

func (s *ChallengeService) GetActiveOpenChallenges(ctx context.Context, req *challengePb.PaginationRequest) (*challengePb.GetActiveOpenChallengesResponse, error) {
	challengeIDs, err := s.GlobalState.Redis.GetChallengesByStatus(ctx, []string{model.ChallengeOpen})
	if err != nil {
		return nil, err
	}
	isPrivate := false
	challenges := s.fetchChallengesConcurrently(ctx, challengeIDs, isPrivate)
	return &challengePb.GetActiveOpenChallengesResponse{List: &challengePb.ChallengeListResponse{
		Challenges: ChallengesToProto(ToPtrSlice(challenges), true),
		TotalCount: int64(len(challenges)),
	}}, nil
}
