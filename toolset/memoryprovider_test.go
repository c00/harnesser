package toolset

import (
	"testing"

	"github.com/c00/harnesser/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryProviderDefinitions(t *testing.T) {
	tests := []struct {
		name string
		act  func(*MemoryProvider)
		want []types.ToolDefinition
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
			want: []types.ToolDefinition{
				testToolDefinition("first", true),
				testToolDefinition("second", false),
			},
		},
		{
			name: "set replaces definitions",
			act: func(provider *MemoryProvider) {
				provider.AddDefinition(testToolDefinition("old", true))
				provider.SetDefinitions([]types.ToolDefinition{testToolDefinition("new", true)})
			},
			want: []types.ToolDefinition{testToolDefinition("new", true)},
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
	provider.SetDefinitions([]types.ToolDefinition{
		testToolDefinition("enabled-first", true),
		testToolDefinition("disabled", false),
		testToolDefinition("enabled-second", true),
	})

	tools := provider.GetTools()

	assert.Equal(t, types.Tools{
		testToolDefinition("enabled-first", true).Tool,
		testToolDefinition("enabled-second", true).Tool,
	}, tools)
}

func TestMemoryProviderGetDefinition(t *testing.T) {
	definitions := []types.ToolDefinition{
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
		assert.Equal(t, types.ToolDefinition{}, definition)
	})
}

func testToolDefinition(name string, enabled bool) types.ToolDefinition {
	return types.ToolDefinition{
		Enabled: enabled,
		Trusted: true,
		Tool: types.Tool{
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
