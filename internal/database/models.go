package database

import "time"

// Question represents a single quiz question with options and answers
type Question struct {
	Question     string   `bson:"question" json:"question"`
	Options      []string `bson:"options" json:"options"`
	CorrectIndex int      `bson:"correct_index" json:"correct_index"`
	Explanation  string   `bson:"explanation" json:"explanation"`
	IntroText    string   `bson:"intro_text,omitempty" json:"intro_text,omitempty"`
}

// Quiz represents a complete saved quiz
type Quiz struct {
	ID          string     `bson:"quiz_id" json:"quiz_id"`
	CreatorID   int64      `bson:"creator_id" json:"creator_id"`
	CreatorName string     `bson:"creator_name" json:"creator_name"`
	Title       string     `bson:"title" json:"title"`
	Description string     `bson:"description" json:"description"`
	Questions   []Question `bson:"questions" json:"questions"`
	Timer       int        `bson:"timer" json:"timer"`     // Time per question in seconds
	Shuffle     string     `bson:"shuffle" json:"shuffle"` // "none", "all", "questions", "answers"
	TotalPlays  int64      `bson:"total_plays" json:"total_plays"`
	CreatedAt   time.Time  `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time  `bson:"updated_at" json:"updated_at"`
}

// PlayerScore logs a game completion record for leaderboard stats
type PlayerScore struct {
	QuizID   string    `bson:"quiz_id" json:"quiz_id"`
	ChatID   int64     `bson:"chat_id" json:"chat_id"`
	UserID   int64     `bson:"user_id" json:"user_id"`
	Username string    `bson:"username" json:"username"`
	Name     string    `bson:"name" json:"name"`
	Score    int       `bson:"score" json:"score"`
	Correct  int       `bson:"correct" json:"correct"`
	Total    int       `bson:"total" json:"total"`
	AvgTime  float64   `bson:"avg_time" json:"avg_time"`
	PlayedAt time.Time `bson:"played_at" json:"played_at"`
}

// User tracks bot users and creators
type User struct {
	UserID         int64     `bson:"user_id" json:"user_id"`
	Username       string    `bson:"username" json:"username"`
	FirstName      string    `bson:"first_name" json:"first_name"`
	QuizzesCreated int64     `bson:"quizzes_created" json:"quizzes_created"`
	QuizzesPlayed  int64     `bson:"quizzes_played" json:"quizzes_played"`
	CreatedAt      time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt      time.Time `bson:"updated_at" json:"updated_at"`
}

// SystemStats for admin analytics
type SystemStats struct {
	TotalUsers   int64 `json:"total_users"`
	TotalQuizzes int64 `json:"total_quizzes"`
	TotalPlays   int64 `json:"total_plays"`
}
