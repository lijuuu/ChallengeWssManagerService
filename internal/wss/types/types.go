package wsstypes

import (
	"sync"

	"github.com/gorilla/websocket"
	"github.com/lijuuu/ChallengeWssManagerService/internal/model"
	"github.com/lijuuu/ChallengeWssManagerService/internal/repo"
)

type State struct {
	Repo       repo.MongoRepository
	Challenges map[string]*model.Challenge
	Mu         sync.RWMutex
}

type WsContext struct {
	Conn    *websocket.Conn
	Payload map[string]any
	UserID  string
	State   *State
}

type WsMessage struct {
	Type    string         `json:"type"`
	Payload map[string]any `json:"payload"`
}

type JoinChallengePayload struct {
	UserId      string `json:"userId"`
	Type        string `json:"type"`
	ChallengeId string `json:"challengeId"`
	Password    string `json:"password"`
	Token       string `json:"token"`
}

type RefetchChallengePayload struct {
	UserId      string `json:"userId"`
	Type        string `json:"type"`
	ChallengeId string `json:"challengeId"`
}

type GenericResponse struct {
	Success bool           `json:"success"`
	Status  int            `json:"status"`
	Payload map[string]any `json:"payload"`
	Error   *ErrorInfo     `json:"error"`
}

type ErrorInfo struct {
	ErrorType string `json:"errorType"`
	Code      int    `json:"code"`
	Message   string `json:"message"`
	Details   string `json:"details"`
}

const (
	PING_SERVER = "PING_SERVER"

	USER_JOINED       = "USER_JOINED"
	USER_LEFT         = "USER_LEFT"
	CREATOR_ABANDON   = "CREATOR_ABANDON"
	CHALLENGE_STARTED = "CHALLENGE_STARTED"
	OWNER_LEFT        = "OWNER_LEFT"
	OWNER_JOINED      = "OWNER_JOINED"

	NEW_OWNER_ASSIGNED = "NEW_OWNER_ASSIGNED"

	JOIN_CHALLENGE    = "JOIN_CHALLENGE"
	REFETCH_CHALLENGE = "REFETCH_CHALLENGE"
)
