// internal/models/models.go
package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// SubredditMetadata represents tracking information for monitored subreddits
type SubredditMetadata struct {
	ID             primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	SubredditName  string             `bson:"subreddit_name" json:"subreddit_name"`
	LastScrapedAt  time.Time          `bson:"last_scraped_at" json:"last_scraped_at"`
	MonitorConfig  MonitorConfig      `bson:"monitor_config" json:"monitor_config"`
	CreatedAt      time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt      time.Time          `bson:"updated_at" json:"updated_at"`
}

// MonitorConfig holds configuration for monitoring subreddits
type MonitorConfig struct {
	Enabled  bool `bson:"enabled" json:"enabled"`
	MaxPosts int  `bson:"max_posts" json:"max_posts"`
}

// SubredditConfig represents a subreddit configuration for monitoring
type SubredditConfig struct {
	ID            primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	SubredditName string             `bson:"subreddit_name" json:"subreddit_name"`
	Enabled       bool               `bson:"enabled" json:"enabled"`
	Schedule      string             `bson:"schedule" json:"schedule"`           
	MaxPosts      int                `bson:"max_posts" json:"max_posts"`
	Priority      int                `bson:"priority" json:"priority"`           // Higher number = higher priority
	Description   string             `bson:"description,omitempty" json:"description,omitempty"`
	CreatedAt     time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt     time.Time          `bson:"updated_at" json:"updated_at"`
}

// Post represents a Reddit post stored in MongoDB
type Post struct {
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	RedditID   string             `bson:"reddit_id" json:"reddit_id"`
	Title      string             `bson:"title" json:"title"`
	Body       string             `bson:"body" json:"body"`
	Author     string             `bson:"author" json:"author"`
	Score      int                `bson:"score" json:"score"`
	Subreddit  string             `bson:"subreddit" json:"subreddit"`
	URL        string             `bson:"url" json:"url"`
	Flair      string             `bson:"flair,omitempty" json:"flair,omitempty"`
	CreatedAt  time.Time          `bson:"created_at" json:"created_at"`
	InsertedAt time.Time          `bson:"inserted_at" json:"inserted_at"`
	UpdatedAt  time.Time          `bson:"updated_at" json:"updated_at"`
}

// IngestionPost represents the structure returned by the ingestion API
type IngestionPost struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	Author    string    `json:"author"`
	Score     int       `json:"score"`
	CreatedAt time.Time `json:"created_at"`
	Flair     string    `json:"flair,omitempty"`
	URL       string    `json:"url"`
}

// TaskExecutionResult represents the result of a task execution
type TaskExecutionResult struct {
	TaskName       string        `json:"task_name"`
	SubredditName  string        `json:"subreddit_name"`
	Success        bool          `json:"success"`
	PostsProcessed int           `json:"posts_processed"`
	Duration       time.Duration `json:"duration"`
	Error          string        `json:"error,omitempty"`
}