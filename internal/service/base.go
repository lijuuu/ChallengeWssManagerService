package service

import (
	"github.com/lijuuu/ChallengeWssManagerService/internal/global"
	"github.com/lijuuu/ChallengeWssManagerService/internal/ports"
	challengePb "github.com/lijuuu/GlobalProtoXcode/ChallengeService"
)

// ChallengeService holds application-level dependencies for challenge operations.
type ChallengeService struct {
	GlobalState *global.State
	RedisRepo   ports.RedisRepository
	MongoRepo   ports.MongoRepository
	Board       ports.Leaderboard
	challengePb.UnimplementedChallengeServiceServer
}

// NewChallengeService constructs a ChallengeService from the shared state.
func NewChallengeService(state *global.State) *ChallengeService {
	svc := &ChallengeService{
		GlobalState: state,
		RedisRepo:   state.Redis,
		MongoRepo:   state.Mongo,
		Board:       state.LeaderboardManager,
	}
	// wire back reference for ws to call schedulers
	if state != nil {
		state.ServiceRef = svc
	}
	return svc
}
