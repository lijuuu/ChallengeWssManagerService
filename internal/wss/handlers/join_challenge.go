package wsshandler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/lijuuu/ChallengeWssManagerService/internal/config"
	"github.com/lijuuu/ChallengeWssManagerService/internal/constants"
	"github.com/lijuuu/ChallengeWssManagerService/internal/model"
	"github.com/lijuuu/ChallengeWssManagerService/internal/utils"
	"github.com/lijuuu/ChallengeWssManagerService/internal/wss/broadcasts"
	wsstypes "github.com/lijuuu/ChallengeWssManagerService/internal/wss/types"
)

type AuthPayload struct {
	UserID string `json:"userId"`
}

func JoinChallengeHandler(ctx *wsstypes.WsContext) error {
	requestID := uuid.New().String()
	clientIP := ctx.Conn.RemoteAddr().String()

	var payload wsstypes.JoinChallengePayload
	raw, err := json.Marshal(ctx.Payload)
	if err != nil {
		log.Printf("[%s] [JoinChallenge] Marshal error: %v", requestID, err)
		return utils.CloseConnectionWithError(ctx.Conn, wsstypes.JOIN_CHALLENGE, "Internal error")
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		log.Printf("[%s] [JoinChallenge] Unmarshal error: %v", requestID, err)
		return utils.CloseConnectionWithError(ctx.Conn, wsstypes.JOIN_CHALLENGE, "Invalid payload format")
	}
	log.Printf("[%s] [JoinChallenge] Incoming request from userId %s IP: %s", requestID, payload.UserId, clientIP)

	//verify the bearer token with the api gateway.
	startAuth := time.Now()
	req, err := http.NewRequestWithContext(context.Background(), "GET", config.LoadConfig().APIGatewayTokenCheckURL, nil)
	if err != nil {
		log.Printf("[%s] [JoinChallenge] Auth request create fail: %v", requestID, err)
		return utils.CloseConnectionWithError(ctx.Conn, wsstypes.JOIN_CHALLENGE, "Internal auth setup error")
	}
	req.Header.Set("Authorization", payload.Token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("[%s] [JoinChallenge] Auth request failed: %v", requestID, err)
		return utils.CloseConnectionWithError(ctx.Conn, wsstypes.JOIN_CHALLENGE, "Authentication service unreachable")
	}
	defer resp.Body.Close()

	log.Printf("[%s] [JoinChallenge] Auth status: %d (took %v)", requestID, resp.StatusCode, time.Since(startAuth))

	var authResp model.GenericResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		log.Printf("[%s] [JoinChallenge] Decode auth response failed: %v", requestID, err)
		return utils.CloseConnectionWithError(ctx.Conn, wsstypes.JOIN_CHALLENGE, "Failed to decode authentication")
	}
	if !authResp.Success {
		log.Printf("[%s] [JoinChallenge] Auth failed: %v", requestID, authResp.Error)
		return utils.CloseUnauthorizedConnection(ctx.Conn, wsstypes.JOIN_CHALLENGE, "Authentication failed", map[string]interface{}{
			"error": authResp.Error,
		})
	}

	var userData AuthPayload
	authPayloadRaw, _ := json.Marshal(authResp.Payload)
	if err := json.Unmarshal(authPayloadRaw, &userData); err != nil {
		log.Printf("[%s] [JoinChallenge] Invalid auth payload structure: %v", requestID, err)
		return utils.CloseConnectionWithError(ctx.Conn, wsstypes.JOIN_CHALLENGE, "Invalid auth data")
	}

	log.Printf("[%s] [JoinChallenge] Authenticated user ID: %s", requestID, userData.UserID)

	//load the challenge from redis.
	startRepoCheck := time.Now()
	challengeDoc, err := ctx.State.Redis.GetChallengeByID(context.Background(), payload.ChallengeId)
	if err != nil {
		log.Printf("[%s] [JoinChallenge] Challenge not found in Redis: %v", requestID, err)
		return utils.CloseConnectionWithError(ctx.Conn, wsstypes.JOIN_CHALLENGE, "Challenge not found")
	}

	//just checking scenarios- assuming whether the challenge is live (in JOin or started phase)
	if challengeDoc.Status == model.ChallengeAbandon || challengeDoc.Status == model.ChallengeEnded || utils.IsChallengeExpired(challengeDoc) {
		fmt.Println("challengeDoc ", challengeDoc)
		fmt.Println("challenge is expired, ", utils.IsChallengeExpired(challengeDoc))
		return utils.CloseConnectionWithError(ctx.Conn, wsstypes.JOIN_CHALLENGE, "Challenge is abandoned")
	}

	//if the challenge is private, require the correct password.
	if challengeDoc.IsPrivate && challengeDoc.Password != payload.Password {
		log.Printf("[%s] [JoinChallenge] Access denied to challenge %s", requestID, payload.ChallengeId)
		return utils.CloseUnauthorizedConnection(ctx.Conn, wsstypes.JOIN_CHALLENGE, "Invalid challenge ID or password", map[string]any{
			"challengeId": payload.ChallengeId,
			"reason":      "invalid_password",
		})
	}

	log.Printf("[%s] [JoinChallenge] Access granted for challenge %s (took %v)", requestID, payload.ChallengeId, time.Since(startRepoCheck))

	// Check max participants limit from Redis data
	if len(challengeDoc.Participants) >= challengeDoc.MaxParticipants {
		log.Printf("[%s] [JoinChallenge] Challenge at capacity: %d/%d participants", requestID, len(challengeDoc.Participants), challengeDoc.MaxParticipants)
		return utils.CloseConnectionWithError(ctx.Conn, wsstypes.JOIN_CHALLENGE, "Challenge is at maximum capacity")
	}

	// If challenge already started, block new joins (allow only existing participants)
	fmt.Println("challengeDoc ", challengeDoc.Participants)

	if challengeDoc.Status == model.ChallengeStarted {
		if _, ok := challengeDoc.Participants[userData.UserID]; !ok {
			log.Printf("[%s] [JoinChallenge] Unauthorized access attempt: user %s not in participants for started challenge %s", requestID, userData.UserID, payload.ChallengeId)
			return utils.CloseUnauthorizedConnection(ctx.Conn, wsstypes.JOIN_CHALLENGE, "Challenge already started; cannot join", map[string]any{
				"challengeId": payload.ChallengeId,
				"userId":      userData.UserID,
				"reason":      "not_participant",
			})
		}
	}

	//add or refresh the participant in redis.
	participant, exists := challengeDoc.Participants[userData.UserID]
	if !exists {
		participant = &model.ParticipantMetadata{
			ProblemsDone:  make(map[string]model.ChallengeProblemMetadata),
			JoinTime:      time.Now().Unix(),
			InitialJoinIP: clientIP,
		}
		err := ctx.State.Redis.AddParticipant(context.Background(), payload.ChallengeId, userData.UserID, participant)
		if err != nil {
			log.Printf("[%s] [JoinChallenge] Failed to persist participant: %v", requestID, err)
		}
		log.Printf("[%s] [JoinChallenge] New participant %s added", requestID, userData.UserID)
	} else {
		log.Printf("[%s] [JoinChallenge] Participant %s rejoined", requestID, userData.UserID)
	}
	participant.LastConnected = time.Now().Unix()

	//persist participant changes in redis.
	err = ctx.State.Redis.UpdateParticipant(context.Background(), payload.ChallengeId, userData.UserID, participant)
	if err != nil {
		log.Printf("[%s] [JoinChallenge] Failed to update participant: %v", requestID, err)
	}

	//track this websocket connection in local state.
	ctx.State.LocalState.AddWSClient(payload.ChallengeId, userData.UserID, ctx.Conn)

	wsClients := ctx.State.LocalState.GetAllWSClients(payload.ChallengeId)
	//notify other clients in the same challenge.
	broadcasts.BroadcastEntityJoinedWithClients(wsClients, userData.UserID, payload.ChallengeId, userData.UserID == challengeDoc.CreatorID)

	// if lobby is active, optionally send lobby info to the joining client
	lobbyActive := challengeDoc.StartTime > 0 && time.Now().Before(time.Unix(challengeDoc.StartTime, 0))
	if lobbyActive {
		_ = broadcasts.SendJSON(ctx.Conn, map[string]any{
			"type":   constants.GET_CHALLENGE_MIN,
			"status": "ok",
			"payload": map[string]any{
				"challengeId": challengeDoc.ChallengeID,
				"lobby": map[string]any{
					"active":    true,
					"countdown": time.Unix(challengeDoc.StartTime, 0).Unix() - time.Now().Unix(),
				},
			},
		})
	}

	// token lifetime should cover from now through end time (start + limit) plus buffer
	var tokenTTL time.Duration
	if challengeDoc.StartTime > 0 {
		endAt := time.Unix(challengeDoc.StartTime, 0).Add(time.Duration(challengeDoc.TimeLimit) * time.Millisecond)
		dur := time.Until(endAt)
		if dur < 0 {
			dur = 0
		}
		tokenTTL = dur + constants.BufferTime
	} else {
		tokenTTL = time.Duration(challengeDoc.TimeLimit) + constants.BufferTime
	}

	challengeToken, _ := ctx.State.JwtManager.GenerateToken(payload.UserId, payload.ChallengeId, tokenTTL)

	return broadcasts.SendJSON(ctx.Conn, map[string]interface{}{
		"type":    wsstypes.JOIN_CHALLENGE,
		"status":  "success",
		"message": "Joined challenge successfully",
		"payload": map[string]interface{}{
			"userId":         userData.UserID,
			"challengeId":    payload.ChallengeId,
			"challenge":      challengeDoc,
			"challengeToken": challengeToken,
		},
	})
}
