package utils

import (
	"fmt"
	"time"

	"github.com/lijuuu/ChallengeWssManagerService/internal/constants"
	"github.com/lijuuu/ChallengeWssManagerService/internal/model"
)

// newsubmissionnotification builds a human-readable notification for submissions.
func NewSubmissionNotification(userID, problemID string, score int) model.Notification {
	return model.Notification{
		Type:    constants.NEW_SUBMISSION,
		Message: fmt.Sprintf("%s scored %d on %s", userID, score, problemID),
		Time:    time.Now().Unix(),
	}
}

// userjoinednotification builds a notification for joins.
func UserJoinedNotification(userID string) model.Notification {
	return model.Notification{
		Type:    constants.USER_JOINED,
		Message: fmt.Sprintf("%s joined the challenge", userID),
		Time:    time.Now().Unix(),
	}
}

// userleftnotification builds a notification for leaves.
func UserLeftNotification(userID string) model.Notification {
	return model.Notification{
		Type:    constants.USER_LEFT,
		Message: fmt.Sprintf("%s left the challenge", userID),
		Time:    time.Now().Unix(),
	}
}

// gamefinishednotification builds a notification for game end.
func GameFinishedNotification() model.Notification {
	return model.Notification{
		Type:    constants.GAME_FINISHED,
		Message: "Challenge finished",
		Time:    time.Now().Unix(),
	}
}

// leaderboardupdatenotification builds a notification for leaderboard updates (optional brief).
func LeaderboardUpdateNotification(updatedUser string) model.Notification {
	msg := "Leaderboard updated"
	if updatedUser != "" {
		msg = fmt.Sprintf("Leaderboard updated (latest: %s)", updatedUser)
	}
	return model.Notification{
		Type:    constants.LEADERBOARD_UPDATE,
		Message: msg,
		Time:    time.Now().Unix(),
	}
}

// chatmessagenotification builds a notification for a chat message (summary only).
func ChatMessageNotification(userID string) model.Notification {
	return model.Notification{
		Type:    constants.CHAT_MESSAGE,
		Message: fmt.Sprintf("%s sent a message", userID),
		Time:    time.Now().Unix(),
	}
}
