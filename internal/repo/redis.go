package repo

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/lijuuu/ChallengeWssManagerService/internal/model"
	"github.com/redis/go-redis/v9"
)

type RedisRepository struct {
	client *redis.Client
}

func NewRedisRepository(client *redis.Client) *RedisRepository {
	return &RedisRepository{
		client: client,
	}
}

// CreateChallenge stores a new challenge in Redis
func (r *RedisRepository) CreateChallenge(ctx context.Context, challenge *model.ChallengeDocument) error {
	key := fmt.Sprintf("challenge:%s", challenge.ChallengeID)

	data, err := json.Marshal(challenge)
	if err != nil {
		return fmt.Errorf("failed to marshal challenge: %w", err)
	}

	return r.client.Set(ctx, key, data, 0).Err()
}

// GetChallenge retrieves a challenge from Redis
func (r *RedisRepository) GetChallenge(ctx context.Context, challengeID string) (*model.ChallengeDocument, error) {
	challengeDoc, err := r.GetChallengeByID(ctx, challengeID)
	if err != nil {
		return nil, err
	}
	return &challengeDoc, nil
}

// UpdateChallenge updates an existing challenge in Redis
func (r *RedisRepository) UpdateChallenge(ctx context.Context, challenge *model.ChallengeDocument) error {
	return r.CreateChallenge(ctx, challenge) // Same as create for Redis
}

// DeleteChallenge removes a challenge from Redis
func (r *RedisRepository) DeleteChallenge(ctx context.Context, challengeID string) error {
	key := fmt.Sprintf("challenge:%s", challengeID)
	return r.client.Del(ctx, key).Err()
}

// GetActiveChallenges returns all challenge IDs from Redis
func (r *RedisRepository) GetActiveChallenges(ctx context.Context) ([]string, error) {
	keys, err := r.client.Keys(ctx, "challenge:*").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get challenge keys: %w", err)
	}

	challengeIDs := make([]string, len(keys))
	for i, key := range keys {
		// Extract challenge ID from key (remove "challenge:" prefix)
		challengeIDs[i] = key[10:]
	}

	return challengeIDs, nil
}

// GetChallengesByStatus returns challenge IDs filtered by status
func (r *RedisRepository) GetChallengesByStatus(ctx context.Context, status string) ([]string, error) {
	challengeIDs, err := r.GetActiveChallenges(ctx)
	if err != nil {
		return nil, err
	}

	var filteredIDs []string
	for _, id := range challengeIDs {
		challenge, err := r.GetChallenge(ctx, id)
		if err != nil {
			continue // Skip challenges that can't be retrieved
		}

		if string(challenge.Status) == status {
			filteredIDs = append(filteredIDs, id)
		}
	}

	return filteredIDs, nil
}

// AddParticipant adds a participant to a challenge
func (r *RedisRepository) AddParticipant(ctx context.Context, challengeID, userID string, metadata *model.ParticipantMetadata) error {
	challenge, err := r.GetChallenge(ctx, challengeID)
	if err != nil {
		return err
	}

	if challenge.Participants == nil {
		challenge.Participants = make(map[string]*model.ParticipantMetadata)
	}

	challenge.Participants[userID] = metadata
	return r.UpdateChallenge(ctx, challenge)
}

// RemoveParticipant removes a participant from a challenge
func (r *RedisRepository) RemoveParticipant(ctx context.Context, challengeID, userID string) error {
	challenge, err := r.GetChallenge(ctx, challengeID)
	if err != nil {
		return err
	}

	if challenge.Participants != nil {
		delete(challenge.Participants, userID)
	}

	if challenge.Submissions != nil {
		delete(challenge.Submissions, userID)
	}

	return r.UpdateChallenge(ctx, challenge)
}

// UpdateParticipant updates participant metadata
func (r *RedisRepository) UpdateParticipant(ctx context.Context, challengeID, userID string, metadata *model.ParticipantMetadata) error {
	return r.AddParticipant(ctx, challengeID, userID, metadata)
}

// GetChallengeByID retrieves a challenge document from Redis
func (r *RedisRepository) GetChallengeByID(ctx context.Context, challengeID string) (model.ChallengeDocument, error) {
	key := fmt.Sprintf("challenge:%s", challengeID)

	data, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return model.ChallengeDocument{}, fmt.Errorf("challenge not found")
		}
		return model.ChallengeDocument{}, fmt.Errorf("failed to get challenge: %w", err)
	}

	var challengeDoc model.ChallengeDocument
	if err := json.Unmarshal([]byte(data), &challengeDoc); err != nil {
		return model.ChallengeDocument{}, fmt.Errorf("failed to unmarshal challenge: %w", err)
	}

	return challengeDoc, nil
}

// AbandonChallenge updates challenge status to ABANDON in Redis
func (r *RedisRepository) AbandonChallenge(ctx context.Context, creatorID, challengeID string) error {
	challenge, err := r.GetChallenge(ctx, challengeID)
	if err != nil {
		return err
	}

	// Verify the creator
	if challenge.CreatorID != creatorID {
		return fmt.Errorf("only the creator can abandon the challenge")
	}

	// Update status to ABANDON
	challenge.Status = model.ChallengeAbandon
	return r.UpdateChallenge(ctx, challenge)
}

// RemoveParticipantInJoinPhase removes a participant during join phase
func (r *RedisRepository) RemoveParticipantInJoinPhase(ctx context.Context, challengeID, userID string) error {
	return r.RemoveParticipant(ctx, challengeID, userID)
}

// GetRedisAddr returns the Redis address from the client
func (r *RedisRepository) GetRedisAddr() string {
	return r.client.Options().Addr
}

// GetRedisPassword returns the Redis password from the client
func (r *RedisRepository) GetRedisPassword() string {
	return r.client.Options().Password
}
