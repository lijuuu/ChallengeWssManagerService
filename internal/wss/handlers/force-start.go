package wsshandler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/lijuuu/ChallengeWssManagerService/internal/constants"
	"github.com/lijuuu/ChallengeWssManagerService/internal/model"
	"github.com/lijuuu/ChallengeWssManagerService/internal/wss/broadcasts"
	wsstypes "github.com/lijuuu/ChallengeWssManagerService/internal/wss/types"
)

// ForceStartHandler allows the challenge owner to start the challenge immediately
func ForceStartHandler(ctx *wsstypes.WsContext) error {
	var payload wsstypes.ForceStartPayload
	raw, err := json.Marshal(ctx.Payload)
	if err != nil {
		log.Printf("[FORCE_START] marshal payload failed: %v", err)
		return broadcasts.SendErrorWithType(ctx.Conn, constants.WS_CHALLENGE_STARTED, "Internal error", nil)
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		log.Printf("[FORCE_START] unmarshal payload failed: %v", err)
		return broadcasts.SendErrorWithType(ctx.Conn, constants.WS_CHALLENGE_STARTED, "Invalid payload format", nil)
	}

	ch, err := ctx.State.Redis.GetChallengeByID(context.Background(), payload.ChallengeId)
	if err != nil {
		return broadcasts.SendErrorWithType(ctx.Conn, constants.WS_CHALLENGE_STARTED, "Challenge not found", nil)
	}

	if ch.CreatorID != payload.UserId {
		return broadcasts.SendErrorWithType(ctx.Conn, constants.WS_CHALLENGE_STARTED, "Only owner can start the challenge", nil)
	}

	if ch.Status == model.ChallengeStarted {
		return broadcasts.SendStandardSuccess(ctx.Conn, constants.WS_CHALLENGE_STARTED, map[string]any{"message": "already started"})
	}

	// set startTime to now if not set; transition to started
	now := time.Now().Unix()
	if ch.StartTime == 0 || now < ch.StartTime {
		ch.StartTime = now
	}
	ch.Status = model.ChallengeStarted
	if err := ctx.State.Redis.UpdateChallenge(context.Background(), &ch); err != nil {
		return broadcasts.SendErrorWithType(ctx.Conn, constants.WS_CHALLENGE_STARTED, "Failed to update challenge", nil)
	}

	// Broadcast start and schedule finish from startTime
	wsClients := ctx.State.LocalState.GetAllWSClients(payload.ChallengeId)
	broadcasts.BroadcastGameStarted(wsClients, payload.ChallengeId, ch.StartTime)
	// Persist and broadcast system notification
	_ = ctx.State.Mongo.AppendNotification(context.Background(), payload.ChallengeId, model.Notification{
		Type:    "SYSTEM",
		Message: "Challenge has started (forced)",
		Time:    time.Now().Unix(),
	})
	broadcasts.BroadcastStandardMessage(wsClients, constants.PUSH_NEW_NOTIFICATION, model.Notification{
		Type:    "SYSTEM",
		Message: "Challenge has started (forced)",
		Time:    time.Now().Unix(),
	}, true, nil)

	endAt := time.Unix(ch.StartTime, 0).Add(time.Duration(ch.TimeLimit) * time.Millisecond)
	dur := time.Until(endAt)
	if dur < 0 {
		dur = 0
	}
	ctx.State.ServiceRef.ScheduleChallengeFinish(ch.ChallengeID, dur)

	fmt.Println("challenge during forcstart ",ch.Participants)

	return broadcasts.SendStandardSuccess(ctx.Conn, constants.WS_CHALLENGE_STARTED, map[string]any{"message": "started"})
}
