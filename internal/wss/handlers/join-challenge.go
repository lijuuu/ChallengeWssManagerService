package wsshandler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/lijuuu/ChallengeWssManagerService/internal/config"
	"github.com/lijuuu/ChallengeWssManagerService/internal/model"
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
		return broadcasts.SendErrorWithType(ctx.Conn, wsstypes.JOIN_CHALLENGE, "Internal error", nil)
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		log.Printf("[%s] [JoinChallenge] Unmarshal error: %v", requestID, err)
		return broadcasts.SendErrorWithType(ctx.Conn, wsstypes.JOIN_CHALLENGE, "Invalid payload format", nil)
	}
	log.Printf("[%s] [JoinChallenge] Incoming request from userId %s IP: %s", requestID, payload.UserId, clientIP)

	// auth
	startAuth := time.Now()
	req, err := http.NewRequestWithContext(context.Background(), "GET", config.LoadConfig().APIGatewayTokenCheckURL, nil)
	if err != nil {
		log.Printf("[%s] [JoinChallenge] Auth request create fail: %v", requestID, err)
		return broadcasts.SendErrorWithType(ctx.Conn, wsstypes.JOIN_CHALLENGE, "Internal auth setup error", nil)
	}
	req.Header.Set("Authorization", payload.Token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("[%s] [JoinChallenge] Auth request failed: %v", requestID, err)
		return broadcasts.SendErrorWithType(ctx.Conn, wsstypes.JOIN_CHALLENGE, "Authentication service unreachable", nil)
	}
	defer resp.Body.Close()

	log.Printf("[%s] [JoinChallenge] Auth status: %d (took %v)", requestID, resp.StatusCode, time.Since(startAuth))

	var authResp model.GenericResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		log.Printf("[%s] [JoinChallenge] Decode auth response failed: %v", requestID, err)
		return broadcasts.SendErrorWithType(ctx.Conn, wsstypes.JOIN_CHALLENGE, "Failed to decode authentication", nil)
	}
	if !authResp.Success {
		log.Printf("[%s] [JoinChallenge] Auth failed: %v", requestID, authResp.Error)
		return broadcasts.SendErrorWithType(ctx.Conn, wsstypes.JOIN_CHALLENGE, "Authentication failed", map[string]interface{}{
			"error": authResp.Error,
		})
	}

	var userData AuthPayload
	authPayloadRaw, _ := json.Marshal(authResp.Payload)
	if err := json.Unmarshal(authPayloadRaw, &userData); err != nil {
		log.Printf("[%s] [JoinChallenge] Invalid auth payload structure: %v", requestID, err)
		return broadcasts.SendErrorWithType(ctx.Conn, wsstypes.JOIN_CHALLENGE, "Invalid auth data", nil)
	}

	log.Printf("[%s] [JoinChallenge] Authenticated user ID: %s", requestID, userData.UserID)

	// repo challenge access
	startRepoCheck := time.Now()
	ok, err := ctx.State.Repo.CheckChallengeAccess(context.Background(), payload.ChallengeId, userData.UserID, payload.Password)
	if err != nil {
		log.Printf("[%s] [JoinChallenge] Repo access error: %v", requestID, err)
		return broadcasts.SendErrorWithType(ctx.Conn, wsstypes.JOIN_CHALLENGE, "Challenge access check failed", nil)
	}
	if !ok {
		log.Printf("[%s] [JoinChallenge] Access denied to challenge %s", requestID, payload.ChallengeId)
		return broadcasts.SendErrorWithType(ctx.Conn, wsstypes.JOIN_CHALLENGE, "Invalid challenge ID or password", nil)
	}

	log.Printf("[%s] [JoinChallenge] Access granted for challenge %s (took %v)", requestID, payload.ChallengeId, time.Since(startRepoCheck))

	// memory state
	ctx.State.Mu.Lock()
	defer ctx.State.Mu.Unlock()

	challenge, exists := ctx.State.Challenges[payload.ChallengeId]
	if !exists {
		log.Printf("[WS] Challenge %s not in memory, loading...", payload.ChallengeId)
		challengeDoc, err := ctx.State.Repo.GetChallengeByID(context.Background(), payload.ChallengeId)
		if err != nil {
			log.Printf("[WS] DB load error: %v", err)
			return broadcasts.SendErrorWithType(ctx.Conn, wsstypes.JOIN_CHALLENGE, "Challenge not found", nil)
		}
		challenge = ConvertChallengeDocToChallenge(&challengeDoc)
		ctx.State.Challenges[payload.ChallengeId] = challenge
		log.Printf("[WS] Loaded challenge %s into memory", payload.ChallengeId)
	}

	participant, exists := challenge.Participants[userData.UserID]
	if !exists {
		participant = &model.ParticipantMetadata{
			ProblemsDone:  make(map[string]model.ChallengeProblemMetadata),
			JoinTime:      time.Now().Unix(),
			InitialJoinIP: clientIP,
		}
		challenge.Participants[userData.UserID] = participant
		err := ctx.State.Repo.AddParticipantInJoinPhase(context.Background(), payload.ChallengeId, userData.UserID, participant)
		if err != nil {
			log.Printf("[%s] [JoinChallenge] Failed to persist participant: %v", requestID, err)
		}
		log.Printf("[%s] [JoinChallenge] New participant %s added", requestID, userData.UserID)
	} else {
		log.Printf("[%s] [JoinChallenge] Participant %s rejoined", requestID, userData.UserID)
	}
	participant.LastConnected = time.Now().Unix()

	// ws session + broadcast
	challenge.WSClients[userData.UserID] = ctx.Conn
	broadcasts.BroadcastEntityJoined(challenge, userData.UserID, payload.ChallengeId, userData.UserID == challenge.CreatorID)

	return broadcasts.SendJSON(ctx.Conn, map[string]interface{}{
		"type":    wsstypes.JOIN_CHALLENGE,
		"status":  "success",
		"message": "Joined challenge successfully",
		"payload": map[string]interface{}{
			"userId":      userData.UserID,
			"challengeId": payload.ChallengeId,
			"challenge":   challenge,
		},
	})
}

func ConvertChallengeDocToChallenge(doc *model.ChallengeDocument) *model.Challenge {
	return &model.Challenge{
		ChallengeID:         doc.ChallengeID,
		CreatorID:           doc.CreatorID,
		CreatedAt:           doc.CreatedAt,
		Title:               doc.Title,
		IsPrivate:           doc.IsPrivate,
		Password:            doc.Password,
		Status:              doc.Status,
		TimeLimit:           doc.TimeLimit,
		StartTime:           doc.StartTime,
		Participants:        doc.Participants,
		Submissions:         doc.Submissions,
		Leaderboard:         doc.Leaderboard,
		Config:              doc.Config,
		ProcessedProblemIds: doc.ProcessedProblemIds,

		//these are runtime-only fields, not persisted
		Sessions:  make(map[string]*model.Session),
		WSClients: make(map[string]*websocket.Conn),
		MU:        sync.RWMutex{},
		EventChan: make(chan model.Event, 100),
	}
}
