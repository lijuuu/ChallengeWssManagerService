package wss

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/lijuuu/ChallengeWssManagerService/internal/wss/broadcasts"
	wsstypes "github.com/lijuuu/ChallengeWssManagerService/internal/wss/types"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func WsHandler(dispatcher *Dispatcher, state *wsstypes.State) http.HandlerFunc {
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

			var wsMsg wsstypes.WsMessage
			if err := json.Unmarshal(msg, &wsMsg); err != nil {
				log.Println("[WS] invalid message format:", err)
				continue
			}

			log.Printf("[WS] received: type=%s payload=%v", wsMsg.Type, wsMsg.Payload)

			// track for cleanup
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

func cleanupConnection(state *wsstypes.State, userID, challengeID string) {
	if userID == "" || challengeID == "" {
		log.Println("[WS] skipping cleanup: userID or challengeID missing")
		return
	}

	log.Printf("[WS] cleaning up session: user=%s challenge=%s", userID, challengeID)

	state.Mu.Lock()
	defer state.Mu.Unlock()

	if err := state.Repo.RemoveParticipantInJoinPhase(context.Background(), challengeID, userID); err != nil {
		log.Printf("[Repo] failed to remove from DB: %v", err)
	} else {
		log.Printf("[Repo] user %s removed from DB for challenge %s", userID, challengeID)
	}

	challenge, exists := state.Challenges[challengeID]
	if !exists {
		log.Printf("[WS] challenge %s not found in memory", challengeID)
		return
	}

	challenge.MU.Lock()
	defer challenge.MU.Unlock()

	delete(challenge.Participants, userID)
	delete(challenge.Sessions, userID)
	delete(challenge.WSClients, userID)

	log.Printf("[WS] user %s removed from in-memory challenge %s", userID, challengeID)

	broadcasts.BroadcastEntityLeft(challenge, userID, challengeID, userID == challenge.CreatorID)
}
