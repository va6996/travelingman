package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/firebase/genkit/go/genkit"
)

// LLMClient defines the interface for LLM interaction
type LLMClient interface {
	GenerateContent(ctx context.Context, prompt string) (string, error)
}

const SystemPromptTemplate = `You are a helpful travel assistant. You have access to the following tools:

%s

Protocol:
1. To call a tool, output ONLY a JSON object in this format: {"tool": "toolName", "input": {...}}
2. Do not add any text before or after the JSON when calling a tool.
3. When you receive a Tool Result, use it to proceed.
4. If you have the final answer, output the text directly (no JSON).

Current Date: %s
User Query: %s`

// ToolCallResult stores the result of a tool call
type ToolCallResult struct {
	ToolName  string      `json:"tool_name"`
	Input     interface{} `json:"input"`
	Output    interface{} `json:"output"`
	Error     string      `json:"error,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

// TripData stores all collected trip information
type TripData struct {
	Query      string           `json:"query"`
	Flights    []interface{}    `json:"flights,omitempty"`
	Hotels     []interface{}    `json:"hotels,omitempty"`
	Dates      []interface{}    `json:"dates,omitempty"`
	UserInputs []string         `json:"user_inputs,omitempty"`
	RawResults []ToolCallResult `json:"raw_results"`
	CreatedAt  time.Time        `json:"created_at"`
}

// Agent wraps Genkit operations
type Agent struct {
	flow     FlowRunner
	tripData *TripData
	llm      LLMClient
}

// FlowRunner defines the interface for running a flow
type FlowRunner interface {
	Run(ctx context.Context, input string) (string, error)
}

// PlanTrip executes the trip planning flow
func (a *Agent) PlanTrip(ctx context.Context, query string) (string, error) {
	// Initialize trip data for this query
	a.tripData = &TripData{
		Query:     query,
		CreatedAt: time.Now(),
	}
	return a.flow.Run(ctx, query)
}

// GetTripData returns the collected trip data
func (a *Agent) GetTripData() *TripData {
	return a.tripData
}

// GetFlights returns the collected flight data
func (a *Agent) GetFlights() []interface{} {
	if a.tripData == nil {
		return nil
	}
	return a.tripData.Flights
}

// GetHotels returns the collected hotel data
func (a *Agent) GetHotels() []interface{} {
	if a.tripData == nil {
		return nil
	}
	return a.tripData.Hotels
}

// GetRawResults returns all tool call results
func (a *Agent) GetRawResults() []ToolCallResult {
	if a.tripData == nil {
		return nil
	}
	return a.tripData.RawResults
}

// ToJSON exports the trip data as JSON
func (td *TripData) ToJSON() ([]byte, error) {
	return json.MarshalIndent(td, "", "  ")
}

// ToJSON exports the agent's trip data as JSON
func (a *Agent) ToJSON() ([]byte, error) {
	if a.tripData == nil {
		return nil, fmt.Errorf("no trip data available")
	}
	return a.tripData.ToJSON()
}

// NewAgent creates a new Agent service using provided Genkit and Registry
func NewAgent(gk *genkit.Genkit, registry *Registry, llmPlugin LLMClient) (*Agent, error) {

	// Create agent instance
	agent := &Agent{
		tripData: nil,
		llm:      llmPlugin,
	}

	// Capture tools to auto-generate system prompt descriptions
	registeredTools := registry.GetTools()

	flow := genkit.DefineFlow(
		gk,
		"planTripFlow",
		func(ctx context.Context, input string) (string, error) {
			log.Printf("[DEBUG] Starting planTripFlow with input: %q", input)

			// Auto-generate tool definitions
			var toolDefsBuilder strings.Builder
			for _, t := range registeredTools {
				def := t.Definition()
				schemaBytes, _ := json.Marshal(def.InputSchema)
				fmt.Fprintf(
					&toolDefsBuilder,
					"Tool: %s\nDescription: %s\nInput Schema: %s\n\n",
					def.Name,
					def.Description,
					string(schemaBytes),
				)
			}
			toolDefs := toolDefsBuilder.String()
			log.Printf("[DEBUG] Generated tool definitions for prompt")

			// System Prompt defining tools and behavioral contract
			systemPrompt := fmt.Sprintf(
				SystemPromptTemplate,
				toolDefs,
				time.Now().Format(time.RFC3339),
				input,
			)

			history := systemPrompt
			maxSteps := 50

			for i := 0; i < maxSteps; i++ {
				// Check for context cancellation
				select {
				case <-ctx.Done():
					return "", ctx.Err()
				default:
				}

				log.Printf("[DEBUG] Step %d/%d: Prompting LLM...", i+1, maxSteps)

				// 1. Generate content from Gemini
				resp, err := agent.llm.GenerateContent(ctx, history)
				if err != nil {
					log.Printf("[ERROR] LLM generation failed: %v", err)
					return "", fmt.Errorf("llm generation failed: %v", err)
				}
				log.Printf("[DEBUG] LLM Response: %q", resp)

				// 2. Parse response for tool call
				// We scan for the first '{' and last '}' to handle markdown blocks or preamble
				start := strings.Index(resp, "{")
				end := strings.LastIndex(resp, "}")

				isToolCall := false
				var toolCall struct {
					Tool  string                 `json:"tool"`
					Input map[string]interface{} `json:"input"`
				}

				if start != -1 && end != -1 && end > start {
					potentialJSON := resp[start : end+1]
					if err := json.Unmarshal([]byte(potentialJSON), &toolCall); err == nil {
						// Double check required fields to avoid false positives on random JSON in text
						if toolCall.Tool != "" {
							isToolCall = true
							log.Printf("[DEBUG] Detected valid tool call JSON in response")
							// CRITICAL: Append the model's internal thought/request to history
							// so it remembers that IT asked for this tool.
							history += fmt.Sprintf("\nModel Response: %s\n", resp)
						}
					}
				}

				if isToolCall {
					log.Printf("[DEBUG] Valid Tool Call: %s with Input: %+v", toolCall.Tool, toolCall.Input)

					var toolRes interface{}
					var toolErr error

					// Generic Tool Execution
					log.Printf("[DEBUG] Executing tool: %s", toolCall.Tool)
					toolRes, toolErr = registry.ExecuteTool(ctx, toolCall.Tool, toolCall.Input)

					// 3. Store Tool Result and Append to History
					result := ToolCallResult{
						ToolName:  toolCall.Tool,
						Input:     toolCall.Input,
						Timestamp: time.Now(),
					}

					if toolErr != nil {
						log.Printf("[ERROR] Tool Execution Failed: %v", toolErr)
						result.Error = toolErr.Error()
						history += fmt.Sprintf("\nTool '%s' Error: %v\n", toolCall.Tool, toolErr)
					} else {
						log.Printf("[DEBUG] Tool Execution Success: %+v", toolRes)
						result.Output = toolRes
						history += fmt.Sprintf("\nTool '%s' Output: %v\n", toolCall.Tool, toolRes)

						// Store categorized results
						if agent.tripData != nil {
							switch toolCall.Tool {
							case "flightTool":
								agent.tripData.Flights = append(agent.tripData.Flights, toolRes)
							case "hotelTool":
								agent.tripData.Hotels = append(agent.tripData.Hotels, toolRes)
							case "dateTool":
								agent.tripData.Dates = append(agent.tripData.Dates, toolRes)
							case "askUserTool":
								if inputStr, ok := toolRes.(string); ok {
									agent.tripData.UserInputs = append(agent.tripData.UserInputs, inputStr)
								}
							}
						}
					}

					// Store raw result
					if agent.tripData != nil {
						agent.tripData.RawResults = append(agent.tripData.RawResults, result)
					}
					continue // Loop again to get next step or final answer
				}

				// If not valid JSON or unmarshal failed, assume it's the final answer text
				log.Printf("[DEBUG] Returning Final Answer: %q", resp)
				return resp, nil
			}

			log.Printf("[WARN] Max steps exceeded in planning loop")
			return "", fmt.Errorf("max steps exceeded")
		},
	)

	log.Println("Genkit initialized with tools and flows.")
	agent.flow = flow
	return agent, nil
}
