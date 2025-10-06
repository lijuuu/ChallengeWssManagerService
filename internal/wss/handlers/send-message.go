package wsshandler

import (
	"context"
	"encoding/json"
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
		return broadcasts.SendErrorWithType(ctx.Conn, constants.PUSH_NEW_CHAT, "Internal error", nil)
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return broadcasts.SendErrorWithType(ctx.Conn, constants.PUSH_NEW_CHAT, "Invalid payload format", nil)
	}

	challengeDoc, err := ctx.State.Redis.GetChallengeByID(context.Background(), payload.ChallengeId)
	if err != nil {
		return broadcasts.SendErrorWithType(ctx.Conn, constants.PUSH_NEW_CHAT, "Challenge not found", nil)
	}

	chatMsg := model.ChatMessage{
		UserID:     payload.UserId,
		ProfilePic: payload.ProfilePic,
		Message:    payload.Message,
		Time:       time.Now().Unix(),
	}
	challengeDoc.Chat = append(challengeDoc.Chat, chatMsg)
	if err := ctx.State.Redis.UpdateChallenge(context.Background(), &challengeDoc); err != nil {
		return broadcasts.SendErrorWithType(ctx.Conn, constants.PUSH_NEW_CHAT, "Failed to update chat", nil)
	}

	wsClients := ctx.State.LocalState.GetAllWSClients(payload.ChallengeId)
	broadcasts.BroadcastStandardMessage(wsClients, constants.PUSH_NEW_CHAT, chatMsg, true, nil)
	return broadcasts.SendStandardSuccess(ctx.Conn, constants.PUSH_NEW_CHAT, map[string]any{"message": "sent"})
}

// PUSHNEWCHAT: append a single chat message (for sync)
func PushNewChatSyncHandler(ctx *wsstypes.WsContext) error {
	var payload PushNewChatPayload
	raw, err := json.Marshal(ctx.Payload)
	if err != nil {
		return broadcasts.SendErrorWithType(ctx.Conn, constants.PUSH_NEW_CHAT, "Internal error", nil)
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return broadcasts.SendErrorWithType(ctx.Conn, constants.PUSH_NEW_CHAT, "Invalid payload format", nil)
	}
	challengeDoc, err := ctx.State.Redis.GetChallengeByID(context.Background(), payload.ChallengeId)
	if err != nil {
		return broadcasts.SendErrorWithType(ctx.Conn, constants.PUSH_NEW_CHAT, "Challenge not found", nil)
	}
	chatMsg := model.ChatMessage{
		UserID:     payload.UserId,
		ProfilePic: payload.ProfilePic,
		Message:    payload.Message,
		Time:       payload.Time,
	}
	challengeDoc.Chat = append(challengeDoc.Chat, chatMsg)
	if err := ctx.State.Redis.UpdateChallenge(context.Background(), &challengeDoc); err != nil {
		return broadcasts.SendErrorWithType(ctx.Conn, constants.PUSH_NEW_CHAT, "Failed to update chat", nil)
	}
	wsClients := ctx.State.LocalState.GetAllWSClients(payload.ChallengeId)
	broadcasts.BroadcastStandardMessage(wsClients, constants.PUSH_NEW_CHAT, chatMsg, true, nil)
	return broadcasts.SendStandardSuccess(ctx.Conn, constants.PUSH_NEW_CHAT, map[string]any{"message": "accepted"})
}

// PUSHNEWNOTIFICATION: append a single notification (for sync)
func PushNewNotificationHandler(ctx *wsstypes.WsContext) error {
	var payload PushNewNotificationPayload
	raw, err := json.Marshal(ctx.Payload)
	if err != nil {
		return broadcasts.SendErrorWithType(ctx.Conn, constants.PUSH_NEW_NOTIFICATION, "Internal error", nil)
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return broadcasts.SendErrorWithType(ctx.Conn, constants.PUSH_NEW_NOTIFICATION, "Invalid payload format", nil)
	}
	challengeDoc, err := ctx.State.Redis.GetChallengeByID(context.Background(), payload.ChallengeId)
	if err != nil {
		return broadcasts.SendErrorWithType(ctx.Conn, constants.PUSH_NEW_NOTIFICATION, "Challenge not found", nil)
	}
	notification := model.Notification{
		Type:    payload.Type,
		Message: payload.Message,
		Time:    payload.Time,
	}
	challengeDoc.Notifications = append(challengeDoc.Notifications, notification)
	if err := ctx.State.Redis.UpdateChallenge(context.Background(), &challengeDoc); err != nil {
		return broadcasts.SendErrorWithType(ctx.Conn, constants.PUSH_NEW_NOTIFICATION, "Failed to update notifications", nil)
	}
	wsClients := ctx.State.LocalState.GetAllWSClients(payload.ChallengeId)
	broadcasts.BroadcastStandardMessage(wsClients, constants.PUSH_NEW_NOTIFICATION, notification, true, nil)
	return broadcasts.SendStandardSuccess(ctx.Conn, constants.PUSH_NEW_NOTIFICATION, map[string]any{"message": "accepted"})
}
