package core

import (
	"context"
	"fmt"
	"time"

	"github.com/dop251/goja"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/sirupsen/logrus"
	"github.com/va6996/travelingman/log"
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

	registry.Register(genkit.DefineTool[*DateInput, []time.Time](
		gk,
		"dateTool",
		t.Description(),
		func(ctx *ai.ToolContext, input *DateInput) ([]time.Time, error) {
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
Return an array of JavaScript Date objects.
Examples:
- Single date (Next Friday): "var d = new Date(now); d.setDate(d.getDate() + (12 - d.getDay()) % 7); if(d.getDay() !== 5 || d <= now) d.setDate(d.getDate() + 7); [d]"
- Tomorrow: "[new Date(now + 86400000)]"
- Multiple dates (Next 3 days): "var d1=new Date(now+86400000); var d2=new Date(now+172800000); var d3=new Date(now+259200000); [d1, d2, d3]"`
}

func (t *DateTool) Execute(ctx context.Context, input *DateInput) ([]time.Time, error) {
	if input == nil {
		return nil, fmt.Errorf("input is required")
	}
	expression := input.Expression
	log.Infof(ctx, "[DateTool] Executing expression: %s", expression)

	vm := goja.New()
	err := vm.Set("now", t.Now().UnixMilli())
	if err != nil {
		return nil, fmt.Errorf("failed to set 'now': %w", err)
	}

	val, err := vm.RunString(expression)
	if err != nil {
		log.Errorf(ctx, "[DateTool] RunString error: %v", err)
		return nil, fmt.Errorf("js execution failed: %w", err)
	}
	// logrus.Debugf("[DateTool] RunString result: %v (IsUndefined: %v, IsNull: %v)", val, val == goja.Undefined(), val == goja.Null())

	exported := val.Export()
	log.Infof(ctx, "[DateTool] Exported result: %v (Type: %T)", exported, exported)

	if exported == nil {
		return nil, fmt.Errorf("result is null or undefined")
	}

	// Check if it's a single time.Time
	if dateObj, ok := exported.(time.Time); ok {
		return []time.Time{dateObj}, nil
	}

	// If it's a string, try to parse it as a single date
	if str, ok := exported.(string); ok {
		// Just validate it parses
		if t, err := time.Parse(time.RFC3339, str); err == nil {
			return []time.Time{t}, nil
		}
	}

	// Check if it's an array/slice
	if arr, ok := exported.([]interface{}); ok {
		return t.processArray(arr)
	}

	// Try to check if the value itself is a goja array/object
	if gojaVal, ok := exported.(goja.Value); ok {
		return t.processGojaValue(gojaVal)
	}

	return nil, fmt.Errorf("result is not a valid Date, ISO string, or array. Got Type: %T, Value: %v", exported, exported)
}

func (t *DateTool) processArray(arr []interface{}) ([]time.Time, error) {
	var dates []time.Time
	for i, item := range arr {
		logrus.Debugf("[DateTool] Processing array element %d: %v (Type: %T)", i, item, item)

		// Handle nested arrays by flattening them
		if nestedArr, ok := item.([]interface{}); ok {
			nestedDates, err := t.processArray(nestedArr)
			if err != nil {
				return nil, fmt.Errorf("nested array error at index %d: %w", i, err)
			}
			dates = append(dates, nestedDates...)
			continue
		}

		date, err := t.extractDate(item)
		if err != nil {
			return nil, fmt.Errorf("element %d error: %w", i, err)
		}
		dates = append(dates, date)
	}
	if len(dates) == 0 {
		return nil, fmt.Errorf("array is empty or contains no valid dates")
	}
	return dates, nil
}

func (t *DateTool) processGojaValue(gojaVal goja.Value) ([]time.Time, error) {
	if gojaVal.SameAs(goja.Null()) || gojaVal.SameAs(goja.Undefined()) {
		return nil, fmt.Errorf("result is null or undefined")
	}

	exported := gojaVal.Export()
	if arr, ok := exported.([]interface{}); ok {
		return t.processArray(arr)
	}

	return nil, fmt.Errorf("goja value is not a valid array")
}

func (t *DateTool) extractDate(item interface{}) (time.Time, error) {
	// Handle time.Time directly
	if dateObj, ok := item.(time.Time); ok {
		return dateObj, nil
	}

	// Handle nested array - this indicates the LLM returned a nested array structure
	// We can't extract a single date from an array, so return an error with helpful message
	if nestedArr, ok := item.([]interface{}); ok {
		return time.Time{}, fmt.Errorf("nested array detected (array contains %d elements). the javascript expression may be returning a nested array like [[date1, date2]] instead of [date1, date2]", len(nestedArr))
	}

	// Handle goja.Value (Date objects might be exported as goja.Value)
	if gojaVal, ok := item.(goja.Value); ok {
		if gojaVal.SameAs(goja.Null()) || gojaVal.SameAs(goja.Undefined()) {
			return time.Time{}, fmt.Errorf("value is null or undefined")
		}

		exportedDate := gojaVal.Export()
		logrus.Debugf("[DateTool] goja.Value exported to: %v (Type: %T)", exportedDate, exportedDate)

		if dateObj, ok := exportedDate.(time.Time); ok {
			return dateObj, nil
		}

		// Try to get ISO string from the date
		if str := gojaVal.String(); str != "" {
			if parsed, err := time.Parse(time.RFC3339, str); err == nil {
				return parsed, nil
			}
		}
	}

	// Handle map (object representation of Date)
	if dateMap, ok := item.(map[string]interface{}); ok {
		// Try to extract ISO string from Date object
		if isoStr, ok := dateMap["iso"].(string); ok {
			if parsed, err := time.Parse(time.RFC3339, isoStr); err == nil {
				return parsed, nil
			}
		}
		// Try other common properties
		for _, key := range []string{"toISOString", "valueOf", "toString"} {
			if val, exists := dateMap[key]; exists {
				if str, ok := val.(string); ok {
					if parsed, err := time.Parse(time.RFC3339, str); err == nil {
						return parsed, nil
					}
				}
			}
		}
	}

	// Handle string
	if str, ok := item.(string); ok {
		if parsed, err := time.Parse(time.RFC3339, str); err == nil {
			return parsed, nil
		}
	}

	return time.Time{}, fmt.Errorf("not a valid Date object or ISO string (got type: %T, value: %v)", item, item)
}
