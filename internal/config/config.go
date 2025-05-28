// internal/config/config.go
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	MongoDBURI   string
	DatabaseName string

	IngestionAPIURL string
	RequestTimeout  time.Duration

	ServerPort string

	// Authentication configuration (required)
	WebAuthUser     string
	WebAuthPassword string

	// Task configuration
	DefaultSubreddits        []string
	SubredditSchedule        string
	DefaultLimit             int
	DefaultLookbackHours     int
	MaxRetries               int
}

func LoadConfig() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		MongoDBURI:           getEnv("MONGODB_URI", "mongodb://localhost:27017"),
		DatabaseName:         getEnv("DATABASE_NAME", "reddit_data"),
		IngestionAPIURL:      getEnv("INGESTION_API_URL", "http://localhost:8080"),
		RequestTimeout:       getEnvDuration("REQUEST_TIMEOUT", 60*time.Second),
		ServerPort:           getEnv("SERVER_PORT", "8080"),
		WebAuthUser:          getEnv("WEB_AUTH_USER", "admin"),
		WebAuthPassword:      getEnv("WEB_AUTH_PASSWORD", "password"),
		SubredditSchedule:    getEnv("SUBREDDIT_SCHEDULE", "@every 1h"),
		DefaultLimit:         getEnvInt("DEFAULT_LIMIT", 100),
		DefaultLookbackHours: getEnvInt("DEFAULT_LOOKBACK_HOURS", 1),
		MaxRetries:           getEnvInt("MAX_RETRIES", 3),
		DefaultSubreddits:    getEnvStringSlice("DEFAULT_SUBREDDITS", []string{"golang", "programming"}),
	}

	if cfg.MongoDBURI == "" {
		return nil, fmt.Errorf("MONGODB_URI is required")
	}
	if cfg.IngestionAPIURL == "" {
		return nil, fmt.Errorf("INGESTION_API_URL is required")
	}
	if cfg.WebAuthUser == "" || cfg.WebAuthPassword == "" {
		return nil, fmt.Errorf("WEB_AUTH_USER and WEB_AUTH_PASSWORD are required")
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

func getEnvStringSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		
		return []string{value} 
	}
	return defaultValue
}