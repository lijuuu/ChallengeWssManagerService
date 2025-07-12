package wsshandler

import (
	"encoding/json"
	"log"

	"github.com/google/uuid"
	"github.com/lijuuu/ChallengeWssManagerService/internal/wss/broadcasts"
	wsstypes "github.com/lijuuu/ChallengeWssManagerService/internal/wss/types"
)

func RefetchChallenge(ctx *wsstypes.WsContext) error {
	requestID := uuid.New().String()

	var payload wsstypes.RefetchChallengePayload
	raw, err := json.Marshal(ctx.Payload)
	if err != nil {
		log.Printf("[%s] [RefetchChallenge] Marshal error: %v", requestID, err)
		return broadcasts.SendJSON(ctx.Conn, map[string]interface{}{
			"type":   wsstypes.REFETCH_CHALLENGE,
			"status": "error",
			"error": map[string]interface{}{
				"code":    "MARSHAL_ERROR",
				"message": "Internal server error",
			},
		})
	}

	if err := json.Unmarshal(raw, &payload); err != nil {
		log.Printf("[%s] [RefetchChallenge] Unmarshal error: %v", requestID, err)
		return broadcasts.SendJSON(ctx.Conn, map[string]interface{}{
			"type":   wsstypes.REFETCH_CHALLENGE,
			"status": "error",
			"error": map[string]interface{}{
				"code":    "INVALID_PAYLOAD",
				"message": "Payload format invalid",
			},
		})
	}

	challenge, ok := ctx.State.Challenges[payload.ChallengeId]
	if !ok {
		log.Printf("[%s] [RefetchChallenge] Challenge %s not found in memory", requestID, payload.ChallengeId)
		return broadcasts.SendJSON(ctx.Conn, map[string]interface{}{
			"type":   wsstypes.REFETCH_CHALLENGE,
			"status": "error",
			"error": map[string]interface{}{
				"code":    "CHALLENGE_NOT_FOUND",
				"message": "Challenge not found or not joined",
			},
		})
	}

	log.Printf("[%s] [RefetchChallenge] Sending latest challenge state to user %s", requestID, payload.UserId)

	return broadcasts.SendJSON(ctx.Conn, map[string]interface{}{
		"type":    wsstypes.REFETCH_CHALLENGE,
		"status":  "ok",
		"message": "Challenge state fetched successfully",
		"payload": map[string]interface{}{
			"userId":      payload.UserId,
			"challengeId": payload.ChallengeId,
			"challenge":   challenge,
		},
	})
}
