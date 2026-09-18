package llm

import (
	"context"

	"github.com/c00/harnesser/types"
)

// LlmProvider is the stateless interface for interacting with an LLM API.
type LlmProvider interface {
	Generate(ctx context.Context, messages []types.Message, tools []types.Tool) (types.Message, error)
	// Return the config related to the llm
	Config() types.LlmConfig
}

type LlmStreamingProvider interface {
	LlmProvider
	GenerateStream(ctx context.Context, messages []types.Message, tools []types.Tool, callback StreamDeltaFunc) (types.Message, error)
}

type StreamDeltaFunc func(types.MessageDelta)

// StructuredOutputProvider is an llm provider that returns data in a specific format.
// Currently unused
// type StructuredOutputProvider interface {
// 	LlmProvider
// 	GenerateStructuredOutput(ctx context.Context, messages []models.Message, tools []models.Tool, output models.SchemaMarshaller) (models.LlmUsage, error)
// }
