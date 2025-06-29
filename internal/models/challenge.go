package models

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	SessionHashKey        = "someone"
	SessionTimeout        = 30 * time.Minute
	CleanupInterval       = 5 * time.Minute
	MaxConcurrentMatches  = 100
	WebsocketReadTimeout  = 60 * time.Second
	EmptyChallengeTimeout = 10 * time.Minute
)

type ChallengeStatus string

const (
	StatusWaiting   ChallengeStatus = "waiting"
	StatusStarted   ChallengeStatus = "started"
	StatusFinished  ChallengeStatus = "finished"
	StatusCancelled ChallengeStatus = "cancelled"
)

type QuestionDifficulty string

const (
	DifficultyEasy   QuestionDifficulty = "easy"
	DifficultyMedium QuestionDifficulty = "medium"
	DifficultyHard   QuestionDifficulty = "hard"
)

type Session struct {
	UserID      string
	ChallengeID string
	LastActive  time.Time
	SessionHash string
}

type ChallengeConfig struct {
	MaxUsers           int                             // Maximum number of participants
	MaxEasyQuestions   int                             // Max easy questions
	MaxMediumQuestions int                             // Max medium questions
	MaxHardQuestions   int                             // Max hard questions
	RandomQuestionPool map[QuestionDifficulty][]string // Pool of random question IDs by difficulty
	InitialQuestions   map[QuestionDifficulty][]string // Initial user-provided questions
}

type Challenge struct {
	ChallengeID  string
	CreatorID    string
	Title        string
	IsPrivate    bool
	Password     string
	Status       ChallengeStatus
	ProblemIDs   []string
	TimeLimit    time.Duration
	StartTime    time.Time
	Participants map[string]*ParticipantMetadata
	// Submissions maps userID -> problemID -> struct{ SubmissionID string; Points int }
	Submissions map[string]map[string]struct {
		SubmissionID string
		TimeTaken time.Duration
		Points       int
	}
	Leaderboard []*LeaderboardEntry
	Sessions    map[string]*Session
	Config      ChallengeConfig
	WSClients   map[string]*websocket.Conn
	MU          sync.RWMutex
	EventChan   chan Event
}

type ParticipantMetadata struct {
	UserID            string
	ProblemsDone      map[string]ChallengeProblemMetadata
	ProblemsAttempted int
	TotalScore        int
	LastConnected     time.Time
}

type ChallengeProblemMetadata struct {
	ProblemID   string
	Score       int
	TimeTaken   int64
	CompletedAt time.Time
}

type LeaderboardEntry struct {
	UserID            string
	ProblemsCompleted int
	TotalScore        int
	Rank              int
}

type CreateChallengeRequest struct {
	UserID             string                          `json:"user_id"`
	Title              string                          `json:"title"`
	IsPrivate          bool                            `json:"is_private"`
	Password           string                          `json:"password"`
	TimeLimit          int                             `json:"time_limit"`
	MaxUsers           int                             `json:"max_users"`
	MaxEasyQuestions   int                             `json:"max_easy_questions"`
	MaxMediumQuestions int                             `json:"max_medium_questions"`
	MaxHardQuestions   int                             `json:"max_hard_questions"`
	InitialQuestions   map[QuestionDifficulty][]string `json:"initial_questions"`
}

type JoinChallengeRequest struct {
	UserID      string `json:"user_id"`
	Password    string `json:"password"`
	SessionHash string `json:"session_hash"`
}

type StartChallengeRequest struct {
	UserID      string `json:"user_id"`
	SessionHash string `json:"session_hash"`
}

type EndChallengeRequest struct {
	UserID      string `json:"user_id"`
	SessionHash string `json:"session_hash"`
}

type DeleteChallengeRequest struct {
	UserID      string `json:"user_id"`
	SessionHash string `json:"session_hash"`
}

type SubmitProblemRequest struct {
	ProblemID   string `json:"problem_id"`
	Score       int    `json:"score"`
	SessionHash string `json:"session_hash"`
}

type ForfeitRequest struct {
	UserID      string `json:"user_id"`
	SessionHash string `json:"session_hash"`
}
