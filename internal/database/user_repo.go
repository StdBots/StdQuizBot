package database

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// EnsureUser creates or updates user profile in database
func (db *DB) EnsureUser(ctx context.Context, userID int64, username, firstName string) (*User, error) {
	now := time.Now()
	update := bson.M{
		"$set": bson.M{
			"username":   username,
			"first_name": firstName,
			"updated_at": now,
		},
		"$setOnInsert": bson.M{
			"user_id":         userID,
			"quizzes_created": 0,
			"quizzes_played":  0,
			"created_at":      now,
		},
	}
	opts := options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)
	var user User
	err := db.Users.FindOneAndUpdate(ctx, bson.M{"user_id": userID}, update, opts).Decode(&user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetSystemStats returns aggregate statistics for admin dashboard
func (db *DB) GetSystemStats(ctx context.Context) (*SystemStats, error) {
	totalUsers, err := db.Users.CountDocuments(ctx, bson.M{})
	if err != nil {
		return nil, err
	}

	totalQuizzes, err := db.Quizzes.CountDocuments(ctx, bson.M{})
	if err != nil {
		return nil, err
	}

	totalPlays, err := db.Scores.CountDocuments(ctx, bson.M{})
	if err != nil {
		return nil, err
	}

	return &SystemStats{
		TotalUsers:   totalUsers,
		TotalQuizzes: totalQuizzes,
		TotalPlays:   totalPlays,
	}, nil
}

// GetAllUserIDs retrieves user IDs for admin broadcasts
func (db *DB) GetAllUserIDs(ctx context.Context) ([]int64, error) {
	cursor, err := db.Users.Find(ctx, bson.M{}, options.Find().SetProjection(bson.M{"user_id": 1}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var ids []int64
	for cursor.Next(ctx) {
		var u struct {
			UserID int64 `bson:"user_id"`
		}
		if err := cursor.Decode(&u); err == nil {
			ids = append(ids, u.UserID)
		}
	}
	return ids, nil
}
