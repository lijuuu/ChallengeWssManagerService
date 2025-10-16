package wss

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/lijuuu/ChallengeWssManagerService/internal/constants"
	"github.com/lijuuu/ChallengeWssManagerService/internal/global"
	"github.com/lijuuu/ChallengeWssManagerService/internal/model"
	"github.com/lijuuu/ChallengeWssManagerService/internal/wss/broadcasts"
	wsstypes "github.com/lijuuu/ChallengeWssManagerService/internal/wss/types"
)

// accept all origins for websocket upgrades.
// consider restricting this in production.
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// wshandler upgrades http connections to websocket and dispatches messages.
func WsHandler(dispatcher *Dispatcher, state *global.State) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("[WS] upgrade error:", err)
			return
		}
		defer conn.Close()
		log.Println("[WS] WebSocket connection established")

		var userID, challengeID string

		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				log.Printf("[WS] read error: %v (user: %s, challenge: %s)", err, userID, challengeID)
				cleanupConnection(state, userID, challengeID)
				return
			}

			var wsMsg wsstypes.WsMessageRequest
			if err := json.Unmarshal(msg, &wsMsg); err != nil {
				log.Println("[WS] invalid message format:", err)
				continue
			}

			if wsMsg.Type != constants.PING_SERVER {
				log.Printf("[WS] received: type=%s payload=%v", wsMsg.Type, wsMsg.Payload)
			}

			//track ids for cleanup if the connection drops later.
			if uid, ok := wsMsg.Payload["userId"].(string); ok {
				userID = uid
			}
			if cid, ok := wsMsg.Payload["challengeId"].(string); ok {
				challengeID = cid
			}

			ctx := &wsstypes.WsContext{
				Conn:    conn,
				Payload: wsMsg.Payload,
				State:   state,
			}

			if err := dispatcher.Dispatch(wsMsg.Type, ctx); err != nil {
				log.Printf("[Dispatch] error handling %s: %v", wsMsg.Type, err)
			}
		}
	}
}

// cleanupconnection removes ephemeral state and notifies peers when a client disconnects.
func cleanupConnection(state *global.State, userID, challengeID string) {
	if userID == "" || challengeID == "" {
		log.Println("[WS] skipping cleanup: userID or challengeID missing")
		return
	}

	log.Printf("[WS] cleaning up session: user=%s challenge=%s", userID, challengeID)

	//fetch challenge info for accurate owner/participant broadcast.
	challengeDoc, err := state.Redis.GetChallengeByID(context.Background(), challengeID)
	if err != nil {
		log.Printf("[WS] failed to get challenge for cleanup broadcast: %v", err)
	}
	//remove participant from redis.
	//but dont remove if its in challengeStart phase

	if challengeDoc.Status != model.ChallengeStarted {
		if err := state.Redis.RemoveParticipantInJoinPhase(context.Background(), challengeID, userID); err != nil {
			log.Printf("[Redis] failed to remove from Redis: %v", err)
		} else {
			log.Printf("[Redis] user %s removed from Redis for challenge %s", userID, challengeID)
		}
	}

	_, exist := challengeDoc.Participants[userID]

	//remove connection and session from local state.
	state.LocalState.RemoveWSClient(challengeID, userID)
	log.Printf("[WS] user %s removed from local state for challenge %s", userID, challengeID)

	//notify remaining clients about the departure only if the userId was part of the challenge
	if err == nil && exist {
		wsClients := state.LocalState.GetAllWSClients(challengeID)
		broadcasts.BroadcastEntityLeftWithClients(wsClients, userID, challengeID, userID == challengeDoc.CreatorID)
	}
}
