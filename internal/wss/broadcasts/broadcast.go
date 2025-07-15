package broadcasts

import (
	"time"

	"github.com/lijuuu/ChallengeWssManagerService/internal/model"
	wsstypes "github.com/lijuuu/ChallengeWssManagerService/internal/wss/types"
)

func BroadcastEntityLeft(challenge *model.Challenge, userID, challengeID string, owner bool) {
	for _, conn := range challenge.WSClients {
		if conn == nil {
			continue
		}
		evtype := wsstypes.USER_LEFT
		message := "User has Left the challenge"
		if owner {
			evtype = wsstypes.OWNER_LEFT
			message = "Owner has Left the challenge"

		}
		SendJSON(conn, map[string]interface{}{
			"type":    evtype,
			"status":  "ok",
			"message": message,
			"payload": map[string]interface{}{
				"userId":      userID,
				"challengeId": challengeID,
				"time":        time.Now(),
			},
		})
	}
}

func BroadcastEntityJoined(challenge *model.Challenge, userID, challengeID string, isOwner bool) {
	for _, conn := range challenge.WSClients {
		if conn == nil {
			continue
		}

		evtype := wsstypes.USER_JOINED
		message := "User has joined the challenge"
		if isOwner {
			evtype = wsstypes.OWNER_JOINED
			message = "Owner has joined the challenge"
		}

		SendJSON(conn, map[string]interface{}{
			"type":    evtype,
			"status":  "ok",
			"message": message,
			"payload": map[string]interface{}{
				"userId":      userID,
				"challengeId": challengeID,
				"time":        time.Now(),
			},
		})
	}
}

func BroadcastChallengeAbandon(challenge *model.Challenge) {
	if challenge == nil{
		return
	}
	
	for _, conn := range challenge.WSClients {
		if conn == nil {
			continue
		}

		SendJSON(conn, map[string]interface{}{
			"type":    wsstypes.CREATOR_ABANDON,
			"status":  "ok",
			"message": "Creator abandoned the challenge",
			"payload": map[string]interface{}{
				"challengeId": challenge.ChallengeID,
				"userId":      challenge.CreatorID,
				"time":        time.Now(),
			},
		})
	}
}

// func TimeSync(){}

//
