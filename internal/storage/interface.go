// internal/storage/interface.go
package storage

import (
	"context"

	"reddit-orchestrator/internal/models"
)

type StorageInterface interface {
	// Subreddit metadata operations
	GetSubredditMetadata(ctx context.Context, subredditName string) (*models.SubredditMetadata, error)
	UpsertSubredditMetadata(ctx context.Context, metadata *models.SubredditMetadata) error
	GetAllSubredditMetadata(ctx context.Context) ([]models.SubredditMetadata, error)

	// Post operations
	UpsertPost(ctx context.Context, post *models.Post) error
	UpsertPosts(ctx context.Context, posts []models.Post) error
	GetPostsBySubreddit(ctx context.Context, subreddit string, limit int) ([]models.Post, error)
	GetPostByRedditID(ctx context.Context, redditID string) (*models.Post, error)
	GetRecentPosts(ctx context.Context, subreddit string, hours int) ([]models.Post, error)
	GetPostsCount(ctx context.Context, subreddit string) (int64, error)

	GetAllSubredditConfigs(ctx context.Context) ([]models.SubredditConfig, error)
	GetActiveSubredditConfigs(ctx context.Context) ([]models.SubredditConfig, error)
	UpsertSubredditConfig(ctx context.Context, config *models.SubredditConfig) error
	GetSubredditConfig(ctx context.Context, subredditName string) (*models.SubredditConfig, error)
	DeleteSubredditConfig(ctx context.Context, subredditName string) error

	// Health check and cleanup
	Ping(ctx context.Context) error
	Close() error
}