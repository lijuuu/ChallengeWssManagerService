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

type GetParticipantsDataPayload struct {
	UserId      string `json:"userId"`
	ChallengeId string `json:"challengeId"`
}

// returns only participants (map[userId]ParticipantMetadata)
func GetParticipantsDataHandler(ctx *wsstypes.WsContext) error {
	requestID := uuid.New().String()

	var payload GetParticipantsDataPayload
	raw, err := json.Marshal(ctx.Payload)
	if err != nil {
		log.Printf("[%s] [GetParticipantsData] Marshal error: %v", requestID, err)
		return broadcasts.SendErrorWithType(ctx.Conn, constants.GET_PARTICIPANTS_DATA, "Internal error", nil)
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		log.Printf("[%s] [GetParticipantsData] Unmarshal error: %v", requestID, err)
		return broadcasts.SendErrorWithType(ctx.Conn, constants.GET_PARTICIPANTS_DATA, "Invalid payload format", nil)
	}

	ch, err := ctx.State.Redis.GetChallengeByID(context.Background(), payload.ChallengeId)
	if err != nil {
		return broadcasts.SendErrorWithType(ctx.Conn, constants.GET_PARTICIPANTS_DATA, "Challenge not found", nil)
	}

	return broadcasts.SendJSON(ctx.Conn, map[string]any{
		"type":   constants.GET_PARTICIPANTS_DATA,
		"status": "ok",
		"payload": map[string]any{
			"participants": ch.Participants,
		},
	})
}
