package global

import (
	"time"

	"github.com/lijuuu/ChallengeWssManagerService/internal/jwt"
	localstate "github.com/lijuuu/ChallengeWssManagerService/internal/local"
	"github.com/lijuuu/ChallengeWssManagerService/internal/ports"
	"github.com/panjf2000/ants/v2"
)

// state groups shared dependencies used by websocket handlers and grpc services.
type State struct {
	Redis              ports.RedisRepository
	Mongo              ports.MongoRepository
	LocalState         *localstate.LocalStateManager
	LeaderboardManager ports.Leaderboard
	JwtManager         *jwt.JWTManager
	AntsWorkerPool     *ants.Pool
	// Challenge scheduler for managing challenge timers globally
	ChallengeSchedulers map[string]*time.Timer
}
