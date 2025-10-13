package ports

import (
	"context"
	"time"

	"github.com/lijuuu/ChallengeWssManagerService/internal/model"
)

// redis repository behaviors used by the app
type RedisRepository interface {
	CreateChallenge(ctx context.Context, challenge *model.ChallengeDocument) error
	GetChallenge(ctx context.Context, challengeID string) (*model.ChallengeDocument, error)
	GetChallengeByID(ctx context.Context, challengeID string) (model.ChallengeDocument, error)
	UpdateChallenge(ctx context.Context, challenge *model.ChallengeDocument) error
	DeleteChallenge(ctx context.Context, challengeID string) error

	GetChallenges(ctx context.Context) ([]string, error)
	GetChallengesByStatus(ctx context.Context, status []string) ([]string, error)

	AddParticipant(ctx context.Context, challengeID, userID string, metadata *model.ParticipantMetadata) error
	UpdateParticipant(ctx context.Context, challengeID, userID string, metadata *model.ParticipantMetadata) error
	RemoveParticipant(ctx context.Context, challengeID, userID string) error
	RemoveParticipantInJoinPhase(ctx context.Context, challengeID, userID string) error

	AbandonChallenge(ctx context.Context, creatorID, challengeID string) error

	CanJoin(ctx context.Context, challengeID, userID string) (bool, error)
	CanCreate(ctx context.Context, userID string) (bool, error)
}

// mongo repository behaviors used by the app
type MongoRepository interface {
	PersistChallengeFromRedis(ctx context.Context, challenge *model.ChallengeDocument) error
	GetChallengeHistory(ctx context.Context, userID string, page, pageSize int, isPrivate bool) ([]model.ChallengeDocument, error)
	GetChallengeByID(ctx context.Context, challengeId string) (model.ChallengeDocument, error)
	GetAllChallenges(ctx context.Context) ([]model.ChallengeDocument, error)
	// Chat/Notifications
	AppendChatMessage(ctx context.Context, challengeId string, msg model.ChatMessage) error
	AppendNotification(ctx context.Context, challengeId string, n model.Notification) error
}

// ChallengeServicePorts exposes scheduling helpers to ws layer
type ChallengeServicePorts interface {
	ScheduleChallengeFinish(challengeID string, duration time.Duration)
}
