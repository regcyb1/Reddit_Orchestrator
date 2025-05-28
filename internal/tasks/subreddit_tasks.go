// internal/tasks/subreddit_tasks.go
package tasks

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/ersauravadhikari/blueberry-go/blueberry"

	"reddit-orchestrator/internal/client"
	"reddit-orchestrator/internal/config"
	"reddit-orchestrator/internal/models"
	"reddit-orchestrator/internal/processor"
	"reddit-orchestrator/internal/storage"
)

// Ensure SubredditTaskManager implements TaskManagerInterface
var _ TaskManagerInterface = (*SubredditTaskManager)(nil)

type SubredditTaskManager struct {
	blueBerry *blueberry.BlueBerry
	storage   storage.StorageInterface
	client    client.IngestionClientInterface
	processor processor.ProcessorInterface
	config    *config.Config
}

func NewSubredditTaskManager(
	bb *blueberry.BlueBerry,
	storage storage.StorageInterface,
	client client.IngestionClientInterface,
	processor processor.ProcessorInterface,
	config *config.Config,
) *SubredditTaskManager {
	return &SubredditTaskManager{
		blueBerry: bb,
		storage:   storage,
		client:    client,
		processor: processor,
		config:    config,
	}
}

// RegisterTasks registers all subreddit monitoring tasks with BlueBerry
func (tm *SubredditTaskManager) RegisterTasks() error {
	// Define task schema
	subredditSchema := blueberry.NewTaskSchema(blueberry.TaskParamDefinition{
		"subreddit":       blueberry.TypeString,
		"limit":           blueberry.TypeString,
		"since_timestamp": blueberry.TypeString,
	})

	// Register the subreddit monitoring task
	task, err := tm.blueBerry.RegisterTask(
		"monitor_subreddit",
		tm.monitorSubreddit,
		subredditSchema,
	)
	if err != nil {
		return fmt.Errorf("failed to register subreddit monitoring task: %w", err)
	}

	// Get active subreddit configurations from database
	ctx := context.Background()
	configs, err := tm.storage.GetActiveSubredditConfigs(ctx)
	if err != nil {
		return fmt.Errorf("failed to get subreddit configs: %w", err)
	}

	if len(configs) == 0 {
		fmt.Println("No active subreddit configurations found. Please add some to the database.")
		return nil
	}

	// Schedule each active subreddit
	for _, config := range configs {
		schedule := config.Schedule
		if schedule == "" {
			schedule = tm.config.SubredditSchedule // Default from config
		}

		_, err := task.RegisterSchedule(blueberry.TaskParams{
			"subreddit":       config.SubredditName,
			"limit":           fmt.Sprintf("%d", config.MaxPosts),
			"since_timestamp": "", // Use automatic timestamp
		}, schedule)
		
		if err != nil {
			fmt.Printf("Failed to schedule subreddit %s: %v\n", config.SubredditName, err)
			continue
		}

		fmt.Printf("Scheduled r/%s (priority: %d, max_posts: %d, schedule: %s)\n", 
			config.SubredditName, config.Priority, config.MaxPosts, schedule)
	}

	fmt.Printf("Successfully scheduled %d subreddits\n", len(configs))
	return nil
}

// monitorSubreddit is the main task function executed by BlueBerry
func (tm *SubredditTaskManager) monitorSubreddit(tctx *blueberry.TaskContext) error {
	ctx := tctx.GetContext()
	logger := tctx.GetLogger()
	params := tctx.GetParams()

	// Extract and validate required parameters
	subredditName, ok := params["subreddit"].(string)
	if !ok || subredditName == "" {
		return logger.Error("invalid or missing subreddit parameter")
	}

	limit := tm.config.DefaultLimit
	if l, exists := params["limit"]; exists {
		if limitStr, ok := l.(string); ok && limitStr != "" {
			if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
				limit = parsed
			} else {
				logger.Info(fmt.Sprintf("Invalid limit value '%s', using default %d", limitStr, tm.config.DefaultLimit))
			}
		}
	}

	var sinceTimestamp int64
	var hasManualTimestamp bool
	if ts, exists := params["since_timestamp"]; exists {
		if tsStr, ok := ts.(string); ok && tsStr != "" {
			if parsed, err := strconv.ParseInt(tsStr, 10, 64); err == nil && parsed > 0 {
				sinceTimestamp = parsed
				hasManualTimestamp = true
				logger.Info(fmt.Sprintf("Using manual since_timestamp: %d", sinceTimestamp))
			} else {
				logger.Info(fmt.Sprintf("Invalid timestamp value '%s', using last scraped time", tsStr))
			}
		}
	}

	logger.Info(fmt.Sprintf("Starting subreddit monitoring for: r/%s (limit: %d)", subredditName, limit))

	// Get last scraped timestamp if no manual override
	if !hasManualTimestamp {
		metadata, err := tm.storage.GetSubredditMetadata(ctx, subredditName)
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to get metadata: %v", err))
			return err
		}

		if metadata != nil && !metadata.LastScrapedAt.IsZero() {
			sinceTimestamp = metadata.LastScrapedAt.Unix()
			logger.Info(fmt.Sprintf("Using since_timestamp: %d", sinceTimestamp))
		} else {
			logger.Info("No previous scrape data found")
		}
	}

	// Record the time we're starting this scrape
	scrapeStartTime := time.Now()

	// Fetch posts from ingestion API
	ingestionPosts, err := tm.client.GetSubredditPosts(ctx, subredditName, limit, sinceTimestamp)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to fetch subreddit posts: %v", err))
		return err
	}

	if len(ingestionPosts) == 0 {
		logger.Info("No new posts found")
		return tm.updateMetadata(ctx, subredditName, limit, scrapeStartTime, logger)
	}

	logger.Info(fmt.Sprintf("Fetched %d posts from ingestion API", len(ingestionPosts)))

	// Process posts (clean and convert)
	processedPosts := tm.processor.ProcessSubredditPosts(ingestionPosts, subredditName)
	logger.Info(fmt.Sprintf("Processed %d valid posts", len(processedPosts)))

	// Store posts in MongoDB
	if err := tm.storage.UpsertPosts(ctx, processedPosts); err != nil {
		logger.Error(fmt.Sprintf("Failed to store posts: %v", err))
		return err
	}

	// Update metadata with scrape start time
	if err := tm.updateMetadata(ctx, subredditName, limit, scrapeStartTime, logger); err != nil {
		return err
	}

	duration := time.Since(scrapeStartTime)
	logger.Success(fmt.Sprintf("Successfully processed r/%s: %d posts stored in %v", 
		subredditName, len(processedPosts), duration.Round(time.Millisecond)))

	return nil
}

// updateMetadata updates the subreddit monitoring metadata
func (tm *SubredditTaskManager) updateMetadata(ctx context.Context, subredditName string, limit int, scrapedAt time.Time, logger *blueberry.Logger) error {
	metadata := &models.SubredditMetadata{
		SubredditName: subredditName,
		LastScrapedAt: scrapedAt,
		MonitorConfig: models.MonitorConfig{
			Enabled:  true,
			MaxPosts: limit,
		},
	}

	if err := tm.storage.UpsertSubredditMetadata(ctx, metadata); err != nil {
		logger.Error(fmt.Sprintf("Failed to update metadata: %v", err))
		return err
	}

	logger.Info(fmt.Sprintf("Updated last_scraped_at timestamp: %d", scrapedAt.Unix()))
	return nil
}