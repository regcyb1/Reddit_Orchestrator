// internal/client/ingestion_client.go
package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"reddit-orchestrator/internal/models"
)

type IngestionClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewIngestionClient(baseURL string, timeout time.Duration) *IngestionClient {
	return &IngestionClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// GetSubredditPosts calls the ingestion API to fetch subreddit posts
func (c *IngestionClient) GetSubredditPosts(ctx context.Context, subreddit string, limit int, sinceTimestamp int64) ([]models.IngestionPost, error) {
	params := url.Values{}
	params.Set("subreddit", subreddit)
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	if sinceTimestamp > 0 {
		params.Set("since_timestamp", strconv.FormatInt(sinceTimestamp, 10))
	}

	endpoint := fmt.Sprintf("%s/subreddit?%s", c.baseURL, params.Encode())
	
	var response struct {
		Posts []models.IngestionPost `json:"posts"`
		Meta  map[string]interface{} `json:"meta"`
	}
	
	if err := c.makeRequest(ctx, endpoint, &response); err != nil {
		return nil, err
	}

	return response.Posts, nil
}

// Health check method
func (c *IngestionClient) HealthCheck(ctx context.Context) error {
	endpoint := fmt.Sprintf("%s/health", c.baseURL)
	
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return fmt.Errorf("creating health check request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("making health check request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ingestion API health check failed with status: %d", resp.StatusCode)
	}

	return nil
}

func (c *IngestionClient) makeRequest(ctx context.Context, endpoint string, result interface{}) error {
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}

	if err := json.Unmarshal(body, result); err != nil {
		return fmt.Errorf("parsing response: %w", err)
	}

	return nil
}