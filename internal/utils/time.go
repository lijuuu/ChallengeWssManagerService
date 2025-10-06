package utils

import (
	"time"

	"github.com/lijuuu/ChallengeWssManagerService/internal/model"
)

// ischallengeexpired returns true if startTime + timeLimit has passed or status is terminal.
func IsChallengeExpired(ch model.ChallengeDocument) bool {
	if ch.Status == model.ChallengeEnded || ch.Status == model.ChallengeAbandon {
		return true
	}
	if ch.StartTime == 0 || ch.TimeLimit <= 0 {
		return false
	}
	expiry := time.Unix(ch.StartTime, 0).Add(time.Duration(ch.TimeLimit) * time.Millisecond)
	return time.Now().After(expiry)
}
