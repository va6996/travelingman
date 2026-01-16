package toolcalling

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"example.com/travelingman/providers/amadeus"
	"example.com/travelingman/providers/gemini"
	"example.com/travelingman/tools"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

const SystemPromptTemplate = `You are a helpful travel assistant. You have access to the following tools:

%s

Protocol:
1. To call a tool, output ONLY a JSON object in this format: {"tool": "toolName", "input": {...}}
2. Do not add any text before or after the JSON when calling a tool.
3. When you receive a Tool Result, use it to proceed.
4. If you have the final answer, output the text directly (no JSON).

Current Date: %s
User Query: %s`

// Input types for tools
type DateInput struct {
	Code string `json:"code"`
}

type FlightInput struct {
	Origin      string `json:"origin"`
	Destination string `json:"destination"`
	Date        string `json:"date"`
	Adults      int    `json:"adults"`
}

type HotelInput struct {
	CityCode string `json:"city_code"`
}

type AskUserInput struct {
	Question string `json:"question"`
}

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

// InitAgent initializes Genkit with tools and returns the Agent service
func InitAgent(amadeusClient *amadeus.Client, geminiClient gemini.Provider) (*Agent, error) {
	ctx := context.Background()

	// Initialize Genkit instance
	gk := genkit.Init(ctx)

	// Initialize Tools
	dateTool := &tools.DateTool{}
	flightTool := &tools.FlightTool{Client: amadeusClient}
	hotelTool := &tools.HotelTool{Client: amadeusClient}

	// Create agent instance
	agent := &Agent{
		tripData: nil,
	}

	// Define Genkit Tools
	// Capture tools to auto-generate system prompt descriptions
	var registeredTools []ai.Tool

	t1 := genkit.DefineTool[*DateInput, string](
		gk,
		"dateTool",
		"Executes JavaScript code. Code must return a Date object.",
		func(ctx *ai.ToolContext, input *DateInput) (string, error) {
			res, err := dateTool.Execute(map[string]interface{}{"code": input.Code})
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("%v", res), nil
		},
	)
	registeredTools = append(registeredTools, t1)

	t2 := genkit.DefineTool[FlightInput, string](
		gk,
		"flightTool",
		flightTool.Description(),
		func(ctx *ai.ToolContext, input FlightInput) (string, error) {
			args := map[string]interface{}{
				"origin":      input.Origin,
				"destination": input.Destination,
				"date":        input.Date,
				"adults":      input.Adults,
			}
			res, err := flightTool.Execute(args)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("%+v", res), nil
		},
	)
	registeredTools = append(registeredTools, t2)

	t3 := genkit.DefineTool[HotelInput, string](
		gk,
		"hotelTool",
		hotelTool.Description(),
		func(ctx *ai.ToolContext, input HotelInput) (string, error) {
			args := map[string]interface{}{
				"city_code": input.CityCode,
			}
			res, err := hotelTool.Execute(args)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("%+v", res), nil
		},
	)
	registeredTools = append(registeredTools, t3)

	t4 := genkit.DefineTool[AskUserInput, string](
		gk,
		"askUserTool",
		"Ask the user for more information. Use this when you need clarification or missing details.",
		func(ctx *ai.ToolContext, input AskUserInput) (string, error) {
			// Implementation handled in flow loop for interactivity,
			// but we define it here for schema generation.
			// The actual execution in the flow switch case intercepts this tool.
			return "", nil
		},
	)
	registeredTools = append(registeredTools, t4)

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
				log.Printf("[DEBUG] Step %d/%d: Prompting Gemini...", i+1, maxSteps)

				// 1. Generate content from Gemini
				resp, err := geminiClient.GenerateContent(history)
				if err != nil {
					log.Printf("[ERROR] Gemini generation failed: %v", err)
					return "", fmt.Errorf("gemini generation failed: %v", err)
				}
				log.Printf("[DEBUG] Gemini Response: %q", resp)

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

					switch toolCall.Tool {
					case "dateTool":
						log.Printf("[DEBUG] Executing dateTool...")
						toolRes, toolErr = dateTool.Execute(toolCall.Input)

					case "flightTool":
						log.Printf("[DEBUG] Executing flightTool...")
						if val, ok := toolCall.Input["adults"].(float64); ok {
							toolCall.Input["adults"] = int(val)
						}
						toolRes, toolErr = flightTool.Execute(toolCall.Input)

					case "hotelTool":
						log.Printf("[DEBUG] Executing hotelTool...")
						toolRes, toolErr = hotelTool.Execute(toolCall.Input)

					case "askUserTool":
						log.Printf("[DEBUG] Executing askUserTool...")
						// Prompt the user via Stdout and read from Stdin
						question, _ := toolCall.Input["question"].(string)
						fmt.Printf("\n[AI Request] %s\n> ", question)

						// Read input
						scanner := bufio.NewScanner(os.Stdin)
						if scanner.Scan() {
							toolRes = scanner.Text()
						} else {
							toolErr = fmt.Errorf("failed to read user input")
						}

					default:
						log.Printf("[WARN] Unknown tool requested: %s", toolCall.Tool)
						toolErr = fmt.Errorf("unknown tool: %s", toolCall.Tool)
					}

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
