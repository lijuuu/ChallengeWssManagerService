package service

import (
	"time"

	"github.com/lijuuu/ChallengeWssManagerService/internal/model"
	challengePb "github.com/lijuuu/GlobalProtoXcode/ChallengeService"
)

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
		Status:              (pb.GetStatus()),
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
			record.Submissions = append(record.Submissions, &challengePb.UserSubmissions{UserId: userId, Entries: entries})
		}

		protoChallenges = append(protoChallenges, record)
	}
	return protoChallenges
}

func ToPtrSlice(in []model.ChallengeDocument) []*model.ChallengeDocument {
	out := make([]*model.ChallengeDocument, len(in))
	for i := range in {
		out[i] = &in[i]
	}
	return out
}
