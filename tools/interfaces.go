package tools

import "context"

// Tool defines the interface for all AI tools
type Tool interface {
	// Name returns the unique name of the tool (e.g. "date_tool")
	Name() string

	// Description returns a description of what the tool does and its arguments
	Description() string

	// Execute runs the tool with the given arguments
	Execute(ctx context.Context, args map[string]interface{}) (interface{}, error)
}

// LLMClient defines the interface for LLM interaction
type LLMClient interface {
	GenerateContent(ctx context.Context, prompt string) (string, error)
}
