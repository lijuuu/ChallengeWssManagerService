package service

import (
	"fmt"
	"sync"

	"github.com/lijuuu/ChallengeWssManagerService/internal/model"
	redisboard "github.com/lijuuu/RedisBoard"
)

// LeaderboardService defines the interface for leaderboard operations
type LeaderboardService interface {
	// Initialize leaderboard for a challenge
	InitializeLeaderboard(challengeID string) error

	// Update participant score after submission
	UpdateParticipantScore(challengeID, userID string, points int) error

	// Get current leaderboard for a challenge
	GetLeaderboard(challengeID string, limit int) ([]*model.LeaderboardEntry, error)

	// Get participant's rank and data
	GetParticipantRank(challengeID, userID string) (*ParticipantLeaderboardData, error)

	// Clean up leaderboard when challenge ends
	CleanupLeaderboard(challengeID string) error
}

// ParticipantLeaderboardData holds user rank information
type ParticipantLeaderboardData struct {
	UserID     string `json:"userId"`
	TotalScore int    `json:"totalScore"`
	GlobalRank int    `json:"globalRank"`
	Rank       int    `json:"rank"` // Alias for GlobalRank for compatibility
}

// WebSocket event payload structs
type LeaderboardUpdateEvent struct {
	Type        string                    `json:"type"`
	ChallengeID string                    `json:"challengeId"`
	Leaderboard []*model.LeaderboardEntry `json:"leaderboard"`
	UpdatedUser string                    `json:"updatedUser"`
}

type NewSubmissionEvent struct {
	Type        string `json:"type"`
	ChallengeID string `json:"challengeId"`
	UserID      string `json:"userId"`
	ProblemID   string `json:"problemId"`
	Points      int    `json:"points"`
	NewRank     int    `json:"newRank"`
}

type CurrentLeaderboardEvent struct {
	Type        string                    `json:"type"`
	ChallengeID string                    `json:"challengeId"`
	Leaderboard []*model.LeaderboardEntry `json:"leaderboard"`
}

// LeaderboardManager manages multiple RedisBoard instances per challenge
type LeaderboardManager struct {
	boards map[string]*redisboard.Leaderboard // challengeID -> RedisBoard instance
	config *redisboard.Config
	mu     sync.RWMutex
}

// NewLeaderboardManager creates a new LeaderboardManager instance
func NewLeaderboardManager(redisAddr, redisPassword string) *LeaderboardManager {
	config := &redisboard.Config{
		K:           50,    // Track top 50 users
		MaxUsers:    10000, // Max users per challenge
		MaxEntities: 1,     // No entity grouping needed
		FloatScores: false, // Use integer scores
		RedisAddr:   redisAddr,
		RedisPass:   redisPassword,
	}

	return &LeaderboardManager{
		boards: make(map[string]*redisboard.Leaderboard),
		config: config,
	}
}

// InitializeLeaderboard creates a RedisBoard instance for a challenge
func (lm *LeaderboardManager) InitializeLeaderboard(challengeID string) error {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	// Check if already initialized
	if _, exists := lm.boards[challengeID]; exists {
		return nil
	}

	// Create challenge-specific config
	config := *lm.config
	config.Namespace = fmt.Sprintf("challenge_%s", challengeID)

	// Create RedisBoard instance
	board, err := redisboard.New(config)
	if err != nil {
		return fmt.Errorf("failed to create leaderboard for challenge %s: %w", challengeID, err)
	}

	lm.boards[challengeID] = board
	return nil
}

// CleanupLeaderboard properly closes RedisBoard instance for a challenge
func (lm *LeaderboardManager) CleanupLeaderboard(challengeID string) error {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	board, exists := lm.boards[challengeID]
	if !exists {
		return nil // Already cleaned up
	}

	// Close the RedisBoard instance
	err := board.Close()
	if err != nil {
		return fmt.Errorf("failed to close leaderboard for challenge %s: %w", challengeID, err)
	}

	// Remove from map
	delete(lm.boards, challengeID)
	return nil
}

// getBoard safely retrieves a RedisBoard instance for a challenge
func (lm *LeaderboardManager) getBoard(challengeID string) (*redisboard.Leaderboard, error) {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	board, exists := lm.boards[challengeID]
	if !exists {
		return nil, fmt.Errorf("leaderboard not initialized for challenge %s", challengeID)
	}

	return board, nil
}

// UpdateParticipantScore updates a participant's score using RedisBoard
func (lm *LeaderboardManager) UpdateParticipantScore(challengeID, userID string, points int) error {
	board, err := lm.getBoard(challengeID)
	if err != nil {
		return err
	}

	// Create user with score
	user := redisboard.User{
		ID:     userID,
		Entity: "", // No entity grouping for challenges
		Score:  float64(points),
	}

	// Add/update user score
	err = board.AddUser(user)
	if err != nil {
		return fmt.Errorf("failed to update score for user %s in challenge %s: %w", userID, challengeID, err)
	}

	return nil
}

// GetLeaderboard retrieves current leaderboard using RedisBoard's GetTopKGlobal
func (lm *LeaderboardManager) GetLeaderboard(challengeID string, limit int) ([]*model.LeaderboardEntry, error) {
	board, err := lm.getBoard(challengeID)
	if err != nil {
		return nil, err
	}

	// Get top users from RedisBoard
	users, err := board.GetTopKGlobal()
	if err != nil {
		// Handle case where no users exist yet
		if err.Error() == "no users in global leaderboard" {
			return []*model.LeaderboardEntry{}, nil
		}
		return nil, fmt.Errorf("failed to get leaderboard for challenge %s: %w", challengeID, err)
	}

	// Convert RedisBoard users to LeaderboardEntry
	leaderboard := make([]*model.LeaderboardEntry, 0, len(users))
	for i, user := range users {
		if limit > 0 && i >= limit {
			break
		}

		entry := &model.LeaderboardEntry{
			UserID:            user.ID,
			ProblemsCompleted: 0, // Will be calculated separately if needed
			TotalScore:        int(user.Score),
			Rank:              i + 1, // 1-based ranking
		}
		leaderboard = append(leaderboard, entry)
	}

	return leaderboard, nil
}

// GetParticipantRank gets a participant's rank and data using RedisBoard
func (lm *LeaderboardManager) GetParticipantRank(challengeID, userID string) (*ParticipantLeaderboardData, error) {
	board, err := lm.getBoard(challengeID)
	if err != nil {
		return nil, err
	}

	// Get user's leaderboard data from RedisBoard
	data, err := board.GetUserLeaderboardData(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get rank for user %s in challenge %s: %w", userID, challengeID, err)
	}

	// Convert to our format
	participantData := &ParticipantLeaderboardData{
		UserID:     userID,
		TotalScore: int(data.Score),
		GlobalRank: data.GlobalRank + 1, // Convert to 1-based ranking
		Rank:       data.GlobalRank + 1, // Alias for compatibility
	}

	// Handle case where user is not found (rank -1)
	if data.GlobalRank == -1 {
		participantData.GlobalRank = -1
		participantData.Rank = -1
	}

	return participantData, nil
}
