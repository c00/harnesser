package interpolator

import (
	"testing"

	"github.com/c00/harnesser/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInterpolatedCommand(t *testing.T) {
	validSchema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path":    map[string]any{"type": "string"},
			"count":   map[string]any{"type": "number"},
			"payload": map[string]any{"type": "object"},
		},
	}

	tests := []struct {
		name    string
		tc      types.ToolCall
		td      types.ToolDefinition
		env     map[string]string
		want    []string
		wantErr bool
		errMsg  string
	}{
		{
			name: "simple string interpolation",
			tc:   types.ToolCall{Args: `{"path": "/tmp/foo.txt"}`},
			td: types.ToolDefinition{
				Tool: types.Tool{Parameters: validSchema},
				Command: &types.ToolCommand{
					Command: []string{"cat", "{{.Params.path}}"},
				},
			},
			want: []string{"cat", "/tmp/foo.txt"},
		},
		{
			name: "multiple params of different types",
			tc:   types.ToolCall{Args: `{"path": "log.txt", "count": 3}`},
			td: types.ToolDefinition{
				Tool: types.Tool{Parameters: validSchema},
				Command: &types.ToolCommand{
					Command: []string{"tail", "-n", "{{.Params.count}}", "{{.Params.path}}"},
				},
			},
			want: []string{"tail", "-n", "3", "log.txt"},
		},
		{
			name: "no params used",
			tc:   types.ToolCall{Args: `{}`},
			td: types.ToolDefinition{
				Tool: types.Tool{Parameters: validSchema},
				Command: &types.ToolCommand{
					Command: []string{"echo", "hello"},
				},
			},
			want: []string{"echo", "hello"},
		},
		{
			name: "json function marshals object",
			tc:   types.ToolCall{Args: `{"payload": {"a": 1, "b": "two"}}`},
			td: types.ToolDefinition{
				Tool: types.Tool{Parameters: validSchema},
				Command: &types.ToolCommand{
					Command: []string{`{{json .Params.payload}}`},
				},
			},
			want: []string{`{"a":1,"b":"two"}`},
		},
		{
			name: "allowed env vars are interpolated",
			tc:   types.ToolCall{Args: `{}`},
			td: types.ToolDefinition{
				Tool: types.Tool{Parameters: validSchema},
				Command: &types.ToolCommand{
					Command:    []string{"deploy", "{{.Env.HOME_DIR}}"},
					AllowedEnv: []string{"HOME_DIR"},
				},
			},
			env:  map[string]string{"HOME_DIR": "/home/coo"},
			want: []string{"deploy", "/home/coo"},
		},
		{
			name: "missing allowed env var errors",
			tc:   types.ToolCall{Args: `{}`},
			td: types.ToolDefinition{
				Tool: types.Tool{Parameters: validSchema},
				Command: &types.ToolCommand{
					Command:    []string{"deploy", "{{.Env.HOME_DIR}}"},
					AllowedEnv: []string{"HOME_DIR"},
				},
			},
			wantErr: true,
			errMsg:  "missing env variable: HOME_DIR",
		},
		{
			name: "nil schema treated as empty schema",
			tc:   types.ToolCall{Args: `{"path": "/tmp/foo.txt"}`},
			td: types.ToolDefinition{
				Tool: types.Tool{Parameters: nil},
				Command: &types.ToolCommand{
					Command: []string{"cat", "{{.Params.path}}"},
				},
			},
			want: []string{"cat", "/tmp/foo.txt"},
		},
		{
			name: "args not matching schema errors",
			tc:   types.ToolCall{Args: `{"count": "not-a-number"}`},
			td: types.ToolDefinition{
				Tool: types.Tool{
					Parameters: map[string]any{
						"type": "object",
						"properties": map[string]any{
							"count": map[string]any{"type": "number"},
						},
					},
				},
				Command: &types.ToolCommand{
					Command: []string{"echo", "{{.Params.count}}"},
				},
			},
			wantErr: true,
			errMsg:  "tool call parameters not valid",
		},
		{
			name: "invalid json args errors",
			tc:   types.ToolCall{Args: `{invalid`},
			td: types.ToolDefinition{
				Command: &types.ToolCommand{
					Command: []string{"echo"},
				},
			},
			wantErr: true,
			errMsg:  "cannot get toolcall args as parameters",
		},
		{
			name: "missing param with missingkey=error errors",
			tc:   types.ToolCall{Args: `{}`},
			td: types.ToolDefinition{
				Tool: types.Tool{Parameters: validSchema},
				Command: &types.ToolCommand{
					Command: []string{"cat", "{{.Params.path}}"},
				},
			},
			wantErr: true,
			errMsg:  "cannot execute template for part {{.Params.path}}",
		},
		{
			name: "invalid template syntax errors",
			tc:   types.ToolCall{Args: `{}`},
			td: types.ToolDefinition{
				Tool: types.Tool{Parameters: validSchema},
				Command: &types.ToolCommand{
					Command: []string{"echo", "{{.Params.path"},
				},
			},
			wantErr: true,
			errMsg:  "cannot parse template",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.env {
				t.Setenv(k, v)
			}

			got, _, err := InterpolatedCommand(tt.tc, tt.td)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				assert.Nil(t, got)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
