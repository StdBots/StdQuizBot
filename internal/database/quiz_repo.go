package database

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// GenerateQuizID creates a unique random 8-character identifier
func GenerateQuizID() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// CreateQuiz initializes a new quiz record in MongoDB
func (db *DB) CreateQuiz(ctx context.Context, creatorID int64, creatorName, title string) (*Quiz, error) {
	now := time.Now()
	quiz := &Quiz{
		ID:          GenerateQuizID(),
		CreatorID:   creatorID,
		CreatorName: creatorName,
		Title:       title,
		Description: "",
		Questions:   []Question{},
		Timer:       30,     // Default 30s
		Shuffle:     "none", // Default no shuffle
		TotalPlays:  0,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	_, err := db.Quizzes.InsertOne(ctx, quiz)
	if err != nil {
		return nil, err
	}

	// Update user stats
	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = db.Users.UpdateOne(bgCtx, bson.M{"user_id": creatorID}, bson.M{
			"$inc": bson.M{"quizzes_created": 1},
			"$set": bson.M{"updated_at": now},
		})
	}()

	return quiz, nil
}

// GetQuiz retrieves a quiz by its unique ID
func (db *DB) GetQuiz(ctx context.Context, quizID string) (*Quiz, error) {
	var quiz Quiz
	err := db.Quizzes.FindOne(ctx, bson.M{"quiz_id": quizID}).Decode(&quiz)
	if err != nil {
		return nil, err
	}
	return &quiz, nil
}

// UpdateQuiz updates fields of a quiz
func (db *DB) UpdateQuiz(ctx context.Context, quizID string, update bson.M) error {
	update["updated_at"] = time.Now()
	_, err := db.Quizzes.UpdateOne(ctx, bson.M{"quiz_id": quizID}, bson.M{"$set": update})
	return err
}

// DeleteQuiz removes a quiz if owned by creator or admin
func (db *DB) DeleteQuiz(ctx context.Context, quizID string, creatorID int64) error {
	filter := bson.M{"quiz_id": quizID}
	if creatorID > 0 {
		filter["creator_id"] = creatorID
	}
	_, err := db.Quizzes.DeleteOne(ctx, filter)
	return err
}

// GetUserQuizzes retrieves all quizzes created by a user
func (db *DB) GetUserQuizzes(ctx context.Context, creatorID int64) ([]Quiz, error) {
	findOpts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	cursor, err := db.Quizzes.Find(ctx, bson.M{"creator_id": creatorID}, findOpts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var quizzes []Quiz
	if err := cursor.All(ctx, &quizzes); err != nil {
		return nil, err
	}
	return quizzes, nil
}

// PushQuestion adds a new question to the quiz
func (db *DB) PushQuestion(ctx context.Context, quizID string, q Question) error {
	_, err := db.Quizzes.UpdateOne(ctx, bson.M{"quiz_id": quizID}, bson.M{
		"$push": bson.M{"questions": q},
		"$set":  bson.M{"updated_at": time.Now()},
	})
	return err
}

// PopQuestion removes the last added question
func (db *DB) PopQuestion(ctx context.Context, quizID string) (bool, error) {
	quiz, err := db.GetQuiz(ctx, quizID)
	if err != nil || len(quiz.Questions) == 0 {
		return false, err
	}

	newQs := quiz.Questions[:len(quiz.Questions)-1]
	err = db.UpdateQuiz(ctx, quizID, bson.M{"questions": newQs})
	return err == nil, err
}

// DeleteQuestionByIndex removes a specific question by 0-based index
func (db *DB) DeleteQuestionByIndex(ctx context.Context, quizID string, index int) error {
	quiz, err := db.GetQuiz(ctx, quizID)
	if err != nil || index < 0 || index >= len(quiz.Questions) {
		return err
	}

	newQs := append(quiz.Questions[:index], quiz.Questions[index+1:]...)
	return db.UpdateQuiz(ctx, quizID, bson.M{"questions": newQs})
}

// SavePlayerScore stores a completed game record
func (db *DB) SavePlayerScore(ctx context.Context, score PlayerScore) error {
	score.PlayedAt = time.Now()
	_, err := db.Scores.InsertOne(ctx, score)

	// Increment user games played
	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = db.Users.UpdateOne(bgCtx, bson.M{"user_id": score.UserID}, bson.M{
			"$inc": bson.M{"quizzes_played": 1},
			"$set": bson.M{"updated_at": time.Now()},
		})
	}()

	return err
}

// GetQuizLeaderboard retrieves top players for a specific quiz
func (db *DB) GetQuizLeaderboard(ctx context.Context, quizID string, limit int) ([]PlayerScore, error) {
	if limit <= 0 {
		limit = 10
	}
	findOpts := options.Find().
		SetSort(bson.D{{Key: "score", Value: -1}, {Key: "avg_time", Value: 1}}).
		SetLimit(int64(limit))

	cursor, err := db.Scores.Find(ctx, bson.M{"quiz_id": quizID}, findOpts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var scores []PlayerScore
	if err := cursor.All(ctx, &scores); err != nil {
		return nil, err
	}
	return scores, nil
}

// IncrementQuizPlays tracks total plays
func (db *DB) IncrementQuizPlays(ctx context.Context, quizID string) {
	_, _ = db.Quizzes.UpdateOne(ctx, bson.M{"quiz_id": quizID}, bson.M{
		"$inc": bson.M{"total_plays": 1},
	})
}
