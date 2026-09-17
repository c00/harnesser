package promptsprovider

import (
	"testing"

	"github.com/c00/harnesser/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPrompt(t *testing.T) {
	prompt := NewPrompt("system", "You are a helpful assistant")

	assert.True(t, prompt.Active)
	assert.Equal(t, "system", prompt.Key)
	assert.Equal(t, models.RoleSystem, prompt.Message.Role)
	assert.Equal(t, "You are a helpful assistant", prompt.Message.Content.String())
}

func TestPromptsActiveMessages(t *testing.T) {
	prompts := Prompts{
		NewPrompt("first", "first prompt"),
		{
			Active:  false,
			Key:     "inactive",
			Message: models.NewSystemMessage("inactive prompt"),
		},
		NewPrompt("second", "second prompt"),
	}

	messages := prompts.ActiveMessages()

	require.Len(t, messages, 2)
	assert.Equal(t, "first prompt", messages[0].Content.String())
	assert.Equal(t, "second prompt", messages[1].Content.String())
}
