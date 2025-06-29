package service

import (
	"context"
	"fmt"

	challengeService "github.com/lijuuu/GlobalProtoXcode/ChallengeService"
)

type ChallengeService struct {
	challengeService.UnimplementedChallengeServiceServer
}

func NewChallengeService() *ChallengeService {
	return &ChallengeService{}
}

func (s *ChallengeService) PushSubmissionStatus(ctx context.Context, req *challengeService.PushSubmissionStatusRequest) (*challengeService.PushSubmissionStatusResponse, error) {
	fmt.Println(req)
	return &challengeService.PushSubmissionStatusResponse{
		Message: "received",
		Success: true,
	}, nil
}


