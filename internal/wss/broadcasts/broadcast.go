package broadcasts

import (
	"time"

	"github.com/gorilla/websocket"
	"github.com/lijuuu/ChallengeWssManagerService/internal/constants"
	"github.com/lijuuu/ChallengeWssManagerService/internal/model"
	wsstypes "github.com/lijuuu/ChallengeWssManagerService/internal/wss/types"
)

// sendstandardmessage writes a structured message to one websocket client.
// most other helpers build on top of this.
func SendStandardMessage(conn *websocket.Conn, msgType string, payload any, success bool, errorMsg *string) error {
	message := wsstypes.StandardBroadcastMessage{
		Type:    msgType,
		Payload: payload,
		Success: success,
		Error:   errorMsg,
	}
	return SendJSON(conn, message)
}

// sendjson writes an arbitrary value as json to one websocket client.
func SendJSON(conn *websocket.Conn, data interface{}) error {
	return conn.WriteJSON(data)
}

// broadcaststandardmessage sends the same structured message to every client in the map.
func BroadcastStandardMessage(wsClients map[string]*websocket.Conn, msgType string, payload any, success bool, errorMsg *string) {
	message := wsstypes.StandardBroadcastMessage{
		Type:    msgType,
		Payload: payload,
		Success: success,
		Error:   errorMsg,
	}

	//send sequentially to avoid concurrent writes on the same connection.
	for _, conn := range wsClients {
		if conn == nil {
			continue //w ignore empty slots
		}
		//w synchronous write per connection.
		if err := SendJSON(conn, message); err != nil {
			//consider removing dead connections upstream.
		}
	}
}

// sendstandarderror sends a structured error to one client.
func SendStandardError(conn *websocket.Conn, msgType string, errorMsg string) error {
	errorPtr := &errorMsg
	return SendStandardMessage(conn, msgType, nil, false, errorPtr)
}

// broadcaststandarderror broadcasts the same error to all clients.
func BroadcastStandardError(wsClients map[string]*websocket.Conn, msgType string, errorMsg string) {
	errorPtr := &errorMsg
	BroadcastStandardMessage(wsClients, msgType, nil, false, errorPtr)
}

// sendstandardsuccess sends a success response with payload to one client.
func SendStandardSuccess(conn *websocket.Conn, msgType string, payload any) error {
	return SendStandardMessage(conn, msgType, payload, true, nil)
}

// broadcaststandardsuccess sends a success response with payload to all clients.
func BroadcastStandardSuccess(wsClients map[string]*websocket.Conn, msgType string, payload any) {
	BroadcastStandardMessage(wsClients, msgType, payload, true, nil)
}

// senderrorwithtype sends an error under a specific event type with extra context.
func SendErrorWithType(conn *websocket.Conn, eventType string, msg string, extra map[string]any) error {
	errorMsg := msg
	return SendStandardMessage(conn, eventType, extra, false, &errorMsg)
}

// broadcastentityjoinedwithclients notifies clients that someone joined the challenge.
func BroadcastEntityJoinedWithClients(wsClients map[string]*websocket.Conn, userID, challengeID string, isOwner bool) {
	eventType := constants.USER_JOINED
	if isOwner {
		eventType = constants.OWNER_JOINED
	}

	payload := map[string]any{
		"userId":      userID,
		"challengeId": challengeID,
		"time":        time.Now(),
	}

	BroadcastStandardMessage(wsClients, eventType, payload, true, nil)
}

// broadcastentityleftwithclients notifies clients that someone left the challenge.
func BroadcastEntityLeftWithClients(wsClients map[string]*websocket.Conn, userID, challengeID string, isOwner bool) {
	eventType := constants.USER_LEFT
	if isOwner {
		eventType = constants.OWNER_LEFT
	}

	payload := map[string]any{
		"userId":      userID,
		"challengeId": challengeID,
		"time":        time.Now(),
	}

	BroadcastStandardMessage(wsClients, eventType, payload, true, nil)
}

// broadcastchallengeabandonwithclients tells clients that the creator abandoned the challenge.
func BroadcastChallengeAbandonWithClients(wsClients map[string]*websocket.Conn, challengeID, creatorID string) {
	payload := map[string]any{
		"challengeId": challengeID,
		"userId":      creatorID,
		"time":        time.Now(),
	}

	BroadcastStandardMessage(wsClients, constants.CREATOR_ABANDON, payload, true, nil)
}

// broadcastnewsubmission informs clients about a new successful submission.
func BroadcastNewSubmission(wsClients map[string]*websocket.Conn, challengeID, userID, problemID string, score, newRank int) {
	payload := map[string]any{
		"challengeId": challengeID,
		"userId":      userID,
		"problemId":   problemID,
		"score":       score,
		"newRank":     newRank,
		"time":        time.Now(),
	}
	BroadcastStandardMessage(wsClients, constants.PUSH_SUBMISSION, payload, true, nil)
}

// broadcastleaderboardupdate sends the latest leaderboard to clients.
func BroadcastLeaderboardUpdate(wsClients map[string]*websocket.Conn, challengeID string, leaderboard []*model.LeaderboardEntry, updatedUser string) {
	payload := map[string]any{
		"challengeId": challengeID,
		"leaderboard": leaderboard,
		"updatedUser": updatedUser,
		"time":        time.Now(),
	}

	BroadcastStandardMessage(wsClients, constants.LEADERBOARD_UPDATE, payload, true, nil)
}

// broadcastgamefinished notifies clients that the game has ended.
func BroadcastGameFinished(wsClients map[string]*websocket.Conn, challengeID string) {
	payload := map[string]any{
		"challengeId": challengeID,
		"time":        time.Now(),
	}
	BroadcastStandardMessage(wsClients, constants.GAME_FINISHED, payload, true, nil)
}

// broadcastchatmessage sends a chat message to all clients in a challenge.
func BroadcastChatMessage(wsClients map[string]*websocket.Conn, challengeID string, message map[string]any) {
	payload := map[string]any{
		"challengeId": challengeID,
		"message":     message,
		"time":        time.Now(),
	}
	BroadcastStandardMessage(wsClients, constants.CHAT_MESSAGE, payload, true, nil)
}
