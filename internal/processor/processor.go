// internal/processor/processor.go
package processor

import (
	"strings"
	"time"

	"reddit-orchestrator/internal/models"
)

// Ensure Processor implements ProcessorInterface
var _ ProcessorInterface = (*Processor)(nil)

type Processor struct{}

func NewProcessor() *Processor {
	return &Processor{}
}

// ProcessSubredditPosts cleans and validates posts from the ingestion API
func (p *Processor) ProcessSubredditPosts(ingestionPosts []models.IngestionPost, subreddit string) []models.Post {
	processed := make([]models.Post, 0, len(ingestionPosts))
	
	for _, ingestionPost := range ingestionPosts {
		redditID := strings.TrimSpace(ingestionPost.ID)
		title := strings.TrimSpace(ingestionPost.Title)
		
		if redditID == "" || title == "" {
			continue
		}

		if len(redditID) < 3 || strings.Contains(redditID, " ") {
			continue
		}

		processedPost := models.Post{
			RedditID:   redditID,
			Title:      title,
			Body:       strings.TrimSpace(ingestionPost.Body),
			Author:     strings.TrimSpace(ingestionPost.Author),
			Score:      ingestionPost.Score,
			Subreddit:  subreddit, // Use the subreddit we're monitoring
			URL:        strings.TrimSpace(ingestionPost.URL),
			Flair:      strings.TrimSpace(ingestionPost.Flair),
			CreatedAt:  ingestionPost.CreatedAt,
			InsertedAt: time.Now(),
			UpdatedAt:  time.Now(),
		}

		if processedPost.RedditID == "" || processedPost.Title == "" {
			continue
		}

		processed = append(processed, processedPost)
	}

	return processed
}