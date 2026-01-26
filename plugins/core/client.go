package core

import (
	"github.com/firebase/genkit/go/genkit"
	"github.com/va6996/travelingman/tools"
)

// Client manages the core set of tools
type Client struct {
	DateTool     *DateTool
	AskUserTool  *AskUserTool
	CurrencyTool *CurrencyTool
}

// NewClient initializes the core plugin and registers its tools
func NewClient(gk *genkit.Genkit, registry *tools.Registry) *Client {
	return &Client{
		DateTool:     NewDateTool(gk, registry),
		AskUserTool:  NewAskUserTool(gk, registry),
		CurrencyTool: NewCurrencyTool(gk, registry),
	}
}
