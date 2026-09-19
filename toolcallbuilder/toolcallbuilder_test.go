package toolcallbuilder

import (
	"testing"

	"github.com/c00/harnesser/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewToolCallBuilder(t *testing.T) {
	tests := []struct {
		name        string
		toolCall    types.ToolCall
		definition  types.ToolDefinition
		wantName    string
		wantArgs    []string
		wantCommand string
		wantErr     string
	}{
		{
			name:     "builds interpolated command",
			toolCall: types.ToolCall{Args: `{"path":"notes.txt"}`},
			definition: types.ToolDefinition{
				Command: &types.ToolCommand{
					Command: []string{"cat", "{{.Params.path}}"},
				},
			},
			wantName:    "cat",
			wantArgs:    []string{"notes.txt"},
			wantCommand: "cat notes.txt ",
		},
		{
			name:     "appends args from parameter and removes empty args",
			toolCall: types.ToolCall{Args: `{"files":["one.txt","","two.txt"]}`},
			definition: types.ToolDefinition{
				Command: &types.ToolCommand{
					Command:  []string{"open", "", "--read-only"},
					ArgsFrom: "files",
				},
			},
			wantName:    "open",
			wantArgs:    []string{"--read-only", "one.txt", "two.txt"},
			wantCommand: "open --read-only one.txt two.txt",
		},
		{
			name:       "rejects missing command",
			toolCall:   types.ToolCall{Args: `{}`},
			definition: types.ToolDefinition{},
			wantErr:    "tool call does not have a command",
		},
		{
			name:     "wraps interpolation error",
			toolCall: types.ToolCall{Args: `{}`},
			definition: types.ToolDefinition{
				Command: &types.ToolCommand{
					Command: []string{"echo", "{{.Params.missing}}"},
				},
			},
			wantErr: "cannot interpolate command: cannot execute template",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder, err := NewToolCallCmdBuilder(tt.toolCall, tt.definition)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErr)
				assert.Nil(t, builder)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantName, builder.CommandName())
			assert.Equal(t, tt.wantArgs, builder.CommandArgs())
			assert.Equal(t, tt.wantCommand, builder.CommandString())
		})
	}
}

func TestToolCallBuilderCommandArgsReturnsCopy(t *testing.T) {
	builder, err := NewToolCallCmdBuilder(
		types.ToolCall{Args: `{"files":["one.txt"]}`},
		types.ToolDefinition{
			Command: &types.ToolCommand{
				Command:  []string{"open", "--read-only"},
				ArgsFrom: "files",
			},
		},
	)
	require.NoError(t, err)

	args := builder.CommandArgs()
	args[0] = "--modified"

	assert.Equal(t, []string{"--read-only", "one.txt"}, builder.CommandArgs())
}

func TestToolCallBuilderBuild(t *testing.T) {
	tests := []struct {
		name      string
		builder   ToolCallCmdBuilder
		wantName  string
		wantArgs  []string
		wantExtra []string
		wantErr   string
	}{
		{
			name: "builds command fields",
			builder: ToolCallCmdBuilder{
				toolCall: types.ToolCall{Args: `{"value":"hello"}`},
				toolCallDef: types.ToolDefinition{
					Command: &types.ToolCommand{
						Command: []string{"echo", "{{.Params.value}}"},
					},
				},
			},
			wantName:  "echo",
			wantArgs:  []string{"hello"},
			wantExtra: []string{},
		},
		{
			name: "does not rebuild initialized fields",
			builder: ToolCallCmdBuilder{
				toolCall:         types.ToolCall{Args: `{invalid`},
				toolCallDef:      types.ToolDefinition{},
				commandName:      "existing",
				interpolatedArgs: []string{"first"},
				extraArgs:        []string{"second"},
			},
			wantName:  "existing",
			wantArgs:  []string{"first"},
			wantExtra: []string{"second"},
		},
		{
			name: "returns interpolation errors",
			builder: ToolCallCmdBuilder{
				toolCall: types.ToolCall{Args: `{invalid`},
				toolCallDef: types.ToolDefinition{
					Command: &types.ToolCommand{
						Command: []string{"echo"},
					},
				},
			},
			wantErr: "cannot interpolate command: cannot get toolcall args as parameters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := tt.builder

			err := builder.build()
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantName, builder.commandName)
			assert.Equal(t, tt.wantArgs, builder.interpolatedArgs)
			assert.Equal(t, tt.wantExtra, builder.extraArgs)
		})
	}
}

func TestGetStringSlice(t *testing.T) {
	tests := []struct {
		name   string
		params map[string]any
		key    string
		want   []string
	}{
		{
			name:   "missing key",
			params: map[string]any{"other": []string{"value"}},
			key:    "args",
			want:   []string{},
		},
		{
			name:   "string slice",
			params: map[string]any{"args": []string{"one", "two"}},
			key:    "args",
			want:   []string{"one", "two"},
		},
		{
			name:   "any slice keeps only strings",
			params: map[string]any{"args": []any{"one", 2, nil, "two"}},
			key:    "args",
			want:   []string{"one", "two"},
		},
		{
			name:   "unsupported value",
			params: map[string]any{"args": "one"},
			key:    "args",
			want:   []string{},
		},
		{
			name:   "nil params",
			params: nil,
			key:    "args",
			want:   []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, getStringSlice(tt.params, tt.key))
		})
	}
}
