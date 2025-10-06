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

type GetChallengeMinPayload struct {
	UserId      string `json:"userId"`
	ChallengeId string `json:"challengeId"`
}

// returns a minimal subset of challenge document (no participants, no leaderboard)
func GetChallengeMinHandler(ctx *wsstypes.WsContext) error {
	requestID := uuid.New().String()

	var payload GetChallengeMinPayload
	raw, err := json.Marshal(ctx.Payload)
	if err != nil {
		log.Printf("[%s] [GetChallengeMin] Marshal error: %v", requestID, err)
		return broadcasts.SendErrorWithType(ctx.Conn, constants.GET_CHALLENGE_MIN, "Internal error", nil)
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		log.Printf("[%s] [GetChallengeMin] Unmarshal error: %v", requestID, err)
		return broadcasts.SendErrorWithType(ctx.Conn, constants.GET_CHALLENGE_MIN, "Invalid payload format", nil)
	}

	ch, err := ctx.State.Redis.GetChallengeByID(context.Background(), payload.ChallengeId)
	if err != nil {
		return broadcasts.SendErrorWithType(ctx.Conn, constants.GET_CHALLENGE_MIN, "Challenge not found", nil)
	}

	// strip fields to keep payload lightweight
	min := map[string]any{
		"challengeId": ch.ChallengeID,
		"title":       ch.Title,
		"status":      ch.Status,
		"isPrivate":   ch.IsPrivate,
		"createdAt":   ch.CreatedAt,
		"startTime":   ch.StartTime,
		"timeLimit":   ch.TimeLimit,
	}

	return broadcasts.SendJSON(ctx.Conn, map[string]any{
		"type":    constants.GET_CHALLENGE_MIN,
		"status":  "ok",
		"payload": min,
	})
}
