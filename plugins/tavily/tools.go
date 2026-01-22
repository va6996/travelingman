package tavily

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/va6996/travelingman/log"
	"github.com/va6996/travelingman/tools"
)

// Tool definitions

// SearchTool implements the Tavily search tool
type SearchTool struct {
	client *Client
}

func (t *SearchTool) Name() string {
	return "tavily_search"
}

func (t *SearchTool) Description() string {
	return "Searches the web for current information using Tavily. Useful for finding recent news, facts, or real-time data. Arguments: query (string, required), search_depth (string: basic/advanced/fast/ultra-fast, optional), max_results (int: 1-20, optional), topic (string: general/news/finance, optional), time_range (string: day/week/month/year, optional)."
}

func (t *SearchTool) Execute(ctx context.Context, input *SearchRequest) (*SearchResponse, error) {
	inputJSON, _ := json.Marshal(input)
	log.Debugf(ctx, "[Tavily] SearchTool executing with input: %s", string(inputJSON))

	if t.client == nil {
		return nil, fmt.Errorf("tavily client not initialized")
	}

	if input.Query == "" {
		return nil, fmt.Errorf("query is required")
	}

	resp, err := t.client.Search(ctx, input)
	if err != nil {
		log.Errorf(ctx, "[Tavily] SearchTool failed: %v", err)
		return nil, err
	}

	log.Debugf(ctx, "[Tavily] SearchTool completed successfully. Found %d results", len(resp.Results))

	return resp, nil
}

// ExtractTool implements the Tavily content extraction tool
type ExtractTool struct {
	client *Client
}

type ExtractInput struct {
	URLs []string `json:"urls" description:"List of URLs to extract content from"`
}

type ExtractResponse struct {
	Results []ExtractResult `json:"results"`
}

type ExtractResult struct {
	URL     string `json:"url"`
	Title   string `json:"title"`
	Content string `json:"content"`
}

func (t *ExtractTool) Name() string {
	return "tavily_extract"
}

func (t *ExtractTool) Description() string {
	return "Extracts clean content from web pages using Tavily. Removes ads, navigation, and other clutter. Useful when you have specific URLs you want to get content from. Arguments: urls (list of strings, required)."
}

func (t *ExtractTool) Execute(ctx context.Context, input *ExtractInput) (*ExtractResponse, error) {
	inputJSON, _ := json.Marshal(input)
	log.Debugf(ctx, "[Tavily] ExtractTool executing with input: %s", string(inputJSON))

	if t.client == nil {
		return nil, fmt.Errorf("tavily client not initialized")
	}

	if len(input.URLs) == 0 {
		return nil, fmt.Errorf("at least one URL is required")
	}

	// For now, we'll implement a simple version that calls the search API
	// The actual extract endpoint would need to be implemented separately
	log.Warnf(ctx, "[Tavily] Extract functionality not fully implemented, using search as fallback")

	// Return empty results for now
	return &ExtractResponse{
		Results: []ExtractResult{},
	}, nil
}

// registerTools registers all Tavily tools with the registry
func (c *Client) registerTools(gk *genkit.Genkit, registry *tools.Registry) {
	if gk == nil || registry == nil {
		log.Warn(context.Background(), "[Tavily] Cannot register tools: genkit or registry is nil")
		return
	}

	// Register SearchTool
	searchTool := &SearchTool{client: c}
	registry.Register(genkit.DefineTool(gk, searchTool.Name(), searchTool.Description(),
		func(ctx *ai.ToolContext, input *SearchRequest) (*SearchResponse, error) {
			return searchTool.Execute(ctx, input)
		},
	), func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		// Adapter for generic registry execution
		query, ok := args["query"].(string)
		if !ok {
			return nil, fmt.Errorf("query is required and must be a string")
		}

		req := &SearchRequest{
			Query: query,
		}

		// Optional parameters
		if depth, ok := args["search_depth"].(string); ok {
			req.SearchDepth = depth
		}
		if maxResults, ok := args["max_results"].(float64); ok {
			req.MaxResults = int(maxResults)
		}
		if topic, ok := args["topic"].(string); ok {
			req.Topic = topic
		}
		if timeRange, ok := args["time_range"].(string); ok {
			req.TimeRange = timeRange
		}
		if startDate, ok := args["start_date"].(string); ok {
			req.StartDate = startDate
		}
		if endDate, ok := args["end_date"].(string); ok {
			req.EndDate = endDate
		}
		if includeAnswer, ok := args["include_answer"].(bool); ok {
			req.IncludeAnswer = includeAnswer
		}
		if includeRawContent, ok := args["include_raw_content"].(bool); ok {
			req.IncludeRawContent = includeRawContent
		}

		return searchTool.Execute(ctx, req)
	})

	log.Info(context.Background(), "[Tavily] Registered tool: tavily_search")

	// Register ExtractTool (commented out until fully implemented)
	// extractTool := &ExtractTool{client: c}
	// registry.Register(genkit.DefineTool(gk, extractTool.Name(), extractTool.Description(),
	// 	func(ctx *ai.ToolContext, input *ExtractInput) (*ExtractResponse, error) {
	// 		return extractTool.Execute(ctx, input)
	// 	},
	// ), func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	// 	urls, ok := args["urls"].([]string)
	// 	if !ok {
	// 		return nil, fmt.Errorf("urls is required and must be a list of strings")
	// 	}
	// 	return extractTool.Execute(ctx, &ExtractInput{URLs: urls})
	// })

	// log.Info(context.Background(), "[Tavily] Registered tool: tavily_extract")
}
