package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/lijuuu/ChallengeWssManagerService/internal/constants"
	"github.com/lijuuu/ChallengeWssManagerService/internal/model"
	"github.com/lijuuu/ChallengeWssManagerService/internal/utils"
	challengePb "github.com/lijuuu/GlobalProtoXcode/ChallengeService"
)

func (s *ChallengeService) CreateChallenge(ctx context.Context, req *challengePb.ChallengeRecord) (*challengePb.CreateChallengeResponse, error) {
	canCreate, err := s.GlobalState.Redis.CanCreate(ctx, req.CreatorId)
	if err != nil {
		return nil, err
	}
	if !canCreate {
		return nil, fmt.Errorf("active challenge already found, can't create new challenge")
	}

	modelChallengeDoc := ChallengeDocumentFromProto(req, false)

	// Initialize challenge document for Redis storage
	modelChallengeDoc.Status = model.ChallengeOpen
	modelChallengeDoc.Participants = make(map[string]*model.ParticipantMetadata)
	modelChallengeDoc.Submissions = make(map[string]map[string]model.Submission)
	modelChallengeDoc.Leaderboard = make([]*model.LeaderboardEntry, 0)

	modelChallengeDoc.Participants[modelChallengeDoc.CreatorID] = &model.ParticipantMetadata{
		JoinTime: time.Now().Unix(),
	}

	if req.IsPrivate {
		modelChallengeDoc.Password = utils.GenerateBigCapPassword(7)
	}

	modelChallengeDoc.Leaderboard = append(modelChallengeDoc.Leaderboard, &model.LeaderboardEntry{
		UserID:            req.CreatorId,
		TotalScore:        0,
		Rank:              0,
		ProblemsCompleted: 0,
	})

	modelChallengeDoc.Config = &model.ChallengeConfig{
		MaxEasyQuestions:   int(req.GetConfig().GetMaxEasyQuestions()),
		MaxMediumQuestions: int(req.GetConfig().GetMaxMediumQuestions()),
		MaxHardQuestions:   int(req.GetConfig().GetMaxHardQuestions()),
		MaxUsers:           int(req.GetConfig().GetMaxUsers()),
	}

	modelChallengeDoc.Status = constants.CHALLENGE_OPEN
	modelChallengeDoc.TimeLimit = req.TimeLimitMillis
	// Ensure arrays exist for Mongo appends from the start
	if modelChallengeDoc.Notifications == nil {
		modelChallengeDoc.Notifications = make([]model.Notification, 0)
	}
	if modelChallengeDoc.Chat == nil {
		modelChallengeDoc.Chat = make([]model.ChatMessage, 0)
	}

	// Create challenge in Redis
	if err := s.GlobalState.Redis.CreateChallenge(ctx, modelChallengeDoc); err != nil {
		return nil, err
	}

	// Upsert initial challenge document to Mongo immediately (immutable metadata + empty chat/notifs)
	if err := s.GlobalState.Mongo.PersistChallengeFromRedis(ctx, modelChallengeDoc); err != nil {
		return nil, fmt.Errorf("failed to create challenge document in Mongo: %w", err)
	}

	// Initialize leaderboard for the new challenge
	if err := s.GlobalState.LeaderboardManager.InitializeLeaderboard(modelChallengeDoc.ChallengeID); err != nil {
		log.Printf("[CreateChallenge] Warning: Failed to initialize leaderboard for challenge %s: %v", modelChallengeDoc.ChallengeID, err)
		// Don't fail challenge creation if leaderboard initialization fails
	} else {
		// Add creator to leaderboard with initial score of 0
		if err := s.GlobalState.LeaderboardManager.UpdateParticipantScore(modelChallengeDoc.ChallengeID, modelChallengeDoc.CreatorID, 0); err != nil {
			log.Printf("[CreateChallenge] Warning: Failed to add creator to leaderboard for challenge %s: %v", modelChallengeDoc.ChallengeID, err)
		}
	}

	// schedule challenge start at startTime
	startAt := time.Unix(req.StartTimeUnix, 0)
	s.ScheduleChallengeStart(modelChallengeDoc.ChallengeID, startAt)

	// also schedule finish relative to start time + time limit
	endTime := startAt.Add(time.Millisecond * time.Duration(req.TimeLimitMillis))
	duration := time.Until(endTime)
	if duration < 0 {
		duration = 0
	}
	s.ScheduleChallengeFinish(modelChallengeDoc.ChallengeID, duration)


	return &challengePb.CreateChallengeResponse{
		Success:   true,
		Challenge: req,
	}, nil
}
