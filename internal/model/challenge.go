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
	LastActive  int64
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
	ChallengeID  string                           `gorm:"column:challenge_id;primaryKey" json:"challenge_id"`
	CreatorID    string                           `gorm:"column:creator_id" json:"creator_id"`
	Title        string                           `gorm:"column:title" json:"title"`
	IsPrivate    bool                             `gorm:"column:is_private" json:"is_private"`
	Password     string                           `gorm:"column:password" json:"password"`
	Status       ChallengeStatus                  `gorm:"column:status" json:"status"`
	TimeLimit    int64                            `gorm:"column:time_limit" json:"time_limit"`
	StartTime    int64                            `gorm:"column:start_time" json:"start_time"`
	Participants map[string]*ParticipantMetadata  `gorm:"column:participants;type:jsonb" json:"participants"` // Store as JSONB
	Submissions  map[string]map[string]Submission `gorm:"column:submissions;type:jsonb" json:"submissions"`   // Store as JSONB
	Leaderboard  []*LeaderboardEntry              `gorm:"column:leaderboard;type:jsonb" json:"leaderboard"`   // Store as JSONB
	Config       *ChallengeConfig                 `gorm:"column:config;type:jsonb" json:"config"`             // Store as JSONB

	Sessions  map[string]*Session        `gorm:"-" json:"sessions"`   // Ignored
	WSClients map[string]*websocket.Conn `gorm:"-" json:"ws_clients"` // Still ignored (cannot persist websocket.Conn)
	MU        sync.RWMutex               `gorm:"-" json:"-"`
	EventChan chan Event                 `gorm:"-" json:"-"`
}

type Submission struct {
	SubmissionID string
	TimeTaken    time.Duration
	Points       int
	UserCode     string
}

type ParticipantMetadata struct {
	UserID            string
	ProblemsDone      map[string]ChallengeProblemMetadata
	ProblemsAttempted int
	TotalScore        int
	JoinTime          int64
	LastConnected     int64
}

type ChallengeProblemMetadata struct {
	ProblemID   string
	Score       int
	TimeTaken   int64
	CompletedAt int64
}

type LeaderboardEntry struct {
	UserID            string
	ProblemsCompleted int
	TotalScore        int
	Rank              int
}
