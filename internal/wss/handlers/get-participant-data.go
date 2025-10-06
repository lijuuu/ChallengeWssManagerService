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

type GetParticipantDataPayload struct {
	UserId      string `json:"userId"`
	ChallengeId string `json:"challengeId"`
}

// returns participant-centric data (metadata + personal submissions summary)
func GetParticipantDataHandler(ctx *wsstypes.WsContext) error {
	requestID := uuid.New().String()

	var payload GetParticipantDataPayload
	raw, err := json.Marshal(ctx.Payload)
	if err != nil {
		log.Printf("[%s] [GetParticipantData] Marshal error: %v", requestID, err)
		return broadcasts.SendErrorWithType(ctx.Conn, constants.GET_PARTICIPANT_DATA, "Internal error", nil)
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		log.Printf("[%s] [GetParticipantData] Unmarshal error: %v", requestID, err)
		return broadcasts.SendErrorWithType(ctx.Conn, constants.GET_PARTICIPANT_DATA, "Invalid payload format", nil)
	}

	ch, err := ctx.State.Redis.GetChallengeByID(context.Background(), payload.ChallengeId)
	if err != nil {
		return broadcasts.SendErrorWithType(ctx.Conn, constants.GET_PARTICIPANT_DATA, "Challenge not found", nil)
	}

	meta, ok := ch.Participants[payload.UserId]
	if !ok {
		return broadcasts.SendErrorWithType(ctx.Conn, constants.GET_PARTICIPANT_DATA, "User not a participant", nil)
	}
	var userSubs map[string]any
	if ch.Submissions != nil {
		if s, exists := ch.Submissions[payload.UserId]; exists {
			// convert to simple map for transport
			userSubs = map[string]any{}
			for pid, sub := range s {
				userSubs[pid] = map[string]any{"submissionId": sub.SubmissionID, "points": sub.Points, "timeTakenMillis": int64(sub.TimeTaken.Milliseconds())}
			}
		}
	}

	return broadcasts.SendJSON(ctx.Conn, map[string]any{
		"type":   constants.GET_PARTICIPANT_DATA,
		"status": "ok",
		"payload": map[string]any{
			"metadata":    meta,
			"submissions": userSubs,
		},
	})
}
