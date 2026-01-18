package tools

import (
	"context"
	"fmt"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

// ToolPlugin defines the interface for plugins that provide tools
type ToolPlugin interface {
	RegisterTools(gk *genkit.Genkit, registry *Registry)
}

// ToolExecutor is the function signature for executing a tool
type ToolExecutor func(ctx context.Context, args map[string]interface{}) (interface{}, error)

// Registry manages the registration of AI tools
type Registry struct {
	tools     []ai.Tool
	executors map[string]ToolExecutor
}

// NewRegistry creates a new tool registry
func NewRegistry() *Registry {
	return &Registry{
		tools:     make([]ai.Tool, 0),
		executors: make(map[string]ToolExecutor),
	}
}

// Register adds a tool to the registry with its executor
func (r *Registry) Register(tool ai.Tool, executor ToolExecutor) {
	r.tools = append(r.tools, tool)
	r.executors[tool.Definition().Name] = executor
}

// GetTools returns all registered tools
func (r *Registry) GetTools() []ai.Tool {
	return r.tools
}

// ExecuteTool runs a registered tool by name
func (r *Registry) ExecuteTool(ctx context.Context, name string, args map[string]interface{}) (interface{}, error) {
	executor, ok := r.executors[name]
	if !ok {
		return nil, fmt.Errorf("tool not found: %s", name)
	}
	return executor(ctx, args)
}
