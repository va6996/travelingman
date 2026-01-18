package tools_test

import (
	"context"
	"testing"

	"github.com/va6996/travelingman/plugins/core"
	"github.com/va6996/travelingman/tools"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/stretchr/testify/assert"
)

func TestNewRegistry(t *testing.T) {
	reg := tools.NewRegistry()
	assert.NotNil(t, reg)
	assert.Empty(t, reg.GetTools())
}

func TestRegistry_Register(t *testing.T) {
	ctx := context.Background()
	gk := genkit.Init(ctx)
	reg := tools.NewRegistry()

	// Register a dummy tool
	reg.Register(genkit.DefineTool[*core.DateInput, string](
		gk,
		"testTool",
		"Test Description",
		func(ctx *ai.ToolContext, input *core.DateInput) (string, error) {
			return "ok", nil
		},
	), func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		return "ok", nil
	})

	tools := reg.GetTools()
	assert.Len(t, tools, 1)
	assert.Equal(t, "testTool", tools[0].Definition().Name)
}
