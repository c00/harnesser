package llmtestsuite

import (
	"strings"
	"testing"

	"github.com/c00/harnesser/llm"
	"github.com/c00/harnesser/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This test suite should work for all LlmProviders.
// It tests basic responses and basic tool calling.
func RunLlmTestSuite(t *testing.T, llmFactory func() llm.LlmProvider) {
	basicResponse(t, llmFactory)
	basicToolCalling(t, llmFactory)
	// Add more tests as needed
}

func basicResponse(t *testing.T, llmFactory func() llm.LlmProvider) {
	llm := llmFactory()
	response, err := llm.Generate(
		t.Context(),
		[]models.Message{models.NewUserTextMessage("Just say the word 'Hi'. Nothing else. This is to test if you're working. Only reply with 'Hi'")},
		[]models.Tool{},
	)

	require.NoError(t, err)

	assert.Contains(t, "hi", strings.ToLower(response.Content.String()))
}

func basicToolCalling(t *testing.T, llmFactory func() llm.LlmProvider) {
	llm := llmFactory()
	response, err := llm.Generate(
		t.Context(),
		[]models.Message{models.NewUserTextMessage("Can tell me the weather in Amsterdam?")},
		[]models.Tool{TestTool},
	)

	require.NoError(t, err)

	assert.Len(t, response.ToolCalls, 1)
	assert.Contains(t, strings.ToLower(response.ToolCalls[0].Args), "amsterdam")
}
