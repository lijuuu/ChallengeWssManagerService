package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/lijuuu/ChallengeWssManagerService/internal/model"
	"github.com/lijuuu/ChallengeWssManagerService/internal/repo"
	"github.com/lijuuu/ChallengeWssManagerService/internal/utils"
	"github.com/lijuuu/ChallengeWssManagerService/internal/wss/broadcasts"
	wsstypes "github.com/lijuuu/ChallengeWssManagerService/internal/wss/types"
	challengePb "github.com/lijuuu/GlobalProtoXcode/ChallengeService"
)

type ChallengeService struct {
	repo           *repo.MongoRepository
	websocketState *wsstypes.State
	challengePb.UnimplementedChallengeServiceServer
}

func NewChallengeService(repo *repo.MongoRepository, websocketState *wsstypes.State) *ChallengeService {
	return &ChallengeService{repo: repo, websocketState: websocketState}
}

func (s *ChallengeService) CreateChallenge(ctx context.Context, req *challengePb.ChallengeRecord) (*challengePb.ChallengeRecord, error) {

	challenges, err := s.repo.GetActiveOpenChallenges(ctx, 1, 1) //TODO:(P1) changes this to check the  specific user and then check on state wss for any current repo, we need mutex locks
	if err != nil {
		return nil, err
	}
	if len(challenges) != 0 {
		return nil, errors.New("active challenge already found, can't create new challenge")
	}

	modelChallenge := ChallengeDocumentFromProto(req, false)

	modelChallenge.Status = model.ChallengeOpen
	modelChallenge.Participants[modelChallenge.CreatorID] = &model.ParticipantMetadata{
		JoinTime: time.Now().Unix(),
	}

	if req.IsPrivate {
		modelChallenge.Password = utils.GenerateBigCapPassword(7)
	}

	modelChallenge.Leaderboard = append(modelChallenge.Leaderboard, &model.LeaderboardEntry{
		UserID:            req.CreatorId,
		TotalScore:        0,
		Rank:              0,
		ProblemsCompleted: 0,
	})

	modelChallenge.Config = &model.ChallengeConfig{
		MaxEasyQuestions:   int(req.GetConfig().GetMaxEasyQuestions()),
		MaxMediumQuestions: int(req.GetConfig().GetMaxMediumQuestions()),
		MaxHardQuestions:   int(req.GetConfig().GetMaxHardQuestions()),
		MaxUsers:           int(req.GetConfig().MaxUsers),
	}

	if err := s.repo.CreateChallenge(ctx, modelChallenge); err != nil {
		return nil, err
	}
	return req, nil
}

func (s *ChallengeService) LeaveChallenge(ctx context.Context, challengeId, userId string) bool {
	// Fetch the challenge to verify the creator
	challenge, err := s.repo.GetChallengeByID(ctx, challengeId)
	if err != nil {
		return false
	}

	if challenge.CreatorID != userId {
		return false
	}

	if err := s.repo.RemoveParticipantInJoinPhase(ctx, challengeId, userId); err != nil {
		return false
	}

	return true
}

func (s *ChallengeService) AbandonChallenge(ctx context.Context, req *challengePb.AbandonChallengeRequest) (*challengePb.AbandonChallengeResponse, error) {
	// Fetch the challenge to verify the creator
	challenge, err := s.repo.GetChallengeByID(ctx, req.ChallengeId)
	if err != nil {
		return &challengePb.AbandonChallengeResponse{
			Success:   false,
			Message:   "Challenge not found",
			ErrorType: "CHALLENGENOTFOUND",
		}, err
	}

	if challenge.CreatorID != req.CreatorId {
		return &challengePb.AbandonChallengeResponse{
			Success:   false,
			Message:   "Only the creator can abandon the challenge",
			ErrorType: "NOTCREATOR",
		}, nil
	}

	if err := s.repo.AbandonChallenge(ctx, req.CreatorId, req.ChallengeId); err != nil {
		return &challengePb.AbandonChallengeResponse{
			Success:   false,
			Message:   err.Error(),
			ErrorType: "CHALLENGEABANDONFAILED",
		}, err
	}

	// Check for nil websocketState or Challenges map
	if s.websocketState == nil || s.websocketState.Challenges == nil {
		// Log the issue (consider adding a proper logger instead of fmt)
		fmt.Printf("Warning: websocketState or Challenges map is nil for challenge ID %s\n", req.ChallengeId)
		return &challengePb.AbandonChallengeResponse{Success: true}, nil
	}

	// Check if the challenge exists in the WebSocket state
	challengeState, exists := s.websocketState.Challenges[challenge.ChallengeID]
	if !exists {
		// Log the issue
		fmt.Printf("Warning: Challenge ID %s not found in websocketState.Challenges\n", challenge.ChallengeID)
		return &challengePb.AbandonChallengeResponse{Success: true}, nil
	}

	// Broadcast the abandon event
	broadcasts.BroadcastChallengeAbandon(challengeState)

	return &challengePb.AbandonChallengeResponse{Success: true}, nil
}
func (s *ChallengeService) GetFullChallengeData(ctx context.Context, req *challengePb.GetFullChallengeDataRequest) (*challengePb.GetFullChallengeDataResponse, error) {
	challenge, err := s.repo.GetChallengeByID(ctx, req.ChallengeId)
	if err != nil {
		return nil, err
	}

	return &challengePb.GetFullChallengeDataResponse{
		Challenge: ChallengesToProto([]*model.ChallengeDocument{&challenge}, false)[0],
	}, nil
}

func (s *ChallengeService) GetChallengeHistory(ctx context.Context, req *challengePb.GetChallengeHistoryRequest) (*challengePb.ChallengeListResponse, error) {
	challenges, err := s.repo.GetChallengeHistory(ctx, req.UserId, int(req.GetPagination().GetPage()), int(req.GetPagination().GetPageSize()), req.GetIsPrivate())
	if err != nil {
		return nil, err
	}
	return &challengePb.ChallengeListResponse{
		Challenges: ChallengesToProto(toPtrSlice(challenges), false),
		TotalCount: int64(len(challenges)),
	}, nil
}

func (s *ChallengeService) GetActiveOpenChallenges(ctx context.Context, req *challengePb.PaginationRequest) (*challengePb.ChallengeListResponse, error) {
	challenges, err := s.repo.GetActiveOpenChallenges(ctx, int(req.Page), int(req.PageSize))
	if err != nil {
		return nil, err
	}

	return &challengePb.ChallengeListResponse{
		Challenges: ChallengesToProto(toPtrSlice(challenges), true),
		TotalCount: int64(len(challenges)),
	}, nil
}

func (s *ChallengeService) GetOwnersActiveChallenges(ctx context.Context, req *challengePb.GetOwnersActiveChallengesRequest) (*challengePb.ChallengeListResponse, error) {
	challenges, err := s.repo.GetOwnersActiveChallenges(ctx, req.UserId, int(req.Pagination.Page), int(req.Pagination.PageSize))
	if err != nil {
		return nil, err
	}
	return &challengePb.ChallengeListResponse{
		Challenges: ChallengesToProto(toPtrSlice(challenges), false),
		TotalCount: int64(len(challenges)),
	}, nil
}

func (s *ChallengeService) PushSubmissionStatus(ctx context.Context, req *challengePb.PushSubmissionStatusRequest) (*challengePb.PushSubmissionStatusResponse, error) {
	return &challengePb.PushSubmissionStatusResponse{Message: "received", Success: true}, nil
}

func ChallengeDocumentFromProto(pb *challengePb.ChallengeRecord, hideProblems bool) *model.ChallengeDocument {
	participants := make(map[string]*model.ParticipantMetadata)
	for k, v := range pb.Participants {
		participants[k] = &model.ParticipantMetadata{
			ProblemsDone:      nil,
			LastConnected:     v.LastConnectedUnix,
			ProblemsAttempted: int(v.ProblemsAttempted),
			TotalScore:        int(v.TotalScore),
			JoinTime:          v.JoinTimeUnix,
		}
	}

	submissions := make(map[string]map[string]model.Submission)
	for _, userSub := range pb.Submissions {
		subMap := make(map[string]model.Submission)
		for _, entry := range userSub.Entries {
			subMap[entry.ProblemId] = model.Submission{
				SubmissionID: entry.Submission.SubmissionId,
				TimeTaken:    time.Duration(entry.Submission.TimeTakenMillis) * time.Millisecond,
				Points:       int(entry.Submission.Points),
			}
		}
		submissions[userSub.UserId] = subMap
	}

	var config *model.ChallengeConfig
	if pb.Config != nil {
		config = &model.ChallengeConfig{
			MaxEasyQuestions:   int(pb.GetConfig().GetMaxEasyQuestions()),
			MaxMediumQuestions: int(pb.GetConfig().GetMaxMediumQuestions()),
			MaxHardQuestions:   int(pb.GetConfig().GetMaxHardQuestions()),
			MaxUsers:           int(pb.GetConfig().GetMaxUsers()),
		}
	}

	var processedIds []string
	if !hideProblems {
		processedIds = pb.GetProcessedProblemIds()
	}

	return &model.ChallengeDocument{
		ChallengeID:         pb.GetChallengeId(),
		CreatorID:           pb.GetCreatorId(),
		CreatedAt:           pb.GetCreatedAt(),
		IsPrivate:           pb.GetIsPrivate(),
		Title:               pb.GetTitle(),
		Password:            pb.GetPassword(),
		ProcessedProblemIds: processedIds,
		ProblemCount:        int64(len(pb.GetProcessedProblemIds())),
		Status:              model.ChallengeStatus(pb.GetStatus()),
		TimeLimit:           pb.GetTimeLimitMillis(),
		StartTime:           pb.StartTimeUnix,
		Participants:        participants,
		Submissions:         submissions,
		Config:              config,
	}
}

func ChallengesToProto(challenges []*model.ChallengeDocument, hideProblems bool) []*challengePb.ChallengeRecord {
	protoChallenges := make([]*challengePb.ChallengeRecord, 0, len(challenges))
	for _, ch := range challenges {
		record := &challengePb.ChallengeRecord{
			ChallengeId:     ch.ChallengeID,
			CreatorId:       ch.CreatorID,
			CreatedAt:       ch.CreatedAt,
			Title:           ch.Title,
			IsPrivate:       ch.IsPrivate,
			Password:        ch.Password,
			Status:          string(ch.Status),
			ProblemCount:    ch.ProblemCount,
			TimeLimitMillis: ch.TimeLimit,
			StartTimeUnix:   ch.StartTime,
			Participants:    make(map[string]*challengePb.ParticipantMetadata),
			Submissions:     make([]*challengePb.UserSubmissions, 0),
		}

		if !hideProblems {
			record.ProcessedProblemIds = ch.ProcessedProblemIds
		}

		for k, v := range ch.Participants {
			record.Participants[k] = &challengePb.ParticipantMetadata{
				LastConnectedUnix: v.LastConnected,
				ProblemsAttempted: int32(v.ProblemsAttempted),
				TotalScore:        int32(v.TotalScore),
				ProblemsDone:      nil,
				JoinTimeUnix:      v.JoinTime,
			}
		}

		if ch.Config != nil {
			record.Config = &challengePb.ChallengeConfig{
				MaxEasyQuestions:   int32(ch.Config.MaxEasyQuestions),
				MaxMediumQuestions: int32(ch.Config.MaxMediumQuestions),
				MaxHardQuestions:   int32(ch.Config.MaxHardQuestions),
				MaxUsers:           int32(ch.Config.MaxUsers),
			}
		}

		for userId, subMap := range ch.Submissions {
			entries := make([]*challengePb.SubmissionEntry, 0, len(subMap))
			for problemId, sub := range subMap {
				entries = append(entries, &challengePb.SubmissionEntry{
					ProblemId: problemId,
					Submission: &challengePb.SubmissionMetadata{
						SubmissionId:    sub.SubmissionID,
						TimeTakenMillis: int64(sub.TimeTaken / time.Millisecond),
						Points:          int32(sub.Points),
					},
				})
			}
			record.Submissions = append(record.Submissions, &challengePb.UserSubmissions{
				UserId:  userId,
				Entries: entries,
			})
		}

		protoChallenges = append(protoChallenges, record)
	}
	return protoChallenges
}

func toPtrSlice(in []model.ChallengeDocument) []*model.ChallengeDocument {
	out := make([]*model.ChallengeDocument, len(in))
	for i := range in {
		out[i] = &in[i]
	}
	return out
}
