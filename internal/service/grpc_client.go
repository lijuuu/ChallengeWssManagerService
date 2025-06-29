package service

import (
	"context"
	"fmt"

	"github.com/lijuuu/ChallengeWssManagerService/internal/repo"
	challengeService "github.com/lijuuu/GlobalProtoXcode/ChallengeService"
)

type ChallengeService struct {
	repo *repo.PSQLRepository
	challengeService.UnimplementedChallengeServiceServer
}

func NewChallengeService(repo *repo.PSQLRepository) *ChallengeService {
	return &ChallengeService{
		repo: repo,
	}
}

func (s *ChallengeService) PushSubmissionStatus(ctx context.Context, req *challengeService.PushSubmissionStatusRequest) (*challengeService.PushSubmissionStatusResponse, error) {
	fmt.Println(req)
	return &challengeService.PushSubmissionStatusResponse{
		Message: "received",
		Success: true,
	}, nil
}
