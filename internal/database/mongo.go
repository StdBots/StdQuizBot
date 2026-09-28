package database

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// DB encapsulates MongoDB client and collection references
type DB struct {
	Client   *mongo.Client
	Database *mongo.Database
	Quizzes  *mongo.Collection
	Scores   *mongo.Collection
	Users    *mongo.Collection
}

// Connect establishes a connection to MongoDB and sets up indexes
func Connect(uri, dbName string) (*DB, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	clientOpts := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return nil, err
	}

	if err := client.Ping(ctx, nil); err != nil {
		return nil, err
	}

	log.Printf("[DATABASE] Connected successfully to MongoDB (%s)", dbName)

	database := client.Database(dbName)
	db := &DB{
		Client:   client,
		Database: database,
		Quizzes:  database.Collection("quizzes"),
		Scores:   database.Collection("scores"),
		Users:    database.Collection("users"),
	}

	db.ensureIndexes()

	return db, nil
}

func (db *DB) ensureIndexes() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 1. Quizzes: Unique QuizID
	_, _ = db.Quizzes.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "quiz_id", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	_, _ = db.Quizzes.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "creator_id", Value: 1}},
	})

	// 2. Scores: QuizID + Score sorting
	_, _ = db.Scores.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "quiz_id", Value: 1}, {Key: "score", Value: -1}},
	})

	// 3. Users: Unique UserID
	_, _ = db.Users.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "user_id", Value: 1}},
		Options: options.Index().SetUnique(true),
	})

	log.Printf("[DATABASE] MongoDB indexes established.")
}

// Disconnect closes client connection
func (db *DB) Disconnect() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = db.Client.Disconnect(ctx)
}
