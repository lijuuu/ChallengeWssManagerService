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

	if newStatus == model.ChallengeStarted {
		if s.GlobalState.LocalState != nil {
			wsClients := s.GlobalState.LocalState.GetAllWSClients(challengeID)
			broadcasts.BroadcastGameStarted(wsClients, challengeID, challenge.StartTime)
		}
		_ = s.GlobalState.Mongo.AppendNotification(ctx, challengeID, model.Notification{
			Type:    "SYSTEM",
			Message: "Challenge has started",
			Time:    time.Now().Unix(),
		})
	}

	if newStatus == model.ChallengeEnded || newStatus == model.ChallengeAbandon {
		fmt.Printf("[Event] Challenge %s status set to %s\n", challengeID, newStatus)

		// Store system notification in Mongo for per-challenge logs
		_ = s.GlobalState.Mongo.AppendNotification(ctx, challengeID, model.Notification{
			Type:    "SYSTEM",
			Message: fmt.Sprintf("Challenge status changed to %s", newStatus),
			Time:    time.Now().Unix(),
		})

		if s.GlobalState.LocalState != nil {
			wsClients := s.GlobalState.LocalState.GetAllWSClients(challengeID)

			fmt.Printf("[Broadcast] GAME_FINISHED for challenge %s\n", challengeID)
			broadcasts.BroadcastGameFinished(wsClients, challengeID)
			// Also push notification to connected clients
			broadcasts.BroadcastStandardMessage(wsClients, constants.PUSH_NEW_NOTIFICATION, model.Notification{
				Type:    "SYSTEM",
				Message: fmt.Sprintf("Challenge status changed to %s", newStatus),
				Time:    time.Now().Unix(),
			}, true, nil)

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

// ScheduleChallengeStart sets up a timer to start a challenge at startTime.
func (s *ChallengeService) ScheduleChallengeStart(challengeID string, startAt time.Time) {
	if s.GlobalState.ChallengeSchedulers == nil {
		s.GlobalState.ChallengeSchedulers = make(map[string]*time.Timer)
	}

	// cancel previous timer if exists
	if timer, ok := s.GlobalState.ChallengeSchedulers[challengeID]; ok {
		timer.Stop()
	}

	delay := time.Until(startAt)
	if delay < 0 {
		delay = 0
	}

	fmt.Printf("[Timer] Scheduling challenge %s to start in %v\n", challengeID, delay)

	timer := time.AfterFunc(delay, func() {
		fmt.Printf("[Timer] Start timer fired for challenge %s, starting now\n", challengeID)
		ctx := context.Background()
		// Transition to STARTED and broadcast
		_ = s.updateChallengeStatus(ctx, challengeID, model.ChallengeStarted)

		// After starting, schedule finish according to time limit from now until start+limit
		ch, err := s.GlobalState.Redis.GetChallenge(ctx, challengeID)
		if err == nil {
			endAt := time.Unix(ch.StartTime, 0).Add(time.Duration(ch.TimeLimit) * time.Millisecond)
			dur := time.Until(endAt)
			if dur < 0 {
				dur = 0
			}
			s.ScheduleChallengeFinish(challengeID, dur)
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

	fmt.Printf("[Timer] WarmUpScheduler: open=%v started=%v\n", openChallengeIds, startedChallengeIds)

	// For OPEN challenges: schedule start at startTime and finish at startTime+limit
	for _, id := range openChallengeIds {
		ch, err := s.RedisRepo.GetChallengeByID(ctx, id)
		if err != nil {
			fmt.Printf("[Timer] WarmUp OPEN get challenge %s failed: %v\n", id, err)
			continue
		}
		if ch.StartTime > 0 {
			s.ScheduleChallengeStart(id, time.Unix(ch.StartTime, 0))
			endAt := time.Unix(ch.StartTime, 0).Add(time.Duration(ch.TimeLimit) * time.Millisecond)
			s.ScheduleChallengeFinish(id, time.Until(endAt))
		}
	}

	// For STARTED challenges: schedule finish at startTime+limit from now
	for _, id := range startedChallengeIds {
		ch, err := s.RedisRepo.GetChallengeByID(ctx, id)
		if err != nil {
			fmt.Printf("[Timer] WarmUp STARTED get challenge %s failed: %v\n", id, err)
			continue
		}
		endAt := time.Unix(ch.StartTime, 0).Add(time.Duration(ch.TimeLimit) * time.Millisecond)
		s.ScheduleChallengeFinish(id, time.Until(endAt))
	}

	return nil
}
