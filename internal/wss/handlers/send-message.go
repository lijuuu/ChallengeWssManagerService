package wsshandler

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/lijuuu/ChallengeWssManagerService/internal/constants"
	"github.com/lijuuu/ChallengeWssManagerService/internal/model"
	"github.com/lijuuu/ChallengeWssManagerService/internal/wss/broadcasts"
	wsstypes "github.com/lijuuu/ChallengeWssManagerService/internal/wss/types"
)

type SendMessagePayload struct {
	UserId      string `json:"userId"`
	ChallengeId string `json:"challengeId"`
	ProfilePic  string `json:"profilePic"`
	Message     string `json:"message"`
}

type PushNewChatPayload struct {
	UserId      string `json:"userId"`
	ChallengeId string `json:"challengeId"`
	ProfilePic  string `json:"profilePic"`
	Message     string `json:"message"`
	Time        int64  `json:"time"`
}

type PushNewNotificationPayload struct {
	ChallengeId string `json:"challengeId"`
	Type        string `json:"type"`
	Message     string `json:"message"`
	Time        int64  `json:"time"`
}

// PUSHNEWCHAT: user sends a new chat message
func PushNewChatHandler(ctx *wsstypes.WsContext) error {
	var payload PushNewChatPayload
	raw, err := json.Marshal(ctx.Payload)
	if err != nil {
		log.Printf("[PUSH_NEW_CHAT] marshal payload failed: %v", err)
		return broadcasts.SendErrorWithType(ctx.Conn, constants.PUSH_NEW_CHAT, "Internal error", nil)
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		log.Printf("[PUSH_NEW_CHAT] unmarshal payload failed: %v", err)
		return broadcasts.SendErrorWithType(ctx.Conn, constants.PUSH_NEW_CHAT, "Invalid payload format", nil)
	}

	log.Printf("[PUSH_NEW_CHAT] received: userId=%s challengeId=%s message=%q", payload.UserId, payload.ChallengeId, payload.Message)

	chatMsg := model.ChatMessage{
		UserID:     payload.UserId,
		ProfilePic: payload.ProfilePic,
		Message:    payload.Message,
		Time:       time.Now().Unix(),
	}
	// Append chat to Mongo only
	if err := ctx.State.Mongo.AppendChatMessage(context.Background(), payload.ChallengeId, chatMsg); err != nil {
		log.Printf("[PUSH_NEW_CHAT] append chat failed: %v", err)
		return broadcasts.SendErrorWithType(ctx.Conn, constants.PUSH_NEW_CHAT, "Failed to store chat", nil)
	}
	log.Printf("[PUSH_NEW_CHAT] chat stored at %d", chatMsg.Time)
	// Optionally append a notification for chat summary
	n := model.Notification{Type: constants.CHAT_MESSAGE, Message: payload.UserId + ": " + payload.Message, Time: chatMsg.Time}
	if err := ctx.State.Mongo.AppendNotification(context.Background(), payload.ChallengeId, n); err != nil {
		log.Printf("[PUSH_NEW_CHAT] append notification failed: %v", err)
	}

	wsClients := ctx.State.LocalState.GetAllWSClients(payload.ChallengeId)
	log.Printf("[PUSH_NEW_CHAT] broadcasting to %d peers (excluding sender)", len(wsClients))
	for userID, conn := range wsClients {
		if userID == payload.UserId {
			continue
		}
		if err := broadcasts.SendStandardMessage(conn, constants.PUSH_NEW_CHAT, chatMsg, true, nil); err != nil {
			log.Printf("[PUSH_NEW_CHAT] broadcast to %s failed: %v", userID, err)
		}
	}
	log.Printf("[PUSH_NEW_CHAT] ack sent to sender %s", payload.UserId)
	return broadcasts.SendStandardSuccess(ctx.Conn, constants.PUSH_NEW_CHAT, map[string]any{"message": "sent"})
}

// PUSHNEWNOTIFICATION: append a single notification (for sync)
func PushNewNotificationHandler(ctx *wsstypes.WsContext) error {
	var payload PushNewNotificationPayload
	raw, err := json.Marshal(ctx.Payload)
	if err != nil {
		log.Printf("[PUSH_NEW_NOTIFICATION] marshal payload failed: %v", err)
		return broadcasts.SendErrorWithType(ctx.Conn, constants.PUSH_NEW_NOTIFICATION, "Internal error", nil)
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		log.Printf("[PUSH_NEW_NOTIFICATION] unmarshal payload failed: %v", err)
		return broadcasts.SendErrorWithType(ctx.Conn, constants.PUSH_NEW_NOTIFICATION, "Invalid payload format", nil)
	}

	log.Printf("[PUSH_NEW_NOTIFICATION] received: challengeId=%s type=%s msg=%q", payload.ChallengeId, payload.Type, payload.Message)
	notification := model.Notification{
		Type:    payload.Type,
		Message: payload.Message,
		Time:    payload.Time,
	}
	if err := ctx.State.Mongo.AppendNotification(context.Background(), payload.ChallengeId, notification); err != nil {
		log.Printf("[PUSH_NEW_NOTIFICATION] append notification failed: %v", err)
		return broadcasts.SendErrorWithType(ctx.Conn, constants.PUSH_NEW_NOTIFICATION, "Failed to store notification", nil)
	}
	wsClients := ctx.State.LocalState.GetAllWSClients(payload.ChallengeId)
	log.Printf("[PUSH_NEW_NOTIFICATION] broadcasting to %d peers (excluding sender if known)", len(wsClients))
	for userID, conn := range wsClients {
		if ctx.UserID != "" && userID == ctx.UserID {
			continue
		}
		if err := broadcasts.SendStandardMessage(conn, constants.PUSH_NEW_NOTIFICATION, notification, true, nil); err != nil {
			log.Printf("[PUSH_NEW_NOTIFICATION] broadcast to %s failed: %v", userID, err)
		}
	}
	log.Printf("[PUSH_NEW_NOTIFICATION] ack sent to sender %s", ctx.UserID)
	return broadcasts.SendStandardSuccess(ctx.Conn, constants.PUSH_NEW_NOTIFICATION, map[string]any{"message": "accepted"})
}
