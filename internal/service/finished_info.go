package service

import (
	"context"
	"log"
	"sort"
	"time"

	pb "github.com/lijuuu/GlobalProtoXcode/ChallengeService"
)

// In a real implementation, you'd inject dependencies like a repository or database client.
// For this skeleton, we'll use simple in-memory maps to simulate state.
// Note: This is not thread-safe or persistent; use mutexes or a proper DB in production.

type inMemoryRepo struct {
	challenges      map[string]*pb.ChallengeRecord
	postChallenges  map[string]*pb.PostChallengeDocument
	history         map[string][]*pb.ChallengeRecord // userId -> list of challenges
	ownerChallenges map[string][]*pb.ChallengeRecord // userId -> list of owned challenges
}

func newInMemoryRepo() *inMemoryRepo {
	return &inMemoryRepo{
		challenges:      make(map[string]*pb.ChallengeRecord),
		postChallenges:  make(map[string]*pb.PostChallengeDocument),
		history:         make(map[string][]*pb.ChallengeRecord),
		ownerChallenges: make(map[string][]*pb.ChallengeRecord),
	}
}

type ChallengeServiceServer struct {
	pb.UnimplementedChallengeServiceServer
	repo *inMemoryRepo
}

func NewChallengeServiceServer() *ChallengeServiceServer {
	return &ChallengeServiceServer{
		repo: newInMemoryRepo(),
	}
}

func (s *ChallengeServiceServer) CreateChallenge(ctx context.Context, req *pb.ChallengeRecord) (*pb.CreateChallengeResponse, error) {
	if req.ChallengeId == "" {
		return &pb.CreateChallengeResponse{
			Success:   false,
			Message:   "Challenge ID is required",
			ErrorType: "VALIDATION_ERROR",
		}, nil
	}

	// Simulate creation: set createdAt if not set
	if req.CreatedAt == 0 {
		req.CreatedAt = time.Now().Unix()
	}
	req.Status = "ACTIVE"

	s.repo.challenges[req.ChallengeId] = req
	s.repo.ownerChallenges[req.CreatorId] = append(s.repo.ownerChallenges[req.CreatorId], req)

	log.Printf("Created challenge: %s", req.ChallengeId)

	return &pb.CreateChallengeResponse{
		Success:   true,
		Message:   "Challenge created successfully",
		Challenge: req,
	}, nil
}

func (s *ChallengeServiceServer) AbandonChallenge(ctx context.Context, req *pb.AbandonChallengeRequest) (*pb.AbandonChallengeResponse, error) {
	challenge, exists := s.repo.challenges[req.ChallengeId]
	if !exists {
		return &pb.AbandonChallengeResponse{
			Success:   false,
			Message:   "Challenge not found",
			ErrorType: "NOT_FOUND",
		}, nil
	}

	if challenge.CreatorId != req.CreatorId {
		return &pb.AbandonChallengeResponse{
			Success:   false,
			Message:   "Unauthorized to abandon this challenge",
			ErrorType: "UNAUTHORIZED",
		}, nil
	}

	// Set status to abandoned
	challenge.Status = "ABANDONED"

	log.Printf("Abandoned challenge: %s", req.ChallengeId)

	return &pb.AbandonChallengeResponse{
		Success: true,
		Message: "Challenge abandoned successfully",
	}, nil
}

func (s *ChallengeServiceServer) GetChallengeRoomInfoMetadata(ctx context.Context, req *pb.GetChallengeRoomInfoMetadataRequest) (*pb.GetChallengeRoomInfoMetadataResponse, error) {
	challenge, exists := s.repo.challenges[req.ChallengeId]
	if !exists {
		return &pb.GetChallengeRoomInfoMetadataResponse{
			Success:   false,
			Message:   "Challenge not found",
			ErrorType: "NOT_FOUND",
		}, nil
	}

	// Check password if private
	if challenge.IsPrivate && req.Password != &challenge.Password {
		return &pb.GetChallengeRoomInfoMetadataResponse{
			Success:   false,
			Message:   "Invalid password",
			ErrorType: "UNAUTHORIZED",
		}, nil
	}

	return &pb.GetChallengeRoomInfoMetadataResponse{
		Success:   true,
		Message:   "Success",
		Challenge: challenge,
	}, nil
}

func (s *ChallengeServiceServer) GetFullChallengeData(ctx context.Context, req *pb.GetFullChallengeDataRequest) (*pb.GetFullChallengeDataResponse, error) {
	challenge, exists := s.repo.challenges[req.ChallengeId]
	if !exists {
		return &pb.GetFullChallengeDataResponse{
			Success:   false,
			Message:   "Challenge not found",
			ErrorType: "NOT_FOUND",
		}, nil
	}

	// Check password if private
	if challenge.IsPrivate && req.Password != &challenge.Password {
		return &pb.GetFullChallengeDataResponse{
			Success:   false,
			Message:   "Invalid password",
			ErrorType: "UNAUTHORIZED",
		}, nil
	}

	return &pb.GetFullChallengeDataResponse{
		Success:   true,
		Message:   "Success",
		Challenge: challenge,
	}, nil
}

func (s *ChallengeServiceServer) GetChallengeHistory(ctx context.Context, req *pb.GetChallengeHistoryRequest) (*pb.GetChallengeHistoryResponse, error) {
	userHistory, exists := s.repo.history[req.UserId]
	if !exists {
		userHistory = []*pb.ChallengeRecord{}
	}

	// Filter by private if specified
	var filtered []*pb.ChallengeRecord
	for _, ch := range userHistory {
		if !req.IsPrivate || ch.IsPrivate {
			filtered = append(filtered, ch)
		}
	}

	// Pagination
	page := int(req.Pagination.Page)
	pageSize := int(req.Pagination.PageSize)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}
	start := (page - 1) * pageSize
	end := start + pageSize
	if start > len(filtered) {
		filtered = []*pb.ChallengeRecord{}
	} else if end > len(filtered) {
		end = len(filtered)
	}
	paginated := filtered[start:end]

	list := &pb.ChallengeListResponse{
		Challenges: paginated,
		TotalCount: int64(len(filtered)),
	}

	return &pb.GetChallengeHistoryResponse{
		Success: true,
		Message: "Success",
		List:    list,
	}, nil
}

func (s *ChallengeServiceServer) GetActiveOpenChallenges(ctx context.Context, req *pb.PaginationRequest) (*pb.GetActiveOpenChallengesResponse, error) {
	var activeOpen []*pb.ChallengeRecord
	for _, ch := range s.repo.challenges {
		if ch.Status == "ACTIVE" && !ch.IsPrivate {
			activeOpen = append(activeOpen, ch)
		}
	}

	// Pagination
	page := int(req.Page)
	pageSize := int(req.PageSize)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}
	start := (page - 1) * pageSize
	end := start + pageSize
	if start > len(activeOpen) {
		activeOpen = []*pb.ChallengeRecord{}
	} else if end > len(activeOpen) {
		end = len(activeOpen)
	}
	paginated := activeOpen[start:end]

	list := &pb.ChallengeListResponse{
		Challenges: paginated,
		TotalCount: int64(len(activeOpen)),
	}

	return &pb.GetActiveOpenChallengesResponse{
		Success: true,
		Message: "Success",
		List:    list,
	}, nil
}

func (s *ChallengeServiceServer) GetOwnersActiveChallenges(ctx context.Context, req *pb.GetOwnersActiveChallengesRequest) (*pb.GetOwnersActiveChallengesResponse, error) {
	ownerChallenges, exists := s.repo.ownerChallenges[req.UserId]
	if !exists {
		ownerChallenges = []*pb.ChallengeRecord{}
	}

	var active []*pb.ChallengeRecord
	for _, ch := range ownerChallenges {
		if ch.Status == "ACTIVE" {
			active = append(active, ch)
		}
	}

	// Pagination
	page := int(req.Pagination.Page)
	pageSize := int(req.Pagination.PageSize)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}
	start := (page - 1) * pageSize
	end := start + pageSize
	if start > len(active) {
		active = []*pb.ChallengeRecord{}
	} else if end > len(active) {
		end = len(active)
	}
	paginated := active[start:end]

	list := &pb.ChallengeListResponse{
		Challenges: paginated,
		TotalCount: int64(len(active)),
	}

	return &pb.GetOwnersActiveChallengesResponse{
		Success: true,
		Message: "Success",
		List:    list,
	}, nil
}

func (s *ChallengeServiceServer) PushSubmissionStatus(ctx context.Context, req *pb.PushSubmissionStatusRequest) (*pb.PushSubmissionStatusResponse, error) {
	challenge, exists := s.repo.challenges[req.ChallengeId]
	if !exists {
		return &pb.PushSubmissionStatusResponse{
			Success:   false,
			Message:   "Challenge not found",
			ErrorType: "NOT_FOUND",
		}, nil
	}

	// Find or create user submissions
	var userSubs *pb.UserSubmissions
	for i := range challenge.Submissions {
		if challenge.Submissions[i].UserId == req.UserId {
			userSubs = challenge.Submissions[i]
			break
		}
	}
	if userSubs == nil {
		userSubs = &pb.UserSubmissions{
			UserId: req.UserId,
		}
		challenge.Submissions = append(challenge.Submissions, userSubs)
	}

	// Add submission entry
	subMeta := &pb.SubmissionMetadata{
		SubmissionId:    req.SubmissionId,
		TimeTakenMillis: req.TimeTakenMillis,
		Points:          req.Score,
		UserCode:        req.UserCode,
	}
	entry := &pb.SubmissionEntry{
		ProblemId:  req.ProblemId,
		Submission: subMeta,
	}
	userSubs.Entries = append(userSubs.Entries, entry)

	// Update participant metadata if participant exists
	if participant, ok := challenge.Participants[req.UserId]; ok {
		if probMeta, ok := participant.ProblemsDone[req.ProblemId]; ok && req.IsSuccessful {
			probMeta.Score = req.Score
			probMeta.TimeTaken = req.TimeTakenMillis
			probMeta.CompletedAtUnix = time.Now().Unix()
		} else if req.IsSuccessful {
			participant.ProblemsDone[req.ProblemId] = &pb.ChallengeProblemMetadata{
				ProblemId:       req.ProblemId,
				Score:           req.Score,
				TimeTaken:       req.TimeTakenMillis,
				CompletedAtUnix: time.Now().Unix(),
			}
			participant.ProblemsAttempted++
			participant.TotalScore += int32(req.Score)
		}
		participant.LastConnectedUnix = time.Now().Unix()
	} else if req.IsSuccessful {
		// Add new participant
		newParticipant := &pb.ParticipantMetadata{
			JoinTimeUnix: time.Now().Unix(),
			ProblemsDone: map[string]*pb.ChallengeProblemMetadata{
				req.ProblemId: {
					ProblemId:       req.ProblemId,
					Score:           req.Score,
					TimeTaken:       req.TimeTakenMillis,
					CompletedAtUnix: time.Now().Unix(),
				},
			},
			ProblemsAttempted: 1,
			TotalScore:        int32(req.Score),
			LastConnectedUnix: time.Now().Unix(),
		}
		challenge.Participants[req.UserId] = newParticipant
	}

	// Update leaderboard (simple sort by totalScore descending)
	s.updateLeaderboard(challenge)

	log.Printf("Pushed submission for user %s on problem %s in challenge %s", req.UserId, req.ProblemId, req.ChallengeId)

	return &pb.PushSubmissionStatusResponse{
		Success: true,
		Message: "Submission status pushed successfully",
	}, nil
}

func (s *ChallengeServiceServer) GetPostChallengeData(ctx context.Context, req *pb.GetPostChallengeDataRequest) (*pb.GetPostChallengeDataResponse, error) {
	postChallenge, exists := s.repo.postChallenges[req.ChallengeId]
	if !exists {
		return &pb.GetPostChallengeDataResponse{
			Success:   false,
			Message:   "Post challenge not found",
			ErrorType: "NOT_FOUND",
		}, nil
	}

	// Simulate auth check with userId (in real impl, check permissions)
	if postChallenge.CreatorId != req.UserId {
		// For now, allow read if not creator; add proper auth
	}

	return &pb.GetPostChallengeDataResponse{
		Success:   true,
		Message:   "Success",
		Challenge: postChallenge,
	}, nil
}

// Helper to update leaderboard
func (s *ChallengeServiceServer) updateLeaderboard(challenge *pb.ChallengeRecord) {
	entries := []*pb.LeaderboardEntry{}
	for userId, participant := range challenge.Participants {
		entries = append(entries, &pb.LeaderboardEntry{
			UserId:            userId,
			ProblemsCompleted: int32(len(participant.ProblemsDone)),
			TotalScore:        participant.TotalScore,
		})
	}

	// Sort by totalScore descending
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].TotalScore > entries[j].TotalScore
	})

	// Assign ranks
	for i, entry := range entries {
		entry.Rank = int32(i + 1)
	}

	challenge.Leaderboard = entries
}

// Note: In a real app, handle errors with status.Error(codes.InvalidArgument, msg) instead of just returning nil error.
