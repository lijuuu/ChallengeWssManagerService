package service

import (
	"context"

	challengePb "github.com/lijuuu/GlobalProtoXcode/ChallengeService"
)

func (s *ChallengeService) GetChallengeHistory(ctx context.Context, req *challengePb.GetChallengeHistoryRequest) (*challengePb.ChallengeListResponse, error) {
	challenges, err := s.GlobalState.Mongo.GetChallengeHistory(ctx, req.UserId, int(req.GetPagination().GetPage()), int(req.GetPagination().GetPageSize()), req.GetIsPrivate())
	if err != nil {
		return nil, err
	}
	return &challengePb.ChallengeListResponse{Challenges: ChallengesToProto(ToPtrSlice(challenges), false), TotalCount: int64(len(challenges))}, nil
}
