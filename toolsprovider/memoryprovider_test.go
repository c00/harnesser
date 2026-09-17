package toolsprovider

import (
	"testing"

	"github.com/c00/harnesser/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryProviderDefinitions(t *testing.T) {
	tests := []struct {
		name string
		act  func(*MemoryProvider)
		want []models.ToolDefinition
	}{
		{
			name: "starts empty",
			act:  func(*MemoryProvider) {},
			want: nil,
		},
		{
			name: "adds definitions",
			act: func(provider *MemoryProvider) {
				provider.AddDefinition(testToolDefinition("first", true))
				provider.AddDefinition(testToolDefinition("second", false))
			},
			want: []models.ToolDefinition{
				testToolDefinition("first", true),
				testToolDefinition("second", false),
			},
		},
		{
			name: "set replaces definitions",
			act: func(provider *MemoryProvider) {
				provider.AddDefinition(testToolDefinition("old", true))
				provider.SetDefinitions([]models.ToolDefinition{testToolDefinition("new", true)})
			},
			want: []models.ToolDefinition{testToolDefinition("new", true)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := NewMemoryProvider()
			tt.act(provider)

			assert.Equal(t, tt.want, provider.GetDefinitions())
		})
	}
}

func TestMemoryProviderGetTools(t *testing.T) {
	provider := NewMemoryProvider()
	provider.SetDefinitions([]models.ToolDefinition{
		testToolDefinition("enabled-first", true),
		testToolDefinition("disabled", false),
		testToolDefinition("enabled-second", true),
	})

	tools := provider.GetTools()

	assert.Equal(t, models.Tools{
		testToolDefinition("enabled-first", true).Tool,
		testToolDefinition("enabled-second", true).Tool,
	}, tools)
}

func TestMemoryProviderGetDefinition(t *testing.T) {
	definitions := []models.ToolDefinition{
		testToolDefinition("first", true),
		testToolDefinition("second", false),
	}
	provider := NewMemoryProvider()
	provider.SetDefinitions(definitions)

	t.Run("returns definition by tool name", func(t *testing.T) {
		definition, err := provider.GetDefinition("second")

		require.NoError(t, err)
		assert.Equal(t, definitions[1], definition)
	})

	t.Run("returns ErrNotFound for an unknown tool", func(t *testing.T) {
		definition, err := provider.GetDefinition("missing")

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrNotFound)
		assert.ErrorContains(t, err, "missing")
		assert.Equal(t, models.ToolDefinition{}, definition)
	})
}

func testToolDefinition(name string, enabled bool) models.ToolDefinition {
	return models.ToolDefinition{
		Enabled: enabled,
		Trusted: true,
		Tool: models.Tool{
			Name:        name,
			Description: name + " description",
			Parameters: map[string]any{
				"type": "object",
			},
		},
		Command:    []string{"run", name},
		ArgsFrom:   "arguments",
		AllowedEnv: []string{"PATH"},
	}
}
