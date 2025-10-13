package service

import (
	"context"
	"sync"

	"github.com/lijuuu/ChallengeWssManagerService/internal/model"
)

func (s *ChallengeService) fetchChallengesConcurrently(ctx context.Context, challengeIDs []string, isPrivate bool) []model.ChallengeDocument {
	var challenges []model.ChallengeDocument
	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, challengeID := range challengeIDs {
		wg.Add(1)
		_ = s.GlobalState.AntsWorkerPool.Submit(func() {
			defer wg.Done()
			challenge, err := s.GlobalState.Redis.GetChallengeByID(ctx, challengeID)
			if err == nil {
				mu.Lock()
				if challenge.IsPrivate == isPrivate {
					challenges = append(challenges, challenge)
				}
				mu.Unlock()
			}
		})
	}
	wg.Wait()
	return challenges
}
