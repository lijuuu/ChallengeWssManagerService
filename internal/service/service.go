package service

import (
	"context"
	"time"

	"github.com/lijuuu/ChallengeWssManagerService/internal/model"
	"github.com/lijuuu/ChallengeWssManagerService/internal/repo"
	challengePb "github.com/lijuuu/GlobalProtoXcode/ChallengeService"
)

type ChallengeService struct {
	repo *repo.MongoRepository
	challengePb.UnimplementedChallengeServiceServer
}

func NewChallengeService(repo *repo.MongoRepository) *ChallengeService {
	return &ChallengeService{repo: repo}
}

func (s *ChallengeService) CreateChallenge(ctx context.Context, req *challengePb.ChallengeRecord) (*challengePb.ChallengeRecord, error) {
	modelChallenge := ChallengeFromProto(req)


	modelChallenge.Status = model.StatusWaiting
	//push creator to participant list
	modelChallenge.Participants[modelChallenge.CreatorID] = &model.ParticipantMetadata{
		UserID: modelChallenge.CreatorID,
		JoinTime: time.Now().Unix(),
	 
	}

	if err := s.repo.CreateChallenge(ctx, modelChallenge); err != nil {
		return nil, err
	}
	return req, nil
}

func (s *ChallengeService) GetPublicChallenges(ctx context.Context, req *challengePb.PaginationRequest) (*challengePb.ChallengeListResponse, error) {
	challenges, err := s.repo.GetPublicChallenges(ctx, int(req.Page), int(req.PageSize))
	if err != nil {
		return nil, err
	}
	return &challengePb.ChallengeListResponse{
		Challenges: ChallengesToProto(toPtrSlice(challenges)),
		TotalCount: int64(len(challenges)),
	}, nil
}

func (s *ChallengeService) GetPrivateChallengesOfUser(ctx context.Context, req *challengePb.PrivateChallengesRequest) (*challengePb.ChallengeListResponse, error) {
	challenges, err := s.repo.GetPrivateChallengesOfUser(ctx, req.UserId, int(req.Pagination.Page), int(req.Pagination.PageSize))
	if err != nil {
		return nil, err
	}
	return &challengePb.ChallengeListResponse{
		Challenges: ChallengesToProto(toPtrSlice(challenges)),
		TotalCount: int64(len(challenges)),
	}, nil
}

func (s *ChallengeService) GetActiveChallenges(ctx context.Context, req *challengePb.PaginationRequest) (*challengePb.ChallengeListResponse, error) {
	challenges, err := s.repo.GetActiveChallenges(ctx, int(req.Page), int(req.PageSize))
	if err != nil {
		return nil, err
	}
	return &challengePb.ChallengeListResponse{
		Challenges: ChallengesToProto(toPtrSlice(challenges)),
		TotalCount: int64(len(challenges)),
	}, nil
}

func (s *ChallengeService) GetUserChallenges(ctx context.Context, req *challengePb.UserChallengesRequest) (*challengePb.ChallengeListResponse, error) {
	challenges, err := s.repo.GetUserChallenges(ctx, req.UserId, int(req.Pagination.Page), int(req.Pagination.PageSize))
	if err != nil {
		return nil, err
	}
	return &challengePb.ChallengeListResponse{
		Challenges: ChallengesToProto(toPtrSlice(challenges)),
		TotalCount: int64(len(challenges)),
	}, nil
}

func (s *ChallengeService) PushSubmissionStatus(ctx context.Context, req *challengePb.PushSubmissionStatusRequest) (*challengePb.PushSubmissionStatusResponse, error) {
	return &challengePb.PushSubmissionStatusResponse{Message: "received", Success: true}, nil
}

func ChallengeFromProto(pb *challengePb.ChallengeRecord) *model.Challenge {
	participants := make(map[string]*model.ParticipantMetadata)
	for k, v := range pb.Participants {
		participants[k] = &model.ParticipantMetadata{
			UserID:            v.UserId,
			ProblemsDone:      nil,
			LastConnected:     v.LastConnectedUnix,
			ProblemsAttempted: int(v.ProblemsAttempted),
			TotalScore:        int(v.TotalScore),
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

	return &model.Challenge{
		ChallengeID:  pb.GetChallengeId(),
		CreatorID:    pb.GetCreatorId(),
		Title:        pb.GetTitle(),
		IsPrivate:    pb.GetIsPrivate(),
		Password:     pb.GetPassword(),
		Status:       model.ChallengeStatus(pb.GetStatus()),
		TimeLimit:    pb.GetTimeLimitMillis(),
		StartTime:    pb.StartTimeUnix,
		Participants: participants,
		Submissions:  submissions,
	}
}

func ChallengesToProto(challenges []*model.Challenge) []*challengePb.ChallengeRecord {
	protoChallenges := make([]*challengePb.ChallengeRecord, 0, len(challenges))
	for _, ch := range challenges {
		record := &challengePb.ChallengeRecord{
			ChallengeId:     ch.ChallengeID,
			CreatorId:       ch.CreatorID,
			Title:           ch.Title,
			IsPrivate:       ch.IsPrivate,
			Password:        ch.Password,
			Status:          string(ch.Status),
			TimeLimitMillis: ch.TimeLimit,
			StartTimeUnix:   ch.StartTime,
			Participants:    make(map[string]*challengePb.ParticipantMetadata),
			Submissions:     make([]*challengePb.UserSubmissions, 0),
		}
		for k, v := range ch.Participants {
			record.Participants[k] = &challengePb.ParticipantMetadata{
				UserId:            v.UserID,
				LastConnectedUnix: v.LastConnected,
				ProblemsAttempted: int32(v.ProblemsAttempted),
				TotalScore:        int32(v.TotalScore),
				ProblemsDone:      nil,
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

func toPtrSlice(in []model.Challenge) []*model.Challenge {
	out := make([]*model.Challenge, len(in))
	for i := range in {
		out[i] = &in[i]
	}
	return out
}
