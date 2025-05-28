// internal/processor/interface.go
package processor

import (
	"reddit-orchestrator/internal/models"
)

type ProcessorInterface interface {
	ProcessSubredditPosts(ingestionPosts []models.IngestionPost, subreddit string) []models.Post
}
