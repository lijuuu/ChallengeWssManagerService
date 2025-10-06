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

type GetChallengeDataPayload struct {
	UserId      string `json:"userId"`
	ChallengeId string `json:"challengeId"`
}

// returns leaderboard + challenge document snapshot in one go (BFF style)
func GetChallengeDataHandler(ctx *wsstypes.WsContext) error {
	requestID := uuid.New().String()

	var payload GetChallengeDataPayload
	raw, err := json.Marshal(ctx.Payload)
	if err != nil {
		log.Printf("[%s] [GetChallengeData] Marshal error: %v", requestID, err)
		return broadcasts.SendErrorWithType(ctx.Conn, constants.GET_CHALLENGE_DATA, "Internal error", nil)
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		log.Printf("[%s] [GetChallengeData] Unmarshal error: %v", requestID, err)
		return broadcasts.SendErrorWithType(ctx.Conn, constants.GET_CHALLENGE_DATA, "Invalid payload format", nil)
	}

	ch, err := ctx.State.Redis.GetChallengeByID(context.Background(), payload.ChallengeId)
	if err != nil {
		return broadcasts.SendErrorWithType(ctx.Conn, constants.GET_CHALLENGE_DATA, "Challenge not found", nil)
	}

	leaderboard, _ := ctx.State.LeaderboardManager.GetLeaderboard(payload.ChallengeId, 100, &ch)

	return broadcasts.SendJSON(ctx.Conn, map[string]any{
		"type":   constants.GET_CHALLENGE_DATA,
		"status": "ok",
		"payload": map[string]any{
			"challenge":   ch,
			"leaderboard": leaderboard,
		},
	})
}
