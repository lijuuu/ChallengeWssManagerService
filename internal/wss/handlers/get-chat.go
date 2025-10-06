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

type GetChatPayload struct {
	UserId      string `json:"userId"`
	ChallengeId string `json:"challengeId"`
}

func GetChatHandler(ctx *wsstypes.WsContext) error {
	requestID := uuid.New().String()

	var payload GetChatPayload
	raw, err := json.Marshal(ctx.Payload)
	if err != nil {
		log.Printf("[%s] [GetChat] Marshal error: %v", requestID, err)
		return broadcasts.SendErrorWithType(ctx.Conn, constants.WHOLE_CHAT, "Internal error", nil)
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		log.Printf("[%s] [GetChat] Unmarshal error: %v", requestID, err)
		return broadcasts.SendErrorWithType(ctx.Conn, constants.WHOLE_CHAT, "Invalid payload format", nil)
	}

	ch, err := ctx.State.Redis.GetChallengeByID(context.Background(), payload.ChallengeId)
	if err != nil {
		return broadcasts.SendErrorWithType(ctx.Conn, constants.WHOLE_CHAT, "Challenge not found", nil)
	}

	return broadcasts.SendJSON(ctx.Conn, map[string]any{
		"type":   constants.WHOLE_CHAT,
		"status": "ok",
		"payload": map[string]any{
			"chat": ch.Chat,
		},
	})
}
