package wsstypes

import (
	"github.com/gorilla/websocket"
	"github.com/lijuuu/ChallengeWssManagerService/internal/constants"
	"github.com/lijuuu/ChallengeWssManagerService/internal/repo"
	"github.com/lijuuu/ChallengeWssManagerService/internal/state"
)

type State struct {
	Redis              *repo.RedisRepository
	Repo               *repo.MongoRepository
	LocalState         *state.LocalStateManager
	LeaderboardManager LeaderboardService
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
	PING_SERVER = constants.PING_SERVER

	USER_JOINED       = constants.USER_JOINED
	USER_LEFT         = constants.USER_LEFT
	CREATOR_ABANDON   = constants.CREATOR_ABANDON
	CHALLENGE_STARTED = constants.CHALLENGE_STARTED
	OWNER_LEFT        = constants.OWNER_LEFT
	OWNER_JOINED      = constants.OWNER_JOINED

	NEW_OWNER_ASSIGNED = constants.NEW_OWNER_ASSIGNED

	JOIN_CHALLENGE      = constants.JOIN_CHALLENGE
	REFETCH_CHALLENGE   = constants.REFETCH_CHALLENGE
	CURRENT_LEADERBOARD = constants.CURRENT_LEADERBOARD
	LEADERBOARD_UPDATE  = constants.LEADERBOARD_UPDATE
	NEW_SUBMISSION      = constants.NEW_SUBMISSION
)
