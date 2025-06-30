package repo

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/lijuuu/ChallengeWssManagerService/internal/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoRepository struct {
	challenges *mongo.Collection
}

func NewMongoRepository(client *mongo.Client, dbName string) *MongoRepository {
	return &MongoRepository{
		challenges: client.Database(dbName).Collection("challenges"),
	}
}

// CreateChallenge inserts a new challenge
func (r *MongoRepository) CreateChallenge(ctx context.Context, c *model.Challenge) error {
	c.ChallengeID = uuid.New().String()
	c.Status = model.StatusWaiting
	c.StartTime = time.Now().Unix()

	_, err := r.challenges.InsertOne(ctx, c)
	return err
}

// GetPublicChallenges returns open, public challenges paginated
func (r *MongoRepository) GetPublicChallenges(ctx context.Context, page, pageSize int) ([]model.Challenge, error) {
	if page < 1 || pageSize < 1 {
		return nil, errors.New("invalid pagination")
	}

	filter := bson.M{"is_private": false}
	opts := options.Find().
		SetSort(bson.D{{Key: "start_time", Value: -1}}).
		SetSkip(int64((page - 1) * pageSize)).
		SetLimit(int64(pageSize))

	cursor, err := r.challenges.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []model.Challenge
	if err := cursor.All(ctx, &results); err != nil {
		return nil, err
	}
	return results, nil
}

// GetPrivateChallengesOfUser returns private challenges for a user (creator or participant)
func (r *MongoRepository) GetPrivateChallengesOfUser(ctx context.Context, userID string, page, pageSize int) ([]model.Challenge, error) {
	if page < 1 || pageSize < 1 || userID == "" {
		return nil, errors.New("invalid pagination or userID")
	}

	filter := bson.M{
		"is_private": true,
		"$or": []bson.M{
			{"creator_id": userID},
			{"participants." + userID: bson.M{"$exists": true}},
		},
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "start_time", Value: -1}}).
		SetSkip(int64((page - 1) * pageSize)).
		SetLimit(int64(pageSize))

	cursor, err := r.challenges.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []model.Challenge
	if err := cursor.All(ctx, &results); err != nil {
		return nil, err
	}
	return results, nil
}

// GetActiveChallenges returns challenges not marked as finished
func (r *MongoRepository) GetActiveChallenges(ctx context.Context, page, pageSize int) ([]model.Challenge, error) {
	if page < 1 || pageSize < 1 {
		return nil, errors.New("invalid pagination")
	}

	filter := bson.M{"status": bson.M{"$ne": model.StatusFinished}}
	opts := options.Find().
		SetSort(bson.D{{Key: "start_time", Value: 1}}).
		SetSkip(int64((page - 1) * pageSize)).
		SetLimit(int64(pageSize))

	cursor, err := r.challenges.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []model.Challenge
	if err := cursor.All(ctx, &results); err != nil {
		return nil, err
	}
	return results, nil
}

// GetUserChallenges returns challenges created by a specific user
func (r *MongoRepository) GetUserChallenges(ctx context.Context, userID string, page, pageSize int) ([]model.Challenge, error) {
	if page < 1 || pageSize < 1 || userID == "" {
		return nil, errors.New("invalid pagination or userID")
	}

	filter := bson.M{"creator_id": userID}
	opts := options.Find().
		SetSort(bson.D{{Key: "start_time", Value: -1}}).
		SetSkip(int64((page - 1) * pageSize)).
		SetLimit(int64(pageSize))

	cursor, err := r.challenges.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []model.Challenge
	if err := cursor.All(ctx, &results); err != nil {
		return nil, err
	}
	return results, nil
}
