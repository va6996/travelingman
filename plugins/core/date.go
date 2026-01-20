package core

import (
	"context"
	"fmt"
	"time"

	"github.com/dop251/goja"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/va6996/travelingman/tools"
)

// DateInput defines the input for the date tool
type DateInput struct {
	Expression string `json:"expression" description:"JavaScript expression to calculate a date. Variable 'now' is available as current timestamp in milliseconds."`
}

// DateTool provides current date functionality
type DateTool struct {
	Now func() time.Time
}

// NewDateTool creates a new DateTool and registers it
func NewDateTool(gk *genkit.Genkit, registry *tools.Registry) *DateTool {
	t := &DateTool{
		Now: time.Now,
	}

	if gk == nil || registry == nil {
		return t
	}

	registry.Register(genkit.DefineTool[*DateInput, *time.Time](
		gk,
		"dateTool",
		t.Description(),
		func(ctx *ai.ToolContext, input *DateInput) (*time.Time, error) {
			return t.Execute(ctx, input)
		},
	), func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		// Adapter for generic registry execution
		expression, ok := args["expression"].(string)
		if !ok {
			return nil, fmt.Errorf("missing expression")
		}
		return t.Execute(ctx, &DateInput{Expression: expression})
	})

	return t
}

func (t *DateTool) Name() string {
	return "dateTool"
}

func (t *DateTool) Description() string {
	return `Executes JavaScript expression to calculate dates. Variable 'now' is available holding the current timestamp (milliseconds).
Return a Date object or ISO string. The last expression is the return value.
Examples:
- Next Friday: "var d = new Date(now); d.setDate(d.getDate() + (12 - d.getDay()) % 7); if(d.getDay() !== 5 || d <= now) d.setDate(d.getDate() + 7); d"
- Tomorrow: "new Date(now + 86400000)"`
}

func (t *DateTool) Execute(ctx context.Context, input *DateInput) (*time.Time, error) {
	if input == nil {
		return nil, fmt.Errorf("input is required")
	}
	expression := input.Expression
	// log usage for debug
	fmt.Printf("[DateTool] Executing expression: %s\n", expression)

	vm := goja.New()
	err := vm.Set("now", t.Now().UnixMilli())
	if err != nil {
		return nil, fmt.Errorf("failed to set 'now': %w", err)
	}

	val, err := vm.RunString(expression)
	if err != nil {
		fmt.Printf("[DateTool] RunString error: %v\n", err)
		return nil, fmt.Errorf("js execution failed: %w", err)
	}
	fmt.Printf("[DateTool] RunString result: %v (IsUndefined: %v, IsNull: %v)\n", val, val == goja.Undefined(), val == goja.Null())

	exported := val.Export()
	fmt.Printf("[DateTool] Exported result: %v (Type: %T)\n", exported, exported)

	// If explicitly nil/undefined
	if exported == nil {
		return nil, fmt.Errorf("result is null or undefined")
	}

	// Check if it's a time.Time (Goja converts JS Date to time.Time)
	if dateObj, ok := exported.(time.Time); ok {
		return &dateObj, nil
	}

	// If it's a string, try to parse it
	if str, ok := exported.(string); ok {
		if t, err := time.Parse(time.RFC3339, str); err == nil {
			return &t, nil
		}
		// Try other formats if needed, or just fail
	}

	return nil, fmt.Errorf("result is not a valid Date object or ISO string")
}
