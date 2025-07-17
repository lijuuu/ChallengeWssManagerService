package broadcasts

import (
	"time"

	"github.com/gorilla/websocket"
	wsstypes "github.com/lijuuu/ChallengeWssManagerService/internal/wss/types"
)

func BroadcastEntityJoinedWithClients(wsClients map[string]*websocket.Conn, userID, challengeID string, isOwner bool) {
	for _, conn := range wsClients {
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

func BroadcastEntityLeftWithClients(wsClients map[string]*websocket.Conn, userID, challengeID string, owner bool) {
	for _, conn := range wsClients {
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

func BroadcastChallengeAbandonWithClients(wsClients map[string]*websocket.Conn, challengeID, creatorID string) {
	for _, conn := range wsClients {
		if conn == nil {
			continue
		}

		SendJSON(conn, map[string]interface{}{
			"type":    wsstypes.CREATOR_ABANDON,
			"status":  "ok",
			"message": "Creator abandoned the challenge",
			"payload": map[string]interface{}{
				"challengeId": challengeID,
				"userId":      creatorID,
				"time":        time.Now(),
			},
		})
	}
}

// func TimeSync(){}

//
