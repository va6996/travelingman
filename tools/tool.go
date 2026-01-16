package tools

// Tool defines the interface for all AI tools
type Tool interface {
	// Name returns the unique name of the tool (e.g. "date_tool")
	Name() string

	// Description returns a description of what the tool does and its arguments
	Description() string

	// Execute runs the tool with the given arguments
	Execute(args map[string]interface{}) (interface{}, error)
}
