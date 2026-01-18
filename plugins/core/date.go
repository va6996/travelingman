package core

import (
	"context"
	"fmt"
	"time"

	"github.com/va6996/travelingman/tools"
	"github.com/dop251/goja"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

// DateInput defines the input for the date tool
type DateInput struct {
	Code string `json:"code"`
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

	registry.Register(genkit.DefineTool[*DateInput, string](
		gk,
		"dateTool",
		t.Description(),
		func(ctx *ai.ToolContext, input *DateInput) (string, error) {
			res, err := t.Execute(ctx, map[string]interface{}{"code": input.Code})
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("%v", res), nil
		},
	), t.Execute)

	return t
}

func (t *DateTool) Name() string {
	return "date_tool"
}

func (t *DateTool) Description() string {
	// Updated description to match the new behavior (JS execution)
	// Or should I revert to simple date? The previous implementation seemed to execute JS.
	// Let's stick to what was there: "Executes JavaScript code..."
	return "Executes JavaScript code directly to calculate dates. The code must evaluate to a Date object. Variable 'now' is available holding the current date."
}

func (t *DateTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	code, ok := args["code"].(string)
	if !ok {
		return nil, fmt.Errorf("argument 'code' is required and must be a string")
	}

	vm := goja.New()

	// Inject 'now' variable
	vm.Set("now", t.Now())

	// Wrap code in IIFE to support top-level return
	wrappedCode := fmt.Sprintf("(function() { return %s })()", code)
	val, err := vm.RunString(wrappedCode)
	if err != nil {
		return nil, fmt.Errorf("js execution failed: %w", err)
	}

	// Check if result is a Date object and get its ISO string
	exported := val.Export()
	if dateObj, ok := exported.(*time.Time); ok {
		return dateObj.UTC().Format(time.RFC3339), nil
	}

	// Try to call toISOString() if it's a Date object
	isoResult, err := vm.RunString(fmt.Sprintf("(() => { try { return (function() { return %s })().toISOString(); } catch(e) { return null; } })()", code))
	if err == nil && isoResult != nil {
		if isoString, ok := isoResult.Export().(string); ok && isoString != "" {
			return isoString, nil
		}
	}

	// For other types, return string representation
	return val.String(), nil
}
