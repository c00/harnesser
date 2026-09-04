package llmtestsuite

import "github.com/c00/harnesser/models"

// TestTool is used for tests where the execution of a tool call is needed
var TestTool = models.Tool{
	Name:        "get_weather",
	Description: "Get the weather in a city",
	// Example: { "city": "Amsterdam" }
	Parameters: map[string]any{
		"type": "object",
		"properties": map[string]any{
			"city": map[string]any{
				"type": "string",
			},
		},
		"required": []string{"city"},
	},
}
