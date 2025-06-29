package model

import "encoding/json"

type EventType string

const (
	EventChallengeCreated       EventType = "challenge_created"
	EventChallengeDeleted       EventType = "challenge_deleted"
	EventUserJoined             EventType = "user_joined"
	EventUserLeft               EventType = "user_left"
	EventUserForfeited          EventType = "user_forfeited"
	EventProblemSubmitted       EventType = "problem_submitted"
	EventLeaderboardUpdated     EventType = "leaderboard_updated"
	EventChallengeStatusChanged EventType = "challenge_status_changed"
	EventTimeUpdate             EventType = "time_update"
	EventError                  EventType = "error"
)

type Event struct {
	Type    EventType   `json:"type"`
	Payload interface{} `json:"payload"`
}

type ChallengeCreatedPayload struct {
	ChallengeID string `json:"challenge_id"`
	Title       string `json:"title"`
}

type ChallengeDeletedPayload struct {
	ChallengeID string `json:"challenge_id"`
}

type UserJoinedPayload struct {
	UserID string `json:"user_id"`
}

type UserLeftPayload struct {
	UserID string `json:"user_id"`
}

type UserForfeitedPayload struct {
	UserID string `json:"user_id"`
}

type ProblemSubmittedPayload struct {
	UserID    string `json:"user_id"`
	ProblemID string `json:"problem_id"`
	Score     int    `json:"score"`
}

type LeaderboardUpdatedPayload struct {
	Leaderboard []*LeaderboardEntry `json:"leaderboard"`
}

type ChallengeStatusChangedPayload struct {
	ChallengeID string          `json:"challenge_id"`
	Status      ChallengeStatus `json:"status"`
}

type TimeUpdatePayload struct {
	RemainingTime int64 `json:"remaining_time"` // In seconds
}

type ErrorPayload struct {
	Message string `json:"message"`
}

type WebSocketMessage struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}
