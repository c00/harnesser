package systemprompts

import (
	"testing"

	"github.com/c00/harnesser/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryProvider(t *testing.T) {
	tests := []struct {
		name      string
		act       func(*MemoryProvider)
		wantKeys  []string
		wantTexts []string
	}{
		{
			name: "starts empty",
			act:  func(*MemoryProvider) {},
		},
		{
			name: "adds prompts",
			act: func(provider *MemoryProvider) {
				provider.AddPrompt(NewPrompt("first", "first prompt"))
				provider.AddPrompt(NewPrompt("second", "second prompt"))
			},
			wantKeys:  []string{"first", "second"},
			wantTexts: []string{"first prompt", "second prompt"},
		},
		{
			name: "set replaces prompts",
			act: func(provider *MemoryProvider) {
				provider.AddPrompt(NewPrompt("old", "old prompt"))
				provider.SetPrompts(Prompts{NewPrompt("new", "new prompt")})
			},
			wantKeys:  []string{"new"},
			wantTexts: []string{"new prompt"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := &MemoryProvider{}
			tt.act(provider)

			prompts := provider.Prompts()
			require.Len(t, prompts, len(tt.wantKeys))
			for i := range prompts {
				assert.Equal(t, tt.wantKeys[i], prompts[i].Key)
				assert.Equal(t, tt.wantTexts[i], prompts[i].Message.Content.String())
				assert.True(t, prompts[i].Active)
				assert.Equal(t, types.RoleSystem, prompts[i].Message.Role)
			}
		})
	}
}
