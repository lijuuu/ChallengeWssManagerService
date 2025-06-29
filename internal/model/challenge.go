package model

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
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
	ChallengeID  string                          `gorm:"column:challenge_id;primaryKey" json:"challenge_id"`
	CreatorID    string                          `gorm:"column:creator_id" json:"creator_id"`
	Title        string                          `gorm:"column:title" json:"title"`
	IsPrivate    bool                            `gorm:"column:is_private" json:"is_private"`
	Password     string                          `gorm:"column:password" json:"password"`
	Status       ChallengeStatus                 `gorm:"column:status" json:"status"`
	ProblemIDs   []string                        `gorm:"column:problem_ids;type:jsonb" json:"problem_ids"`
	TimeLimit    time.Duration                   `gorm:"column:time_limit" json:"time_limit"`
	StartTime    time.Time                       `gorm:"column:start_time" json:"start_time"`
	Participants map[string]*ParticipantMetadata `gorm:"column:participants;type:jsonb" json:"participants"` // Store as JSONB
	Submissions  map[string]map[string]struct {
		SubmissionID string        `json:"submission_id"`
		TimeTaken    time.Duration `json:"time_taken"`
		Points       int           `json:"points"`
	} `gorm:"column:submissions;type:jsonb" json:"submissions"` // Still ignored by GORM unless you want to persist
	Leaderboard []*LeaderboardEntry        `gorm:"-" json:"leaderboard"` // Ignored by GORM
	Sessions    map[string]*Session        `gorm:"-" json:"sessions"`    // Ignored by GORM
	Config      ChallengeConfig            `gorm:"-" json:"config"`      // Ignored by GORM
	WSClients   map[string]*websocket.Conn `gorm:"-" json:"ws_clients"`  // Ignored by GORM
	MU          sync.RWMutex               `gorm:"-" json:"-"`
	EventChan   chan Event                 `gorm:"-" json:"-"`
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
