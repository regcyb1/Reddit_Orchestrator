// internal/storage/mongo_storage.go
package storage

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"reddit-orchestrator/internal/models"
)

const (
	SubredditMetadataCollection = "subreddit_metadata" 
	SubredditPostsCollection   = "subreddit_post"
	SubredditConfigCollection  = "subreddit_config"
)

var _ StorageInterface = (*MongoStorage)(nil)

type MongoStorage struct {
	client   *mongo.Client
	database *mongo.Database
}

func NewMongoStorage(mongoURI, databaseName string) (*MongoStorage, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Test the connection
	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	database := client.Database(databaseName)

	storage := &MongoStorage{
		client:   client,
		database: database,
	}

	// Create indexes
	if err := storage.createIndexes(ctx); err != nil {
		return nil, fmt.Errorf("failed to create indexes: %w", err)
	}

	return storage, nil
}

func (s *MongoStorage) createIndexes(ctx context.Context) error {
	// Clean up any problematic indexes first
	postsCollection := s.database.Collection(SubredditPostsCollection)
	
	postsCollection.Indexes().DropOne(ctx, "reddit_name_1")
	postsCollection.Indexes().DropOne(ctx, "reddit_id_1") 

	// Subreddit metadata collection indexes
	metadataIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "subreddit_name", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{Keys: bson.D{{Key: "last_scraped_at", Value: -1}}},
	}
	if _, err := s.database.Collection(SubredditMetadataCollection).Indexes().CreateMany(ctx, metadataIndexes); err != nil {
		return err
	}

	// Subreddit posts collection indexes
	postsIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "reddit_id", Value: 1}},
			Options: options.Index().SetUnique(true).SetSparse(true), // Sparse to handle any nulls
		},
		{Keys: bson.D{{Key: "subreddit", Value: 1}}},
		{Keys: bson.D{{Key: "author", Value: 1}}},
		{Keys: bson.D{{Key: "created_at", Value: -1}}},
		{Keys: bson.D{{Key: "updated_at", Value: -1}}},
		{Keys: bson.D{{Key: "inserted_at", Value: -1}}},
		{Keys: bson.D{{Key: "subreddit", Value: 1}, {Key: "created_at", Value: -1}}},
	}
	if _, err := postsCollection.Indexes().CreateMany(ctx, postsIndexes); err != nil {
		return err
	}

	configIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "subreddit_name", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{Keys: bson.D{{Key: "enabled", Value: 1}}},
		{Keys: bson.D{{Key: "priority", Value: -1}}},
		{Keys: bson.D{{Key: "updated_at", Value: -1}}},
	}
	if _, err := s.database.Collection(SubredditConfigCollection).Indexes().CreateMany(ctx, configIndexes); err != nil {
		return err
	}

	return nil
}



// Subreddit metadata operations
func (s *MongoStorage) GetSubredditMetadata(ctx context.Context, subredditName string) (*models.SubredditMetadata, error) {
	collection := s.database.Collection(SubredditMetadataCollection)
	
	filter := bson.M{"subreddit_name": subredditName}

	var metadata models.SubredditMetadata
	err := collection.FindOne(ctx, filter).Decode(&metadata)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}

	return &metadata, nil
}

func (s *MongoStorage) UpsertSubredditMetadata(ctx context.Context, metadata *models.SubredditMetadata) error {
	collection := s.database.Collection(SubredditMetadataCollection)
	
	filter := bson.M{"subreddit_name": metadata.SubredditName}

	now := time.Now()
	update := bson.M{
		"$set": bson.M{
			"subreddit_name":   metadata.SubredditName,
			"last_scraped_at":  metadata.LastScrapedAt,
			"monitor_config":   metadata.MonitorConfig,
			"updated_at":       now,
		},
		"$setOnInsert": bson.M{
			"created_at": now,
		},
	}

	opts := options.Update().SetUpsert(true)
	_, err := collection.UpdateOne(ctx, filter, update, opts)
	return err
}

func (s *MongoStorage) GetAllSubredditMetadata(ctx context.Context) ([]models.SubredditMetadata, error) {
	collection := s.database.Collection(SubredditMetadataCollection)
	
	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var metadatas []models.SubredditMetadata
	if err := cursor.All(ctx, &metadatas); err != nil {
		return nil, err
	}

	return metadatas, nil
}

// Post operations
func (s *MongoStorage) UpsertPost(ctx context.Context, post *models.Post) error {
	// Validate post data before attempting to insert
	if post.RedditID == "" || post.Title == "" {
		return fmt.Errorf("invalid post data: reddit_id and title are required")
	}

	collection := s.database.Collection(SubredditPostsCollection)
	
	filter := bson.M{"reddit_id": post.RedditID}

	now := time.Now()
	post.UpdatedAt = now
	if post.InsertedAt.IsZero() {
		post.InsertedAt = now
	}

	update := bson.M{
		"$set": bson.M{
			"reddit_id":   post.RedditID,
			"title":       post.Title,
			"body":        post.Body,
			"author":      post.Author,
			"score":       post.Score,
			"subreddit":   post.Subreddit,
			"url":         post.URL,
			"flair":       post.Flair,
			"created_at":  post.CreatedAt,
			"updated_at":  post.UpdatedAt,
		},
		"$setOnInsert": bson.M{
			"inserted_at": post.InsertedAt,
		},
	}

	opts := options.Update().SetUpsert(true)
	_, err := collection.UpdateOne(ctx, filter, update, opts)
	return err
}

func (s *MongoStorage) UpsertPosts(ctx context.Context, posts []models.Post) error {
	if len(posts) == 0 {
		return nil
	}

	// Filter and validate posts before bulk operation
	validPosts := make([]models.Post, 0, len(posts))
	for _, post := range posts {
		// Only include posts with valid reddit_id and title
		if strings.TrimSpace(post.RedditID) != "" && strings.TrimSpace(post.Title) != "" {
			// Clean the data
			post.RedditID = strings.TrimSpace(post.RedditID)
			post.Title = strings.TrimSpace(post.Title)
			post.Body = strings.TrimSpace(post.Body)
			post.Author = strings.TrimSpace(post.Author)
			post.URL = strings.TrimSpace(post.URL)
			post.Flair = strings.TrimSpace(post.Flair)
			
			validPosts = append(validPosts, post)
		}
	}

	if len(validPosts) == 0 {
		return fmt.Errorf("no valid posts to insert")
	}

	// Use individual upserts to handle duplicates gracefully
	collection := s.database.Collection(SubredditPostsCollection)
	now := time.Now()
	
	successCount := 0
	errorCount := 0

	for _, post := range validPosts {
		post.UpdatedAt = now
		if post.InsertedAt.IsZero() {
			post.InsertedAt = now
		}

		filter := bson.M{"reddit_id": post.RedditID}
		update := bson.M{
			"$set": bson.M{
				"reddit_id":   post.RedditID,
				"title":       post.Title,
				"body":        post.Body,
				"author":      post.Author,
				"score":       post.Score,
				"subreddit":   post.Subreddit,
				"url":         post.URL,
				"flair":       post.Flair,
				"created_at":  post.CreatedAt,
				"updated_at":  post.UpdatedAt,
			},
			"$setOnInsert": bson.M{
				"inserted_at": post.InsertedAt,
			},
		}

		opts := options.Update().SetUpsert(true)
		_, err := collection.UpdateOne(ctx, filter, update, opts)
		if err != nil {
			fmt.Printf("Failed to upsert post %s: %v\n", post.RedditID, err)
			errorCount++
		} else {
			successCount++
		}
	}

	fmt.Printf("Bulk operation completed: %d successful, %d errors\n", successCount, errorCount)
	
	// Only return error if all operations failed
	if errorCount > 0 && successCount == 0 {
		return fmt.Errorf("all post insertions failed")
	}

	return nil
}

func (s *MongoStorage) GetPostsBySubreddit(ctx context.Context, subreddit string, limit int) ([]models.Post, error) {
	collection := s.database.Collection(SubredditPostsCollection)
	
	filter := bson.M{"subreddit": subreddit}
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	if limit > 0 {
		opts.SetLimit(int64(limit))
	}

	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var posts []models.Post
	if err := cursor.All(ctx, &posts); err != nil {
		return nil, err
	}

	return posts, nil
}

func (s *MongoStorage) GetPostByRedditID(ctx context.Context, redditID string) (*models.Post, error) {
	collection := s.database.Collection(SubredditPostsCollection)
	
	filter := bson.M{"reddit_id": redditID}

	var post models.Post
	err := collection.FindOne(ctx, filter).Decode(&post)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}

	return &post, nil
}

func (s *MongoStorage) GetRecentPosts(ctx context.Context, subreddit string, hours int) ([]models.Post, error) {
	collection := s.database.Collection(SubredditPostsCollection)
	
	cutoff := time.Now().Add(-time.Duration(hours) * time.Hour)
	filter := bson.M{
		"subreddit": subreddit,
		"$or": []bson.M{
			{"created_at": bson.M{"$gte": cutoff}},
			{"updated_at": bson.M{"$gte": cutoff}},
		},
	}

	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var posts []models.Post
	if err := cursor.All(ctx, &posts); err != nil {
		return nil, err
	}

	return posts, nil
}

func (s *MongoStorage) GetPostsCount(ctx context.Context, subreddit string) (int64, error) {
	collection := s.database.Collection(SubredditPostsCollection)
	
	filter := bson.M{}
	if subreddit != "" {
		filter["subreddit"] = subreddit
	}

	count, err := collection.CountDocuments(ctx, filter)
	if err != nil {
		return 0, err
	}

	return count, nil
}

// Subreddit config operations
func (s *MongoStorage) GetAllSubredditConfigs(ctx context.Context) ([]models.SubredditConfig, error) {
	collection := s.database.Collection(SubredditConfigCollection)
	
	opts := options.Find().SetSort(bson.D{{Key: "priority", Value: -1}, {Key: "subreddit_name", Value: 1}})
	cursor, err := collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var configs []models.SubredditConfig
	if err := cursor.All(ctx, &configs); err != nil {
		return nil, err
	}

	return configs, nil
}

func (s *MongoStorage) GetActiveSubredditConfigs(ctx context.Context) ([]models.SubredditConfig, error) {
	collection := s.database.Collection(SubredditConfigCollection)
	
	filter := bson.M{"enabled": true}
	opts := options.Find().SetSort(bson.D{{Key: "priority", Value: -1}, {Key: "subreddit_name", Value: 1}})
	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var configs []models.SubredditConfig
	if err := cursor.All(ctx, &configs); err != nil {
		return nil, err
	}

	return configs, nil
}

func (s *MongoStorage) UpsertSubredditConfig(ctx context.Context, config *models.SubredditConfig) error {
	collection := s.database.Collection(SubredditConfigCollection)
	
	filter := bson.M{"subreddit_name": config.SubredditName}

	now := time.Now()
	config.UpdatedAt = now
	if config.CreatedAt.IsZero() {
		config.CreatedAt = now
	}

	update := bson.M{
		"$set": bson.M{
			"subreddit_name": config.SubredditName,
			"enabled":        config.Enabled,
			"schedule":       config.Schedule,
			"max_posts":      config.MaxPosts,
			"priority":       config.Priority,
			"description":    config.Description,
			"updated_at":     config.UpdatedAt,
		},
		"$setOnInsert": bson.M{
			"created_at": config.CreatedAt,
		},
	}

	opts := options.Update().SetUpsert(true)
	_, err := collection.UpdateOne(ctx, filter, update, opts)
	return err
}

func (s *MongoStorage) GetSubredditConfig(ctx context.Context, subredditName string) (*models.SubredditConfig, error) {
	collection := s.database.Collection(SubredditConfigCollection)
	
	filter := bson.M{"subreddit_name": subredditName}

	var config models.SubredditConfig
	err := collection.FindOne(ctx, filter).Decode(&config)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}

	return &config, nil
}

func (s *MongoStorage) DeleteSubredditConfig(ctx context.Context, subredditName string) error {
	collection := s.database.Collection(SubredditConfigCollection)
	
	filter := bson.M{"subreddit_name": subredditName}
	_, err := collection.DeleteOne(ctx, filter)
	return err
}
// Health check and cleanup
func (s *MongoStorage) Ping(ctx context.Context) error {
	return s.client.Ping(ctx, nil)
}

func (s *MongoStorage) Close() error {
	return s.client.Disconnect(context.Background())
}