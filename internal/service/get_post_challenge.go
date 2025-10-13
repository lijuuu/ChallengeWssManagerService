package service

import (
	"context"

	challengePb "github.com/lijuuu/GlobalProtoXcode/ChallengeService"
)

func (s *ChallengeService) GetPostChallengeData(ctx context.Context, req *challengePb.GetPostChallengeDataRequest) (*challengePb.GetPostChallengeDataResponse, error) {
	if req == nil || req.ChallengeId == "" {
		return &challengePb.GetPostChallengeDataResponse{Success: false, Message: "invalid request"}, nil
	}

	ch, err := s.MongoRepo.GetChallengeByID(ctx, req.ChallengeId)
	if err != nil {
		return &challengePb.GetPostChallengeDataResponse{Success: false, Message: "challenge not found"}, nil
	}

	out := &challengePb.PostChallengeDocument{
		ChallengeId: ch.ChallengeID,
		CreatorId:   ch.CreatorID,
		CreatedAt:   ch.CreatedAt,
		Title:       ch.Title,
		IsPrivate:   ch.IsPrivate,
		Status:      ch.Status,
		TimeLimit:   ch.TimeLimit,
		StartTime:   ch.StartTime,
	}

	if len(ch.Notifications) > 0 {
		out.Notifications = make([]*challengePb.PostChallengeNotification, 0, len(ch.Notifications))
		for _, n := range ch.Notifications {
			out.Notifications = append(out.Notifications, &challengePb.PostChallengeNotification{
				Type:    n.Type,
				Message: n.Message,
				Time:    n.Time,
			})
		}
	}

	if len(ch.Chat) > 0 {
		out.Chat = make([]*challengePb.PostChallengeChatMessage, 0, len(ch.Chat))
		for _, m := range ch.Chat {
			out.Chat = append(out.Chat, &challengePb.PostChallengeChatMessage{
				UserId:     m.UserID,
				ProfilePic: m.ProfilePic,
				Message:    m.Message,
				Time:       m.Time,
			})
		}
	}

	return &challengePb.GetPostChallengeDataResponse{Success: true, Challenge: out}, nil
}
