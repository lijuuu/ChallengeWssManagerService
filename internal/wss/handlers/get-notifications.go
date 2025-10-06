package wsshandler

import (
	"context"
	"encoding/json"
	"log"

	"github.com/google/uuid"
	"github.com/lijuuu/ChallengeWssManagerService/internal/constants"
	"github.com/lijuuu/ChallengeWssManagerService/internal/wss/broadcasts"
	wsstypes "github.com/lijuuu/ChallengeWssManagerService/internal/wss/types"
)

type GetNotificationsPayload struct {
	UserId      string `json:"userId"`
	ChallengeId string `json:"challengeId"`
}

func GetNotificationsHandler(ctx *wsstypes.WsContext) error {
	requestID := uuid.New().String()

	var payload GetNotificationsPayload
	raw, err := json.Marshal(ctx.Payload)
	if err != nil {
		log.Printf("[%s] [GetNotifications] Marshal error: %v", requestID, err)
		return broadcasts.SendErrorWithType(ctx.Conn, constants.GET_NOTIFICATIONS, "Internal error", nil)
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		log.Printf("[%s] [GetNotifications] Unmarshal error: %v", requestID, err)
		return broadcasts.SendErrorWithType(ctx.Conn, constants.GET_NOTIFICATIONS, "Invalid payload format", nil)
	}

	ch, err := ctx.State.Redis.GetChallengeByID(context.Background(), payload.ChallengeId)
	if err != nil {
		return broadcasts.SendErrorWithType(ctx.Conn, constants.GET_NOTIFICATIONS, "Challenge not found", nil)
	}

	return broadcasts.SendJSON(ctx.Conn, map[string]any{
		"type":   constants.GET_NOTIFICATIONS,
		"status": "ok",
		"payload": map[string]any{
			"notifications": ch.Notifications,
		},
	})
}
