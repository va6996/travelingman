package tools

import (
	"fmt"
	"time"

	"github.com/dop251/goja"
)

type DateTool struct{}

func (t *DateTool) Name() string {
	return "date_tool"
}

func (t *DateTool) Description() string {
	return "Executes JavaScript code directly to calculate dates. The code must evaluate to a Date object."
}

func (t *DateTool) Execute(args map[string]interface{}) (interface{}, error) {
	code, ok := args["code"].(string)
	if !ok {
		return nil, fmt.Errorf("argument 'code' is required and must be a string")
	}

	vm := goja.New()

	// Execute the provided code
	// Wrap code in IIFE to support top-level return
	wrappedCode := fmt.Sprintf("(function() { %s })()", code)
	val, err := vm.RunString(wrappedCode)
	if err != nil {
		return nil, fmt.Errorf("JS execution failed: %v", err)
	}

	// Export the result to a Go value
	// If the result is a JS Date, Goja converts it to time.Time
	export := val.Export()

	if t, ok := export.(time.Time); ok {
		return t, nil
	}

	// If it's not a time.Time, maybe it's a string implementation of date?
	// But the requirement says "get the date + timezone output and convert that into a date object"
	// Let's assume the JS code evaluates to a Date object.

	return nil, fmt.Errorf("JS code did not return a valid Date object, got %T: %v", export, export)
}
