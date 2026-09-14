//go:build integration

package openrouter

import (
	"os"
	"testing"

	"github.com/c00/harnesser/llm"
	"github.com/c00/harnesser/llm/llmtestsuite"
	"github.com/c00/harnesser/models"
)

func TestOpenRouter_LlmTestSuite(t *testing.T) {
	t.Parallel()

	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		t.Skip("No OPENROUTER_API_KEY set. Skipping openrouter llm test")
	}

	cfg := models.LlmConfig{
		Models:          []string{"openai/gpt-3.5-turbo-16k"},
		Name:            "openrouter",
		MaxOutputTokens: 25,
	}

	llmtestsuite.RunLlmTestSuite(t, func() llm.LlmProvider {
		provider := New(t.Context(), cfg, apiKey)
		return provider
	})
}
