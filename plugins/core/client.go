package core

import (
	"github.com/va6996/travelingman/tools"
	"github.com/firebase/genkit/go/genkit"
)

// Client manages the core set of tools
type Client struct {
	DateTool    *DateTool
	AskUserTool *AskUserTool
}

// NewClient initializes the core plugin and registers its tools
func NewClient(gk *genkit.Genkit, registry *tools.Registry) *Client {
	return &Client{
		DateTool:    NewDateTool(gk, registry),
		AskUserTool: NewAskUserTool(gk, registry),
	}
}
