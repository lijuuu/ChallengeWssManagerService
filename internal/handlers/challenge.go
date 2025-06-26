package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/lijuuu/ChallengeWssManagerService/internal/models"
	"github.com/lijuuu/ChallengeWssManagerService/internal/service"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		allowedOrigins := []string{
			"http://localhost",
			"http://localhost:3000",
			"https://zenxbattle.space",
			"http://zenxbattle.space",
		}
		for _, o := range allowedOrigins {
			if origin == o {
				return true
			}
		}
		return false
	},
}

// WebSocketHandler handles WebSocket connections
func WebSocketHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	challengeID := vars["challenge_id"]
	userID := r.URL.Query().Get("user_id")
	sessionHash := r.URL.Query().Get("session_hash")

	if challengeID == "" || userID == "" || sessionHash == "" {
		WriteJSONError(w, "Missing required parameters", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Error upgrading to WebSocket: %v", err)
		WriteJSONError(w, "WebSocket upgrade failed", http.StatusInternalServerError)
		return
	}

	if err := service.ConnectWebSocket(challengeID, userID, sessionHash, conn); err != nil {
		log.Printf("Error connecting WebSocket: %v", err)
		conn.Close()
		WriteJSONError(w, err.Error(), http.StatusUnauthorized)
		return
	}
}

// CreateChallengeHandler handles challenge creation
func CreateChallengeHandler(w http.ResponseWriter, r *http.Request) {
	var req models.CreateChallengeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteJSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.UserID == "" {
		WriteJSONError(w, "Missing user_id", http.StatusBadRequest)
		return
	}

	config := models.ChallengeConfig{
		MaxUsers:           req.MaxUsers,
		MaxEasyQuestions:   req.MaxEasyQuestions,
		MaxMediumQuestions: req.MaxMediumQuestions,
		MaxHardQuestions:   req.MaxHardQuestions,
		RandomQuestionPool: service.ValidProblems(),
		InitialQuestions:   req.InitialQuestions,
	}

	challengeID := service.GenerateChallengeID()
	c, err := service.NewChallenge(challengeID, req.UserID, req.Title, req.IsPrivate, req.Password, req.TimeLimit, config)
	if err != nil {
		WriteJSONError(w, err.Error(), http.StatusBadRequest)
		return
	}

	sessionHash, err := service.JoinChallenge(challengeID, req.UserID, req.Password, "")
	if err != nil {
		WriteJSONError(w, err.Error(), http.StatusBadRequest)
		return
	}

	response := map[string]interface{}{
		"challenge_id": challengeID,
		"session_hash": sessionHash,
		"title":        c.Challenge.Title,
		"status":       c.Challenge.Status,
		"problem_ids":  c.Challenge.ProblemIDs,
	}
	WriteJSONResponse(w, response, http.StatusCreated)
}

// ListChallengesHandler lists open challenges
func ListChallengesHandler(w http.ResponseWriter, r *http.Request) {
	challenges, err := service.ListOpenChallenges()
	if err != nil {
		WriteJSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := make([]map[string]interface{}, len(challenges))
	for i, c := range challenges {
		response[i] = map[string]interface{}{
			"challenge_id": c.Challenge.ChallengeID,
			"title":        c.Challenge.Title,
			"status":       c.Challenge.Status,
			"participants": len(c.Challenge.Participants),
			"time_limit":   int(c.Challenge.TimeLimit.Minutes()),
		}
	}

	WriteJSONResponse(w, response, http.StatusOK)
}

// GetChallengeHandler gets challenge details
func GetChallengeHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	challengeID := vars["challenge_id"]

	c, ok := service.GetChallenge(challengeID)
	if !ok {
		WriteJSONError(w, "Challenge not found", http.StatusNotFound)
		return
	}

	c.Challenge.MU.RLock()
	defer c.Challenge.MU.RUnlock()

	remainingTime := int64(0)
	if c.Challenge.Status == models.StatusStarted {
		remaining := c.Challenge.TimeLimit - time.Since(c.Challenge.StartTime)
		if remaining > 0 {
			remainingTime = int64(remaining.Seconds())
		}
	}

	response := map[string]interface{}{
		"challenge_id":   c.Challenge.ChallengeID,
		"title":          c.Challenge.Title,
		"status":         c.Challenge.Status,
		"is_private":     c.Challenge.IsPrivate,
		"time_limit":     int(c.Challenge.TimeLimit.Minutes()),
		"remaining_time": remainingTime,
		"participants":   len(c.Challenge.Participants),
		"leaderboard":    c.Challenge.Leaderboard,
		"problem_ids":    c.Challenge.ProblemIDs,
	}
	WriteJSONResponse(w, response, http.StatusOK)
}
