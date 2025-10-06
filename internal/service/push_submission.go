package service

import (
	"context"
	"log"
	"time"

	"github.com/lijuuu/ChallengeWssManagerService/internal/model"
	"github.com/lijuuu/ChallengeWssManagerService/internal/utils"
	"github.com/lijuuu/ChallengeWssManagerService/internal/wss/broadcasts"
	challengePb "github.com/lijuuu/GlobalProtoXcode/ChallengeService"
)

func (s *ChallengeService) PushSubmissionStatus(ctx context.Context, req *challengePb.PushSubmissionStatusRequest) (*challengePb.PushSubmissionStatusResponse, error) {
	// Extract submission data from request
	challengeID := req.GetChallengeId()
	userID := req.GetUserId()
	problemID := req.GetProblemId()
	score := int(req.GetScore())
	submissionID := req.GetSubmissionId()
	isSuccessful := req.GetIsSuccessful()
	timeTaken := time.Duration(req.GetTimeTakenMillis()) * time.Millisecond

	log.Printf("[PushSubmissionStatus] Processing submission: challenge=%s, user=%s, problem=%s, score=%d, successful=%v",
		challengeID, userID, problemID, score, isSuccessful)

	// Only process successful submissions for leaderboard updates
	if !isSuccessful {
		log.Printf("[PushSubmissionStatus] Submission unsuccessful, skipping leaderboard update")
		return &challengePb.PushSubmissionStatusResponse{Message: "received unsuccessful submission", Success: true}, nil
	}

	// Get challenge from Redis to verify it exists and is active
	challenge, err := s.GlobalState.Redis.GetChallengeByID(ctx, challengeID)
	if err != nil {
		log.Printf("[PushSubmissionStatus] Challenge not found: %v", err)
		return &challengePb.PushSubmissionStatusResponse{Message: "challenge not found", Success: false}, err
	}

	// Stop accepting submissions after timer end or terminal status
	if challenge.Status == model.ChallengeEnded || challenge.Status == model.ChallengeAbandon || utils.IsChallengeExpired(challenge) {
		return &challengePb.PushSubmissionStatusResponse{Message: "challenge has ended", Success: false}, nil
	}

	// Verify user is a participant
	participant, exists := challenge.Participants[userID]
	if !exists {
		log.Printf("[PushSubmissionStatus] User %s not a participant in challenge %s", userID, challengeID)
		return &challengePb.PushSubmissionStatusResponse{Message: "user not a participant", Success: false}, nil
	}

	// Update submission data in Redis
	submission := model.Submission{
		SubmissionID: submissionID,
		TimeTaken:    timeTaken,
		Points:       score,
	}

	// Initialize submissions map if needed
	if challenge.Submissions == nil {
		challenge.Submissions = make(map[string]map[string]model.Submission)
	}
	if challenge.Submissions[userID] == nil {
		challenge.Submissions[userID] = make(map[string]model.Submission)
	}

	// Store the submission
	challenge.Submissions[userID][problemID] = submission

	// Update participant metadata
	if participant.ProblemsDone == nil {
		participant.ProblemsDone = make(map[string]model.ChallengeProblemMetadata)
	}
	participant.ProblemsDone[problemID] = model.ChallengeProblemMetadata{
		Score:     score,
		TimeTaken: int64(timeTaken),
	}

	// Calculate new total score for the participant
	totalScore := 0
	for _, problemMeta := range participant.ProblemsDone {
		totalScore += problemMeta.Score
	}
	participant.TotalScore = totalScore
	participant.ProblemsAttempted = len(participant.ProblemsDone)

	// Update participant in Redis
	err = s.GlobalState.Redis.UpdateParticipant(ctx, challengeID, userID, participant)
	if err != nil {
		log.Printf("[PushSubmissionStatus] Failed to update participant: %v", err)
		return &challengePb.PushSubmissionStatusResponse{Message: "failed to update participant", Success: false}, err
	}

	// Update challenge in Redis
	err = s.GlobalState.Redis.UpdateChallenge(ctx, &challenge)
	if err != nil {
		log.Printf("[PushSubmissionStatus] Failed to update challenge: %v", err)
		return &challengePb.PushSubmissionStatusResponse{Message: "failed to update challenge", Success: false}, err
	}

	// Initialize leaderboard if not already done
	err = s.GlobalState.LeaderboardManager.InitializeLeaderboard(challengeID)
	if err != nil {
		log.Printf("[PushSubmissionStatus] Failed to initialize leaderboard: %v", err)
		// Continue processing even if leaderboard fails
	}

	// Update participant score in leaderboard
	err = s.GlobalState.LeaderboardManager.UpdateParticipantScore(challengeID, userID, totalScore)
	if err != nil {
		log.Printf("[PushSubmissionStatus] Failed to update leaderboard score: %v", err)
		// Continue processing even if leaderboard update fails
	}

	// Get updated leaderboard for broadcasting
	var leaderboard []*model.LeaderboardEntry
	var newRank int = -1

	// Get updated leaderboard data
	leaderboard, err = s.GlobalState.LeaderboardManager.GetLeaderboard(challengeID, 50, &challenge) // Get top 50
	if err != nil {
		log.Printf("[PushSubmissionStatus] Failed to get leaderboard: %v", err)
	}

	// Get user's new rank and add a human-readable notification
	participantData, err := s.GlobalState.LeaderboardManager.GetParticipantRank(challengeID, userID)
	if err != nil {
		log.Printf("[PushSubmissionStatus] Failed to get participant rank: %v", err)
	} else if participantData != nil {
		newRank = participantData.Rank
		challenge.Notifications = append(challenge.Notifications, utils.NewSubmissionNotification(userID, problemID, score))
		if err := s.GlobalState.Redis.UpdateChallenge(ctx, &challenge); err != nil {
			log.Printf("[PushSubmissionStatus] Failed to append notification: %v", err)
		}
	}

	// Broadcast events to WebSocket clients
	if s.GlobalState != nil && s.GlobalState.LocalState != nil {
		wsClients := s.GlobalState.LocalState.GetAllWSClients(challengeID)

		// Broadcast PUSH_SUBMISSION event
		broadcasts.BroadcastNewSubmission(wsClients, challengeID, userID, problemID, score, newRank)

		// Broadcast LEADERBOARD_UPDATE event if we have leaderboard data
		if leaderboard != nil {
			broadcasts.BroadcastLeaderboardUpdate(wsClients, challengeID, leaderboard, userID)
		}
	}

	log.Printf("[PushSubmissionStatus] Successfully processed submission for user %s in challenge %s, new total score: %d",
		userID, challengeID, totalScore)

	return &challengePb.PushSubmissionStatusResponse{Message: "submission processed successfully", Success: true}, nil
}
