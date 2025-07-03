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

const ChallengeOpen = "CHALLENGEOPEN"
const ChallengeClose = "CHALLENGECLOSE"
const ChallengeStarted = "CHALLENGESTARTED"
const ChallengeForfieted = "CHALLENGEFORFIETED"
const ChallengeEnded = "CHALLENGEENDED"
const ChallengeAbandon = "CHALLENGEABANDON"

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
	MaxUsers           int // Maximum number of participants
	MaxEasyQuestions   int // Max easy questions
	MaxMediumQuestions int // Max medium questions
	MaxHardQuestions   int // Max hard questions
}

type Challenge struct {
	ChallengeID  string                           `bson:"challengeId" json:"challengeId"`
	CreatorID    string                           `bson:"creatorId" json:"creatorId"`
	CreatedAt    int64                            `bson:"createdAt" json:"createdAt"`
	Title        string                           `bson:"title" json:"title"`
	IsPrivate    bool                             `bson:"isPrivate" json:"isPrivate"`
	Password     string                           `bson:"password" json:"password"`
	Status       ChallengeStatus                  `bson:"status" json:"status"`
	TimeLimit    int64                            `bson:"timeLimit" json:"timeLimit"`
	StartTime    int64                            `bson:"startTime" json:"startTime"`
	Participants map[string]*ParticipantMetadata  `bson:"participants" json:"participants"`
	Submissions  map[string]map[string]Submission `bson:"submissions" json:"submissions"`
	Leaderboard  []*LeaderboardEntry              `bson:"leaderboard" json:"leaderboard"`
	Config       *ChallengeConfig                 `bson:"config" json:"config"`

	Sessions  map[string]*Session        `bson:"-" json:"sessions"`  // Ignored by MongoDB
	WSClients map[string]*websocket.Conn `bson:"-" json:"wsClients"` // Ignored by MongoDB
	MU        sync.RWMutex               `bson:"-" json:"-"`
	EventChan chan Event                 `bson:"-" json:"-"`
}

type ChallengeDocument struct { //for mongo
	ChallengeID         string                           `bson:"challengeId" json:"challengeId"`
	CreatorID           string                           `bson:"creatorId" json:"creatorId"`
	CreatedAt           int64                            `bson:"createdAt" json:"createdAt"`
	Title               string                           `bson:"title" json:"title"`
	IsPrivate           bool                             `bson:"isPrivate" json:"isPrivate"`
	Password            string                           `bson:"password" json:"password"`
	Status              ChallengeStatus                  `bson:"status" json:"status"`
	TimeLimit           int64                            `bson:"timeLimit" json:"timeLimit"`
	StartTime           int64                            `bson:"startTime" json:"startTime"`
	Participants        map[string]*ParticipantMetadata  `bson:"participants" json:"participants"`
	Submissions         map[string]map[string]Submission `bson:"submissions" json:"submissions"`
	Leaderboard         []*LeaderboardEntry              `bson:"leaderboard" json:"leaderboard"`
	Config              *ChallengeConfig                 `bson:"config" json:"config"`
	ProcessedProblemIds []string                         `bson:"processedProblemIds" json:"processedProblemIds"`
	ProblemCount        int64                            `bson:"problemCount" json:"problemCount"`
}

type Submission struct {
	SubmissionID string
	TimeTaken    time.Duration
	Points       int
	UserCode     string
}

type ParticipantMetadata struct {
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
