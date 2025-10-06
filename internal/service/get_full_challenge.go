package service

import (
	"context"

	"github.com/lijuuu/ChallengeWssManagerService/internal/model"
	challengePb "github.com/lijuuu/GlobalProtoXcode/ChallengeService"
)

func (s *ChallengeService) GetFullChallengeData(ctx context.Context, req *challengePb.GetFullChallengeDataRequest) (*challengePb.GetFullChallengeDataResponse, error) {
	// Read from Redis repository only - no MongoDB fallback for active challenge data
	challenge, err := s.GlobalState.Redis.GetChallengeByID(ctx, req.ChallengeId)
	if err != nil {
		return nil, err
	}

	return &challengePb.GetFullChallengeDataResponse{Challenge: ChallengesToProto([]*model.ChallengeDocument{&challenge}, false)[0]}, nil
}
