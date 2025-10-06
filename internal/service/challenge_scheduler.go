package service

import (
	"context"
	"fmt"
	"time"

	"github.com/lijuuu/ChallengeWssManagerService/internal/constants"
	"github.com/lijuuu/ChallengeWssManagerService/internal/model"
	"github.com/lijuuu/ChallengeWssManagerService/internal/wss/broadcasts"
)

// persistChallengeToMongoDB moves a finished challenge from Redis to MongoDB and cleans up Redis.
func (s *ChallengeService) persistChallengeToMongoDB(ctx context.Context, challengeID string) error {
	challengeDoc, err := s.GlobalState.Redis.GetChallengeByID(ctx, challengeID)
	if err != nil {
		return fmt.Errorf("failed to get challenge from Redis: %w", err)
	}

	if err := s.GlobalState.Mongo.PersistChallengeFromRedis(ctx, &challengeDoc); err != nil {
		return fmt.Errorf("failed to persist challenge to MongoDB: %w", err)
	}

	if err := s.GlobalState.Redis.DeleteChallenge(ctx, challengeID); err != nil {
		fmt.Printf("Warning: Failed to clean up Redis data for challenge %s: %v\n", challengeID, err)
	}

	return nil
}

// updateChallengeStatus updates the challenge status and triggers broadcasts/persistence if terminal.
func (s *ChallengeService) updateChallengeStatus(ctx context.Context, challengeID, newStatus string) error {
	challenge, err := s.GlobalState.Redis.GetChallenge(ctx, challengeID)
	if err != nil {
		return fmt.Errorf("failed to get challenge: %w", err)
	}

	challenge.Status = newStatus
	if err := s.GlobalState.Redis.UpdateChallenge(ctx, challenge); err != nil {
		return fmt.Errorf("failed to update challenge status: %w", err)
	}

	if newStatus == model.ChallengeEnded || newStatus == model.ChallengeAbandon {
		fmt.Printf("[Event] Challenge %s status set to %s\n", challengeID, newStatus)

		if s.GlobalState.LocalState != nil {
			wsClients := s.GlobalState.LocalState.GetAllWSClients(challengeID)

			fmt.Printf("[Broadcast] GAME_FINISHED for challenge %s\n", challengeID)
			broadcasts.BroadcastGameFinished(wsClients, challengeID)

			if newStatus == model.ChallengeAbandon {
				fmt.Printf("[Broadcast] CREATOR_ABANDON for challenge %s by user %s\n", challengeID, challenge.CreatorID)
				broadcasts.BroadcastChallengeAbandonWithClients(wsClients, challengeID, challenge.CreatorID)
			}
		}

		fmt.Printf("[Timer] Scheduling graceful close for challenge %s in 5s\n", challengeID)
		s.ScheduleGracefulClose(challengeID, 5*time.Second)
	}

	return nil
}

// ScheduleChallengeFinish sets up a timer to end a challenge after the specified duration.
func (s *ChallengeService) ScheduleChallengeFinish(challengeID string, duration time.Duration) {
	if s.GlobalState.ChallengeSchedulers == nil {
		s.GlobalState.ChallengeSchedulers = make(map[string]*time.Timer)
	}

	// cancel previous timer if exists
	if timer, ok := s.GlobalState.ChallengeSchedulers[challengeID]; ok {
		timer.Stop()
	}

	fmt.Printf("[Timer] Scheduling challenge %s to end in %v\n", challengeID, duration)

	timer := time.AfterFunc(duration, func() {
		fmt.Printf("[Timer] Timer fired for challenge %s, ending challenge now\n", challengeID)
		ctx := context.Background()
		if err := s.HandleChallengeTimeout(ctx, challengeID); err != nil {
			fmt.Printf("Error handling challenge timeout for %s: %v\n", challengeID, err)
		}
	})

	s.GlobalState.ChallengeSchedulers[challengeID] = timer
}

// HandleChallengeTimeout processes a challenge that has reached its time limit.
func (s *ChallengeService) HandleChallengeTimeout(ctx context.Context, challengeID string) error {
	fmt.Printf("[Timer] HandleChallengeTimeout called for challenge %s\n", challengeID)

	challenge, err := s.GlobalState.Redis.GetChallenge(ctx, challengeID)
	if err != nil {
		return fmt.Errorf("failed to get challenge for timeout handling: %w", err)
	}

	if challenge.Status == model.ChallengeEnded || challenge.Status == model.ChallengeAbandon {
		fmt.Printf("[Timer] Challenge %s already ended or abandoned, skipping timeout\n", challengeID)
		s.CancelChallengeTimer(challengeID)
		return nil
	}

	fmt.Printf("[Timer] Challenge %s naturally ending due to timer\n", challengeID)
	if err := s.updateChallengeStatus(ctx, challengeID, model.ChallengeEnded); err != nil {
		return fmt.Errorf("failed to update challenge status on timeout: %w", err)
	}

	s.CancelChallengeTimer(challengeID)
	fmt.Printf("[Timer] Challenge timer cleaned for challenge %s\n", challengeID)

	return nil
}

// ScheduleGracefulClose waits for a grace period, then persists and cleans up local state and Redis.
func (s *ChallengeService) ScheduleGracefulClose(challengeID string, delay time.Duration) {
	// cancel previous timer if exists
	if timer, ok := s.GlobalState.ChallengeSchedulers[challengeID]; ok {
		timer.Stop()
	}

	fmt.Printf("[Timer] Scheduling graceful cleanup for challenge %s in %v\n", challengeID, delay)

	timer := time.AfterFunc(delay, func() {
		ctx := context.Background()

		if err := s.persistChallengeToMongoDB(ctx, challengeID); err != nil {
			fmt.Printf("Warning: Failed to persist challenge %s after grace period: %v\n", challengeID, err)
		}

		if s.GlobalState.LocalState != nil {
			s.GlobalState.LocalState.CleanupChallenge(challengeID)
		}

		if s.GlobalState.Redis != nil {
			if err := s.GlobalState.Redis.DeleteChallenge(ctx, challengeID); err != nil {
				fmt.Printf("Warning: Failed to delete challenge %s from Redis during cleanup: %v\n", challengeID, err)
			}
		}

		fmt.Printf("[Cleanup] Graceful cleanup done for challenge %s\n", challengeID)
		delete(s.GlobalState.ChallengeSchedulers, challengeID)
	})

	s.GlobalState.ChallengeSchedulers[challengeID] = timer
}

// CancelChallengeTimer cancels the timer for a challenge.
func (s *ChallengeService) CancelChallengeTimer(challengeID string) {
	if s.GlobalState.ChallengeSchedulers == nil {
		return
	}

	if timer, exists := s.GlobalState.ChallengeSchedulers[challengeID]; exists {
		timer.Stop()
		delete(s.GlobalState.ChallengeSchedulers, challengeID)
	}
}

// WarmUpScheduler reinitializes timers for open/started challenges on server restart.
func (s *ChallengeService) WarmUpScheduler(ctx context.Context) error {
	openChallengeIds, _ := s.RedisRepo.GetChallengesByStatus(ctx, constants.CHALLENGE_OPEN)
	startedChallengeIds, _ := s.RedisRepo.GetChallengesByStatus(ctx, constants.CHALLENGE_STARTED)

	challengeIds := append(openChallengeIds, startedChallengeIds...)

	fmt.Printf("[Timer] WarmingUp Challenge Timeouts for %v\n", challengeIds)
	for _, id := range challengeIds {
		s.HandleChallengeTimeout(ctx, id)
	}

	return nil
}
