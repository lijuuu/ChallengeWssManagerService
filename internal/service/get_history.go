package service

import (
	"context"

	challengePb "github.com/lijuuu/GlobalProtoXcode/ChallengeService"
)

func (s *ChallengeService) GetChallengeHistory(ctx context.Context, req *challengePb.GetChallengeHistoryRequest) (*challengePb.GetChallengeHistoryResponse, error) {
	page := int(req.GetPagination().GetPage())
	pageSize := int(req.GetPagination().GetPageSize())
	challenges, err := s.GlobalState.Mongo.GetChallengeHistory(ctx, req.UserId, page, pageSize, req.GetIsPrivate())
	if err != nil {
		return &challengePb.GetChallengeHistoryResponse{
			Success:   false,
			ErrorType: "INTERNAL_ERROR",
			Message:   "failed to fetch history",
		}, nil
	}
	resp := &challengePb.GetChallengeHistoryResponse{
		Success: true,
		Message: "Success",
		List: &challengePb.ChallengeListResponse{
			Challenges: ChallengesToProto(ToPtrSlice(challenges), false),
			// TotalCount: total,
		},
	}
	return resp, nil
}