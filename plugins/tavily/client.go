package tavily

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/firebase/genkit/go/genkit"
	"github.com/va6996/travelingman/log"
	"github.com/va6996/travelingman/tools"
)

const (
	BaseURL = "https://api.tavily.com"
)

// Client is the Tavily API client
type Client struct {
	apiKey     string
	httpClient *http.Client
}

// NewClient creates a new Tavily client
func NewClient(apiKey string, gk *genkit.Genkit, registry *tools.Registry, timeout int) *Client {
	if apiKey == "" {
		log.Warn(context.Background(), "Tavily API key is empty, Tavily tools will not work properly")
	}

	client := &Client{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: time.Duration(timeout) * time.Second,
		},
	}

	// Register tools
	client.registerTools(gk, registry)

	return client
}

// SearchRequest represents a Tavily search request
type SearchRequest struct {
	Query             string   `json:"query" description:"The search query to execute"`
	SearchDepth       string   `json:"search_depth,omitempty" description:"Search depth: basic, advanced, fast, or ultra-fast (default: basic)"`
	MaxResults        int      `json:"max_results,omitempty" description:"Maximum number of results (1-20, default: 5)"`
	Topic             string   `json:"topic,omitempty" description:"Search category: general, news, or finance (default: general)"`
	TimeRange         string   `json:"time_range,omitempty" description:"Time range: day, week, month, or year"`
	StartDate         string   `json:"start_date,omitempty" description:"Start date in YYYY-MM-DD format"`
	EndDate           string   `json:"end_date,omitempty" description:"End date in YYYY-MM-DD format"`
	IncludeAnswer     bool     `json:"include_answer,omitempty" description:"Include an LLM-generated answer"`
	IncludeRawContent bool     `json:"include_raw_content,omitempty" description:"Include raw content from search results"`
	IncludeImages     bool     `json:"include_images,omitempty" description:"Include image search results"`
	IncludeDomains    []string `json:"include_domains,omitempty" description:"Domains to specifically include"`
	ExcludeDomains    []string `json:"exclude_domains,omitempty" description:"Domains to specifically exclude"`
}

// SearchResult represents a single search result
type SearchResult struct {
	Title      string  `json:"title"`
	URL        string  `json:"url"`
	Content    string  `json:"content"`
	Score      float64 `json:"score"`
	RawContent *string `json:"raw_content,omitempty"`
	Favicon    *string `json:"favicon,omitempty"`
}

// SearchResponse represents the Tavily search response
type SearchResponse struct {
	Query        string          `json:"query"`
	Answer       string          `json:"answer,omitempty"`
	Images       []SearchImage   `json:"images,omitempty"`
	Results      []SearchResult  `json:"results"`
	ResponseTime string          `json:"response_time"`
	AutoParams   *AutoParameters `json:"auto_parameters,omitempty"`
	Usage        *Usage          `json:"usage,omitempty"`
	RequestID    string          `json:"request_id"`
}

type SearchImage struct {
	URL         string  `json:"url"`
	Description *string `json:"description,omitempty"`
}

type AutoParameters struct {
	Topic       string `json:"topic"`
	SearchDepth string `json:"search_depth"`
}

type Usage struct {
	Credits int `json:"credits"`
}

// Search performs a Tavily search
func (c *Client) Search(ctx context.Context, req *SearchRequest) (*SearchResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request is required")
	}
	if req.Query == "" {
		return nil, fmt.Errorf("query is required")
	}

	// Set defaults
	if req.SearchDepth == "" {
		req.SearchDepth = "basic"
	}
	if req.MaxResults == 0 {
		req.MaxResults = 5
	}
	if req.Topic == "" {
		req.Topic = "general"
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", BaseURL+"/search", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	log.Debugf(ctx, "[Tavily] Sending search request: query=%s, depth=%s, max_results=%d", req.Query, req.SearchDepth, req.MaxResults)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %s", resp.Status)
	}

	var searchResp SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	log.Debugf(ctx, "[Tavily] Search completed successfully: %d results found", len(searchResp.Results))

	return &searchResp, nil
}
