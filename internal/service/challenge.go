package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"sort"
	"sync"
	"time"

	"github.com/lijuuu/ChallengeWssManagerService/internal/models"

	"github.com/gorilla/websocket"
)

type ChallengeWrapper struct {
	Challenge *models.Challenge
}

var (
	challenges           = make(map[string]*ChallengeWrapper)
	challengeMu          sync.RWMutex
	currentMatches       int
	matchesMu            sync.Mutex
	ErrMatchLimitReached = errors.New("maximum concurrent matches reached")
	ErrMaxUsersReached   = errors.New("maximum number of users reached")
	ErrNotCreator        = errors.New("only creator can perform this action")
	ErrInvalidMessage    = errors.New("invalid websocket message")
	ErrUnauthorized      = errors.New("unauthorized action")
	ErrInvalidProblem    = errors.New("invalid problem ID")
)

// validProblems is a mock problem database
var validProblems = map[models.QuestionDifficulty][]string{
	models.DifficultyEasy:   {"easy1", "easy2", "easy3", "easy4"},
	models.DifficultyMedium: {"med1", "med2", "med3", "med4"},
	models.DifficultyHard:   {"hard1", "hard2", "hard3"},
}

func ValidProblems() map[models.QuestionDifficulty][]string {
	return validProblems
}

// GenerateChallengeID creates a unique challenge ID
func GenerateChallengeID() string {
	return hex.EncodeToString(hmac.New(sha256.New, []byte(time.Now().String())).Sum(nil))[:8]
}

// validateProblems checks if provided problems exist
func validateProblems(problems map[models.QuestionDifficulty][]string) error {
	for difficulty, questionIDs := range problems {
		if _, exists := validProblems[difficulty]; !exists {
			return fmt.Errorf("invalid difficulty: %s", difficulty)
		}
		for _, qID := range questionIDs {
			valid := false
			for _, validID := range validProblems[difficulty] {
				if qID == validID {
					valid = true
					break
				}
			}
			if !valid {
				return fmt.Errorf("invalid problem ID: %s for difficulty %s", qID, difficulty)
			}
		}
	}
	return nil
}

// NewChallenge creates a new challenge with configuration
func NewChallenge(challengeID, creatorID, title string, isPrivate bool, password string, timeLimitMinutes int, config models.ChallengeConfig) (*ChallengeWrapper, error) {
	matchesMu.Lock()
	if currentMatches >= models.MaxConcurrentMatches {
		matchesMu.Unlock()
		return nil, ErrMatchLimitReached
	}
	currentMatches++
	matchesMu.Unlock()

	// Validate initial questions
	if err := validateProblems(config.InitialQuestions); err != nil {
		matchesMu.Lock()
		currentMatches--
		matchesMu.Unlock()
		return nil, err
	}

	problemIDs := generateProblemIDs(config)
	userQuestions := make([]string, 0)
	for _, questions := range config.InitialQuestions {
		userQuestions = append(userQuestions, questions...)
		problemIDs = append(problemIDs, questions...)
	}

	challenge := &models.Challenge{
		ChallengeID:  challengeID,
		CreatorID:    creatorID,
		Title:        title,
		IsPrivate:    isPrivate,
		Password:     password,
		Status:       models.StatusWaiting,
		ProblemIDs:   problemIDs,
		TimeLimit:    time.Duration(timeLimitMinutes) * time.Minute,
		StartTime:    time.Now(),
		Participants: make(map[string]*models.ParticipantMetadata),
		Leaderboard:  []*models.LeaderboardEntry{},
		Sessions:     make(map[string]*models.Session),
		Config:       config,
		WSClients:    make(map[string]*websocket.Conn),
		EventChan:    make(chan models.Event, 100),
	}

	wrapper := &ChallengeWrapper{Challenge: challenge}

	challengeMu.Lock()
	challenges[challengeID] = wrapper
	challengeMu.Unlock()

	go wrapper.CleanupRoutine()
	go wrapper.BroadcastRoutine()
	go wrapper.TimeUpdateRoutine()

	wrapper.Challenge.EventChan <- models.Event{
		Type: models.EventChallengeCreated,
		Payload: models.ChallengeCreatedPayload{
			ChallengeID: challengeID,
			Title:       title,
		},
	}

	return wrapper, nil
}

// generateProblemIDs creates problem IDs based on config
func generateProblemIDs(config models.ChallengeConfig) []string {
	var problemIDs []string
	rand.Seed(time.Now().UnixNano())

	for _, difficulty := range []models.QuestionDifficulty{models.DifficultyEasy, models.DifficultyMedium, models.DifficultyHard} {
		maxQuestions := 0
		switch difficulty {
		case models.DifficultyEasy:
			maxQuestions = config.MaxEasyQuestions
		case models.DifficultyMedium:
			maxQuestions = config.MaxMediumQuestions
		case models.DifficultyHard:
			maxQuestions = config.MaxHardQuestions
		}

		availableQuestions, exists := config.RandomQuestionPool[difficulty]
		if !exists || len(availableQuestions) == 0 {
			continue
		}

		shuffled := make([]string, len(availableQuestions))
		copy(shuffled, availableQuestions)
		rand.Shuffle(len(shuffled), func(i, j int) {
			shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
		})

		for i := 0; i < maxQuestions && i < len(shuffled); i++ {
			problemIDs = append(problemIDs, shuffled[i])
		}
	}

	return problemIDs
}

// TimeUpdateRoutine sends periodic time updates
func (cw *ChallengeWrapper) TimeUpdateRoutine() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		cw.Challenge.MU.RLock()
		if cw.Challenge.Status != models.StatusStarted {
			cw.Challenge.MU.RUnlock()
			continue
		}

		elapsed := time.Since(cw.Challenge.StartTime)
		remaining := cw.Challenge.TimeLimit - elapsed
		if remaining <= 0 {
			cw.Challenge.MU.RUnlock()
			EndChallenge(cw.Challenge.ChallengeID)
			return
		}

		cw.Challenge.EventChan <- models.Event{
			Type: models.EventTimeUpdate,
			Payload: models.TimeUpdatePayload{
				RemainingTime: int64(remaining.Seconds()),
			},
		}
		cw.Challenge.MU.RUnlock()
	}
}

// CleanupRoutine periodically cleans up expired sessions and empty challenges
func (cw *ChallengeWrapper) CleanupRoutine() {
	ticker := time.NewTicker(models.CleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		cw.Challenge.MU.Lock()
		if cw.Challenge.Status == models.StatusFinished || cw.Challenge.Status == models.StatusCancelled {
			cw.Challenge.MU.Unlock()
			return
		}

		// Check for empty challenge
		if cw.Challenge.Status == models.StatusWaiting && len(cw.Challenge.Participants) == 0 && time.Since(cw.Challenge.StartTime) > models.EmptyChallengeTimeout {
			cw.Challenge.Status = models.StatusCancelled
			challengeMu.Lock()
			delete(challenges, cw.Challenge.ChallengeID)
			challengeMu.Unlock()
			matchesMu.Lock()
			currentMatches--
			matchesMu.Unlock()

			cw.Challenge.EventChan <- models.Event{
				Type: models.EventChallengeDeleted,
				Payload: models.ChallengeDeletedPayload{
					ChallengeID: cw.Challenge.ChallengeID,
				},
			}
			cw.Challenge.MU.Unlock()
			return
		}

		for userID, session := range cw.Challenge.Sessions {
			if time.Since(session.LastActive) > models.SessionTimeout {
				delete(cw.Challenge.Sessions, userID)
				deleteSession(userID + cw.Challenge.ChallengeID)

				if conn, exists := cw.Challenge.WSClients[userID]; exists {
					conn.Close()
					delete(cw.Challenge.WSClients, userID)
				}

				cw.Challenge.EventChan <- models.Event{
					Type: models.EventUserLeft,
					Payload: models.UserLeftPayload{
						UserID: userID,
					},
				}

				delete(cw.Challenge.Participants, userID)
				updateLeaderboard(cw.Challenge)
			}
		}
		cw.Challenge.MU.Unlock()
	}
}

// BroadcastRoutine handles broadcasting events to WebSocket clients
func (cw *ChallengeWrapper) BroadcastRoutine() {
	for event := range cw.Challenge.EventChan {
		cw.Challenge.MU.RLock()
		data, err := json.Marshal(event)
		if err != nil {
			log.Printf("Error marshaling event: %v", err)
			continue
		}

		for userID, conn := range cw.Challenge.WSClients {
			if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
				log.Printf("Error broadcasting to %s: %v", userID, err)
				conn.Close()
				delete(cw.Challenge.WSClients, userID)

				cw.Challenge.EventChan <- models.Event{
					Type: models.EventUserLeft,
					Payload: models.UserLeftPayload{
						UserID: userID,
					},
				}

				delete(cw.Challenge.Participants, userID)
				updateLeaderboard(cw.Challenge)
			}
		}
		cw.Challenge.MU.RUnlock()

		if event.Type == models.EventChallengeStatusChanged && event.Payload.(models.ChallengeStatusChangedPayload).Status == models.StatusFinished || event.Type == models.EventChallengeDeleted {
			close(cw.Challenge.EventChan)
			return
		}
	}
}

// updateLeaderboard updates the leaderboard in-place with sorting
func updateLeaderboard(c *models.Challenge) {
	var board []*models.LeaderboardEntry
	for _, p := range c.Participants {
		entry := &models.LeaderboardEntry{
			UserID:            p.UserID,
			ProblemsCompleted: len(p.ProblemsDone),
			TotalScore:        p.TotalScore,
		}
		board = append(board, entry)
	}

	sort.Slice(board, func(i, j int) bool {
		if board[i].TotalScore == board[j].TotalScore {
			return board[i].ProblemsCompleted > board[j].ProblemsCompleted
		}
		return board[i].TotalScore > board[j].TotalScore
	})

	for i, entry := range board {
		entry.Rank = i + 1
	}

	c.Leaderboard = board

	c.EventChan <- models.Event{
		Type: models.EventLeaderboardUpdated,
		Payload: models.LeaderboardUpdatedPayload{
			Leaderboard: board,
		},
	}
}

// JoinChallenge adds a participant to a challenge with session creation
func JoinChallenge(challengeID, userID, password, sessionHash string) (string, error) {
	challengeMu.RLock()
	wrapper, ok := challenges[challengeID]
	challengeMu.RUnlock()
	if !ok {
		return "", errors.New("challenge not found")
	}

	wrapper.Challenge.MU.Lock()
	defer wrapper.Challenge.MU.Unlock()

	if wrapper.Challenge.IsPrivate && wrapper.Challenge.Password != password {
		return "", errors.New("invalid password")
	}

	if wrapper.Challenge.Config.MaxUsers > 0 && len(wrapper.Challenge.Participants) >= wrapper.Challenge.Config.MaxUsers {
		return "", ErrMaxUsersReached
	}

	if wrapper.Challenge.Status != models.StatusWaiting {
		return "", errors.New("challenge not accepting new participants")
	}

	if _, exists := wrapper.Challenge.Participants[userID]; exists {
		return "", errors.New("user already joined")
	}

	newSessionHash := sessionHash
	if newSessionHash == "" {
		newSessionHash = GenerateSessionHash(userID, challengeID, password)
	}

	wrapper.Challenge.Participants[userID] = &models.ParticipantMetadata{
		UserID:        userID,
		ProblemsDone:  make(map[string]models.ChallengeProblemMetadata),
		LastConnected: time.Now(),
	}

	wrapper.Challenge.EventChan <- models.Event{
		Type: models.EventUserJoined,
		Payload: models.UserJoinedPayload{
			UserID: userID,
		},
	}

	session := &models.Session{
		UserID:      userID,
		ChallengeID: challengeID,
		LastActive:  time.Now(),
		SessionHash: newSessionHash,
	}

	setSession(userID+challengeID, session)

	wrapper.Challenge.Sessions[userID] = session
	return newSessionHash, nil
}

// ReconnectChallenge handles participant reconnection
func ReconnectChallenge(challengeID, userID, sessionHash string) error {
	challengeMu.RLock()
	wrapper, ok := challenges[challengeID]
	challengeMu.RUnlock()
	if !ok {
		return errors.New("challenge not found")
	}

	wrapper.Challenge.MU.Lock()
	defer wrapper.Challenge.MU.Unlock()

	session, exists := wrapper.Challenge.Sessions[userID]
	if !exists {
		return errors.New("no active session found")
	}

	if !ValidateSessionHash(userID, challengeID, wrapper.Challenge.Password, sessionHash) {
		return errors.New("invalid session hash")
	}

	session.LastActive = time.Now()
	participant, ok := wrapper.Challenge.Participants[userID]
	if ok {
		participant.LastConnected = time.Now()
	}

	return nil
}

// StartChallenge marks the challenge as started
func StartChallenge(challengeID, userID, sessionHash string) error {
	challengeMu.RLock()
	wrapper, ok := challenges[challengeID]
	challengeMu.RUnlock()
	if !ok {
		return errors.New("challenge not found")
	}

	if err := ReconnectChallenge(challengeID, userID, sessionHash); err != nil {
		return err
	}

	wrapper.Challenge.MU.Lock()
	defer wrapper.Challenge.MU.Unlock()

	if wrapper.Challenge.CreatorID != userID {
		return ErrNotCreator
	}

	if wrapper.Challenge.Status != models.StatusWaiting {
		return errors.New("challenge already started or finished")
	}

	if len(wrapper.Challenge.Participants) == 0 {
		return errors.New("no participants in challenge")
	}

	wrapper.Challenge.Status = models.StatusStarted
	wrapper.Challenge.StartTime = time.Now()

	wrapper.Challenge.EventChan <- models.Event{
		Type: models.EventChallengeStatusChanged,
		Payload: models.ChallengeStatusChangedPayload{
			ChallengeID: challengeID,
			Status:      wrapper.Challenge.Status,
		},
	}

	return nil
}

// SubmitProblem updates the challenge with problem submission data
func SubmitProblem(challengeID, userID, problemID, sessionHash string, score int) error {
	challengeMu.RLock()
	wrapper, ok := challenges[challengeID]
	challengeMu.RUnlock()
	if !ok {
		return errors.New("challenge not found")
	}

	if err := ReconnectChallenge(challengeID, userID, sessionHash); err != nil {
		return err
	}

	wrapper.Challenge.MU.Lock()
	defer wrapper.Challenge.MU.Unlock()

	participant, ok := wrapper.Challenge.Participants[userID]
	if !ok {
		return errors.New("user not in challenge")
	}

	if wrapper.Challenge.Status != models.StatusStarted {
		return errors.New("challenge not in started state")
	}

	if time.Since(wrapper.Challenge.StartTime) > wrapper.Challenge.TimeLimit {
		wrapper.Challenge.Status = models.StatusFinished
		return errors.New("challenge time limit exceeded")
	}

	if _, done := participant.ProblemsDone[problemID]; done {
		return errors.New("problem already submitted")
	}

	// Validate problemID exists in challenge
	validProblem := false
	for _, pid := range wrapper.Challenge.ProblemIDs {
		if pid == problemID {
			validProblem = true
			break
		}
	}
	if !validProblem {
		return ErrInvalidProblem
	}

	timeTaken := time.Since(wrapper.Challenge.StartTime).Milliseconds()
	participant.ProblemsDone[problemID] = models.ChallengeProblemMetadata{
		ProblemID:   problemID,
		Score:       score,
		TimeTaken:   timeTaken,
		CompletedAt: time.Now(),
	}
	participant.TotalScore += score
	participant.ProblemsAttempted++
	participant.LastConnected = time.Now()

	wrapper.Challenge.EventChan <- models.Event{
		Type: models.EventProblemSubmitted,
		Payload: models.ProblemSubmittedPayload{
			UserID:    userID,
			ProblemID: problemID,
			Score:     score,
		},
	}

	updateLeaderboard(wrapper.Challenge)
	return nil
}

// ForfeitChallenge allows a user to forfeit
func ForfeitChallenge(challengeID, userID, sessionHash string) error {
	challengeMu.RLock()
	wrapper, ok := challenges[challengeID]
	challengeMu.RUnlock()
	if !ok {
		return errors.New("challenge not found")
	}

	if err := ReconnectChallenge(challengeID, userID, sessionHash); err != nil {
		return err
	}

	wrapper.Challenge.MU.Lock()
	defer wrapper.Challenge.MU.Unlock()

	_, ok = wrapper.Challenge.Participants[userID]
	if !ok {
		return errors.New("user not in challenge")
	}

	delete(wrapper.Challenge.Participants, userID)
	delete(wrapper.Challenge.Sessions, userID)

	if conn, exists := wrapper.Challenge.WSClients[userID]; exists {
		conn.Close()
		delete(wrapper.Challenge.WSClients, userID)
	}

	deleteSession(userID + wrapper.Challenge.ChallengeID)

	wrapper.Challenge.EventChan <- models.Event{
		Type: models.EventUserForfeited,
		Payload: models.UserForfeitedPayload{
			UserID: userID,
		},
	}

	updateLeaderboard(wrapper.Challenge)
	return nil
}

// EndChallenge marks the challenge as finished and cleans up
func EndChallenge(challengeID string) error {
	challengeMu.RLock()
	wrapper, ok := challenges[challengeID]
	challengeMu.RUnlock()
	if !ok {
		return errors.New("challenge not found")
	}

	wrapper.Challenge.MU.Lock()
	defer wrapper.Challenge.MU.Unlock()

	if wrapper.Challenge.Status == models.StatusFinished {
		return nil
	}

	wrapper.Challenge.Status = models.StatusFinished
	for userID := range wrapper.Challenge.Sessions {
		deleteSession(userID + wrapper.Challenge.ChallengeID)
	}

	for userID, conn := range wrapper.Challenge.WSClients {
		conn.Close()
		delete(wrapper.Challenge.WSClients, userID)
	}

	wrapper.Challenge.EventChan <- models.Event{
		Type: models.EventChallengeStatusChanged,
		Payload: models.ChallengeStatusChangedPayload{
			ChallengeID: challengeID,
			Status:      wrapper.Challenge.Status,
		},
	}

	matchesMu.Lock()
	currentMatches--
	matchesMu.Unlock()

	return nil
}

// DeleteChallenge allows the creator to delete a challenge
func DeleteChallenge(challengeID, userID, sessionHash string) error {
	challengeMu.RLock()
	wrapper, ok := challenges[challengeID]
	challengeMu.RUnlock()
	if !ok {
		return errors.New("challenge not found")
	}

	if err := ReconnectChallenge(challengeID, userID, sessionHash); err != nil {
		return err
	}

	wrapper.Challenge.MU.Lock()
	defer wrapper.Challenge.MU.Unlock()

	if wrapper.Challenge.CreatorID != userID {
		return ErrNotCreator
	}

	wrapper.Challenge.Status = models.StatusCancelled
	for userID := range wrapper.Challenge.Sessions {
		deleteSession(userID + wrapper.Challenge.ChallengeID)
	}

	for userID, conn := range wrapper.Challenge.WSClients {
		conn.Close()
		delete(wrapper.Challenge.WSClients, userID)
	}

	wrapper.Challenge.EventChan <- models.Event{
		Type: models.EventChallengeDeleted,
		Payload: models.ChallengeDeletedPayload{
			ChallengeID: challengeID,
		},
	}

	challengeMu.Lock()
	delete(challenges, challengeID)
	challengeMu.Unlock()

	matchesMu.Lock()
	currentMatches--
	matchesMu.Unlock()

	return nil
}

// ListOpenChallenges returns a list of open (non-private) challenges
func ListOpenChallenges() ([]*ChallengeWrapper, error) {
	challengeMu.RLock()
	defer challengeMu.RUnlock()

	var openChallenges []*ChallengeWrapper
	for _, wrapper := range challenges {
		if !wrapper.Challenge.IsPrivate && wrapper.Challenge.Status == models.StatusWaiting {
			openChallenges = append(openChallenges, wrapper)
		}
	}

	return openChallenges, nil
}

// GetChallenge retrieves a challenge by ID
func GetChallenge(challengeID string) (*ChallengeWrapper, bool) {
	challengeMu.RLock()
	defer challengeMu.RUnlock()
	wrapper, ok := challenges[challengeID]
	return wrapper, ok
}

// ConnectWebSocket establishes a WebSocket connection for a user
func ConnectWebSocket(challengeID, userID, sessionHash string, conn *websocket.Conn) error {
	challengeMu.RLock()
	wrapper, ok := challenges[challengeID]
	challengeMu.RUnlock()
	if !ok {
		return errors.New("challenge not found")
	}

	if err := ReconnectChallenge(challengeID, userID, sessionHash); err != nil {
		return err
	}

	wrapper.Challenge.MU.Lock()
	defer wrapper.Challenge.MU.Unlock()

	if oldConn, exists := wrapper.Challenge.WSClients[userID]; exists {
		oldConn.Close()
	}

	wrapper.Challenge.WSClients[userID] = conn

	event := models.Event{
		Type: models.EventLeaderboardUpdated,
		Payload: models.LeaderboardUpdatedPayload{
			Leaderboard: wrapper.Challenge.Leaderboard,
		},
	}
	data, _ := json.Marshal(event)
	conn.WriteMessage(websocket.TextMessage, data)

	event = models.Event{
		Type: models.EventChallengeStatusChanged,
		Payload: models.ChallengeStatusChangedPayload{
			ChallengeID: challengeID,
			Status:      wrapper.Challenge.Status,
		},
	}
	data, _ = json.Marshal(event)
	conn.WriteMessage(websocket.TextMessage, data)

	if wrapper.Challenge.Status == models.StatusStarted {
		remaining := wrapper.Challenge.TimeLimit - time.Since(wrapper.Challenge.StartTime)
		if remaining > 0 {
			event = models.Event{
				Type: models.EventTimeUpdate,
				Payload: models.TimeUpdatePayload{
					RemainingTime: int64(remaining.Seconds()),
				},
			}
			data, _ = json.Marshal(event)
			conn.WriteMessage(websocket.TextMessage, data)
		}
	}

	go wrapper.handleWebSocketMessages(userID, conn)
	return nil
}

// handleWebSocketMessages processes incoming WebSocket messages
func (cw *ChallengeWrapper) handleWebSocketMessages(userID string, conn *websocket.Conn) {
	conn.SetReadDeadline(time.Now().Add(models.WebsocketReadTimeout))
	for {
		messageType, data, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Error reading WebSocket message for user %s: %v", userID, err)
			cw.Challenge.MU.Lock()
			delete(cw.Challenge.WSClients, userID)
			delete(cw.Challenge.Sessions, userID)
			delete(cw.Challenge.Participants, userID)
			deleteSession(userID + cw.Challenge.ChallengeID)
			cw.Challenge.EventChan <- models.Event{
				Type: models.EventUserLeft,
				Payload: models.UserLeftPayload{
					UserID: userID,
				},
			}
			updateLeaderboard(cw.Challenge)
			cw.Challenge.MU.Unlock()
			conn.Close()
			return
		}

		if messageType != websocket.TextMessage {
			continue
		}

		var msg models.WebSocketMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			log.Printf("Error unmarshaling WebSocket message: %v", err)
			cw.Challenge.EventChan <- models.Event{
				Type: models.EventError,
				Payload: models.ErrorPayload{
					Message: "Invalid message format",
				},
			}
			continue
		}

		cw.Challenge.MU.Lock()
		switch msg.Type {
		case "join_challenge":
			var payload models.JoinChallengeRequest
			if err := json.Unmarshal(msg.Payload, &payload); err != nil {
				log.Printf("Error unmarshaling join_challenge payload: %v", err)
				cw.Challenge.EventChan <- models.Event{
					Type: models.EventError,
					Payload: models.ErrorPayload{
						Message: "Invalid join challenge payload",
					},
				}
				continue
			}
			if payload.UserID != userID {
				log.Printf("Unauthorized join attempt by user %s", userID)
				cw.Challenge.EventChan <- models.Event{
					Type: models.EventError,
					Payload: models.ErrorPayload{
						Message: "Unauthorized action",
					},
				}
				continue
			}
			sessionHash, err := JoinChallenge(cw.Challenge.ChallengeID, userID, payload.Password, payload.SessionHash)
			if err != nil {
				log.Printf("Error joining challenge: %v", err)
				cw.Challenge.EventChan <- models.Event{
					Type: models.EventError,
					Payload: models.ErrorPayload{
						Message: err.Error(),
					},
				}
				continue
			}
			conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf(`{"type":"join_success","payload":{"session_hash":"%s"}}`, sessionHash)))

		case "start_challenge":
			var payload models.StartChallengeRequest
			if err := json.Unmarshal(msg.Payload, &payload); err != nil {
				log.Printf("Error unmarshaling start_challenge payload: %v", err)
				cw.Challenge.EventChan <- models.Event{
					Type: models.EventError,
					Payload: models.ErrorPayload{
						Message: "Invalid start challenge payload",
					},
				}
				continue
			}
			if payload.UserID != userID || !ValidateSessionHash(userID, cw.Challenge.ChallengeID, cw.Challenge.Password, payload.SessionHash) {
				log.Printf("Unauthorized start attempt by user %s", userID)
				cw.Challenge.EventChan <- models.Event{
					Type: models.EventError,
					Payload: models.ErrorPayload{
						Message: "Unauthorized action",
					},
				}
				continue
			}
			err := StartChallenge(cw.Challenge.ChallengeID, userID, payload.SessionHash)
			if err != nil {
				log.Printf("Error starting challenge: %v", err)
				cw.Challenge.EventChan <- models.Event{
					Type: models.EventError,
					Payload: models.ErrorPayload{
						Message: err.Error(),
					},
				}
				continue
			}

		case "end_challenge":
			var payload models.EndChallengeRequest
			if err := json.Unmarshal(msg.Payload, &payload); err != nil {
				log.Printf("Error unmarshaling end_challenge payload: %v", err)
				cw.Challenge.EventChan <- models.Event{
					Type: models.EventError,
					Payload: models.ErrorPayload{
						Message: "Invalid end challenge payload",
					},
				}
				continue
			}
			if payload.UserID != userID || !ValidateSessionHash(userID, cw.Challenge.ChallengeID, cw.Challenge.Password, payload.SessionHash) {
				log.Printf("Unauthorized end attempt by user %s", userID)
				cw.Challenge.EventChan <- models.Event{
					Type: models.EventError,
					Payload: models.ErrorPayload{
						Message: "Unauthorized action",
					},
				}
				continue
			}
			err := EndChallenge(cw.Challenge.ChallengeID)
			if err != nil {
				log.Printf("Error ending challenge: %v", err)
				cw.Challenge.EventChan <- models.Event{
					Type: models.EventError,
					Payload: models.ErrorPayload{
						Message: err.Error(),
					},
				}
				continue
			}

		case "delete_challenge":
			var payload models.DeleteChallengeRequest
			if err := json.Unmarshal(msg.Payload, &payload); err != nil {
				log.Printf("Error unmarshaling delete_challenge payload: %v", err)
				cw.Challenge.EventChan <- models.Event{
					Type: models.EventError,
					Payload: models.ErrorPayload{
						Message: "Invalid delete challenge payload",
					},
				}
				continue
			}
			if payload.UserID != userID || !ValidateSessionHash(userID, cw.Challenge.ChallengeID, cw.Challenge.Password, payload.SessionHash) {
				log.Printf("Unauthorized delete attempt by user %s", userID)
				cw.Challenge.EventChan <- models.Event{
					Type: models.EventError,
					Payload: models.ErrorPayload{
						Message: "Unauthorized action",
					},
				}
				continue
			}
			err := DeleteChallenge(cw.Challenge.ChallengeID, userID, payload.SessionHash)
			if err != nil {
				log.Printf("Error deleting challenge: %v", err)
				cw.Challenge.EventChan <- models.Event{
					Type: models.EventError,
					Payload: models.ErrorPayload{
						Message: err.Error(),
					},
				}
				continue
			}

		case "submit_problem":
			var payload models.SubmitProblemRequest
			if err := json.Unmarshal(msg.Payload, &payload); err != nil {
				log.Printf("Error unmarshaling submit_problem payload: %v", err)
				cw.Challenge.EventChan <- models.Event{
					Type: models.EventError,
					Payload: models.ErrorPayload{
						Message: "Invalid submit problem payload",
					},
				}
				continue
			}
			_, exists := cw.Challenge.Sessions[userID]
			if !exists || !ValidateSessionHash(userID, cw.Challenge.ChallengeID, cw.Challenge.Password, payload.SessionHash) {
				log.Printf("Unauthorized problem submission attempt by user %s", userID)
				cw.Challenge.EventChan <- models.Event{
					Type: models.EventError,
					Payload: models.ErrorPayload{
						Message: "Unauthorized action",
					},
				}
				continue
			}
			err := SubmitProblem(cw.Challenge.ChallengeID, userID, payload.ProblemID, payload.SessionHash, payload.Score)
			if err != nil {
				log.Printf("Error submitting problem: %v", err)
				cw.Challenge.EventChan <- models.Event{
					Type: models.EventError,
					Payload: models.ErrorPayload{
						Message: err.Error(),
					},
				}
				continue
			}

		case "forfeit":
			var payload models.ForfeitRequest
			if err := json.Unmarshal(msg.Payload, &payload); err != nil {
				log.Printf("Error unmarshaling forfeit payload: %v", err)
				cw.Challenge.EventChan <- models.Event{
					Type: models.EventError,
					Payload: models.ErrorPayload{
						Message: "Invalid forfeit payload",
					},
				}
				continue
			}
			if payload.UserID != userID || !ValidateSessionHash(userID, cw.Challenge.ChallengeID, cw.Challenge.Password, payload.SessionHash) {
				log.Printf("Unauthorized forfeit attempt by user %s", userID)
				cw.Challenge.EventChan <- models.Event{
					Type: models.EventError,
					Payload: models.ErrorPayload{
						Message: "Unauthorized action",
					},
				}
				continue
			}
			err := ForfeitChallenge(cw.Challenge.ChallengeID, userID, payload.SessionHash)
			if err != nil {
				log.Printf("Error processing forfeit: %v", err)
				cw.Challenge.EventChan <- models.Event{
					Type: models.EventError,
					Payload: models.ErrorPayload{
						Message: err.Error(),
					},
				}
				continue
			}

		case "ping":
			if session, exists := cw.Challenge.Sessions[userID]; exists {
				session.LastActive = time.Now()
			}
			if participant, exists := cw.Challenge.Participants[userID]; exists {
				participant.LastConnected = time.Now()
			}
			conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"pong"}`))
		}
		cw.Challenge.MU.Unlock()
	}
}