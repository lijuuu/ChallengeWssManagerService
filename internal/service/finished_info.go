//go:build proto_finished

package service

// import (
// 	"context"

// 	challengePb "github.com/lijuuu/GlobalProtoXcode/ChallengeService"
// )

// // GetFinishedChallengeInfo returns finished challenge info only if the requester participated (or is creator).
// // Note: This file is behind the `proto_finished` build tag until the proto message is added to the proto repo.
// func (s *ChallengeService) GetFinishedChallengeInfo(ctx context.Context, req *challengePb.FinishedChallengeInfoRequest) (*challengePb.FinishedChallengeInfoResponse, error) {
// 	ch, err := s.mongoRepo.GetChallengeByID(ctx, req.GetChallengeId())
// 	if err != nil {
// 		return &challengePb.FinishedChallengeInfoResponse{Success: false, Message: "not found"}, nil
// 	}

// 	// allow creator or participant only
// 	userID := req.GetUserId()
// 	if ch.CreatorID != userID {
// 		if _, ok := ch.Participants[userID]; !ok {
// 			return &challengePb.FinishedChallengeInfoResponse{Success: false, Message: "forbidden"}, nil
// 		}
// 	}

// 	// participants
// 	parts := map[string]*challengePb.ParticipantSummary{}
// 	for uid, p := range ch.Participants {
// 		parts[uid] = &challengePb.ParticipantSummary{
// 			UserId:            uid,
// 			TotalScore:        int32(p.TotalScore),
// 			ProblemsAttempted: int32(p.ProblemsAttempted),
// 			JoinTimeUnix:      p.JoinTime,
// 			LastConnectedUnix: p.LastConnected,
// 		}
// 	}

// 	// submissions: user -> problem -> submission
// 	subs := make(map[string]*challengePb.FinishedChallengeInfoResponse_SubmissionsEntry)
// 	for uid, perUser := range ch.Submissions {
// 		inner := make(map[string]*challengePb.SubmissionSummary)
// 		for pid, sMeta := range perUser {
// 			inner[pid] = &challengePb.SubmissionSummary{
// 				SubmissionId:    sMeta.SubmissionID,
// 				TimeTakenMillis: int64(sMeta.TimeTaken.Milliseconds()),
// 				Points:          int32(sMeta.Points),
// 			}
// 		}
// 		subs[uid] = &challengePb.FinishedChallengeInfoResponse_SubmissionsEntry{Value: inner}
// 	}

// 	// leaderboard
// 	lb := make([]*challengePb.LeaderboardEntryPb, 0, len(ch.Leaderboard))
// 	for _, e := range ch.Leaderboard {
// 		lb = append(lb, &challengePb.LeaderboardEntryPb{
// 			UserId:            e.UserID,
// 			ProblemsCompleted: int32(e.ProblemsCompleted),
// 			TotalScore:        int32(e.TotalScore),
// 			Rank:              int32(e.Rank),
// 		})
// 	}

// 	// notifications
// 	notifs := make([]*challengePb.NotificationPb, 0, len(ch.Notifications))
// 	for _, n := range ch.Notifications {
// 		notifs = append(notifs, &challengePb.NotificationPb{Type: n.Type, Message: n.Message, TimeUnix: n.Time})
// 	}

// 	// chat
// 	chat := make([]*challengePb.ChatMessagePb, 0, len(ch.Chat))
// 	for _, m := range ch.Chat {
// 		chat = append(chat, &challengePb.ChatMessagePb{UserId: m.UserID, ProfilePic: m.ProfilePic, Message: m.Message, TimeUnix: m.Time})
// 	}

// 	return &challengePb.FinishedChallengeInfoResponse{
// 		Success: true,
// 		Message: "ok",
// 		Challenge: &challengePb.FinishedChallengeFull{
// 			ChallengeId:     ch.ChallengeID,
// 			Title:           ch.Title,
// 			CreatorId:       ch.CreatorID,
// 			IsPrivate:       ch.IsPrivate,
// 			Status:          ch.Status,
// 			CreatedAtUnix:   ch.CreatedAt,
// 			StartTimeUnix:   ch.StartTime,
// 			TimeLimitMillis: ch.TimeLimit,
// 			Participants:    parts,
// 			Submissions:     subs,
// 			Leaderboard:     lb,
// 			Notifications:   notifs,
// 			Chat:            chat,
// 		},
// 	}, nil
// }
