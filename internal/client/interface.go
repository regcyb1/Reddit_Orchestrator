// internal/client/interface.go
package client

import (
	"context"

	"reddit-orchestrator/internal/models"
)

type IngestionClientInterface interface {
	GetSubredditPosts(ctx context.Context, subreddit string, limit int, sinceTimestamp int64) ([]models.IngestionPost, error)
	HealthCheck(ctx context.Context) error
}


