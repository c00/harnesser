package llm

import (
	"context"

	"github.com/c00/harnesser/models"
)

// LlmProvider is the stateless interface for interacting with an LLM API.
type LlmProvider interface {
	Generate(ctx context.Context, messages []models.Message, tools []models.Tool) (models.Message, error)
	// Return the config related to the llm
	Config() models.LlmConfig
}

// StructuredOutputProvider is an llm provider that returns data in a specific format.
type StructuredOutputProvider interface {
	LlmProvider
	GenerateStructuredOutput(ctx context.Context, messages []models.Message, tools []models.Tool, output models.SchemaMarshaller) (models.LlmUsage, error)
}
