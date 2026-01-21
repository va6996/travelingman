package core

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/va6996/travelingman/tools"
	"github.com/firebase/genkit/go/genkit"
)

type AskUserInput struct {
	Question string `json:"question"`
}

type AskUserTool struct{}

func (t *AskUserTool) Name() string {
	return "ask_user_tool"
}

func (t *AskUserTool) Description() string {
	return "Ask the user for more information. Use this when you need clarification or missing details. Arguments: question (string)."
}

func (t *AskUserTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	question, _ := args["question"].(string)
	if question == "" {
		return nil, fmt.Errorf("question is required")
	}

	fmt.Printf("\n[AI Request] %s\n> ", question)

	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		return strings.TrimSpace(scanner.Text()), nil
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading input: %w", err)
	}

	return "", fmt.Errorf("no input provided")
}

func NewAskUserTool(gk *genkit.Genkit, registry *tools.Registry) *AskUserTool {
	t := &AskUserTool{}

	if gk == nil || registry == nil {
		return t
	}

	// registry.Register(genkit.DefineTool[AskUserInput, string](
	// 	gk,
	// 	"askUserTool",
	// 	t.Description(),
	// 	func(ctx *ai.ToolContext, input AskUserInput) (string, error) {
	// 		res, err := t.Execute(ctx, map[string]interface{}{"question": input.Question})
	// 		if err != nil {
	// 			return "", err
	// 		}
	// 		return fmt.Sprintf("%v", res), nil
	// 	},
	// ), t.Execute)

	return t
}
