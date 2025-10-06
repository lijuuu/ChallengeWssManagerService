package ports

import (
	"context"

	"github.com/lijuuu/ChallengeWssManagerService/internal/model"
	challengePb "github.com/lijuuu/GlobalProtoXcode/ChallengeService"
)

// challenge service api surface used by transports
type ChallengeService interface {
	CreateChallenge(ctx context.Context, req *challengePb.ChallengeRecord) (*challengePb.ChallengeRecord, error)
	AbandonChallenge(ctx context.Context, req *challengePb.AbandonChallengeRequest) (*challengePb.AbandonChallengeResponse, error)
	GetFullChallengeData(ctx context.Context, req *challengePb.GetFullChallengeDataRequest) (*challengePb.GetFullChallengeDataResponse, error)
	GetChallengeHistory(ctx context.Context, req *challengePb.GetChallengeHistoryRequest) (*challengePb.ChallengeListResponse, error)
	GetActiveOpenChallenges(ctx context.Context, req *challengePb.PaginationRequest) (*challengePb.ChallengeListResponse, error)
	GetOwnersActiveChallenges(ctx context.Context, req *challengePb.GetOwnersActiveChallengesRequest) (*challengePb.ChallengeListResponse, error)
	PushSubmissionStatus(ctx context.Context, req *challengePb.PushSubmissionStatusRequest) (*challengePb.PushSubmissionStatusResponse, error)
}

// leaderboard port for inversion of control
type Leaderboard interface {
	InitializeLeaderboard(challengeID string) error
	CleanupLeaderboard(challengeID string) error
	UpdateParticipantScore(challengeID, userID string, score int) error
	GetLeaderboard(challengeID string, limit int, challenge *model.ChallengeDocument) ([]*model.LeaderboardEntry, error)
	GetParticipantRank(challengeID, userID string) (*model.ParticipantRank, error)
}
