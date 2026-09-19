package toolset

import (
	"context"
	"errors"
	"testing"

	"github.com/c00/harnesser/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunDecision(t *testing.T) {
	tests := []struct {
		name     string
		trusted  bool
		decision types.ToolCallDecision
		wantErr  error
	}{
		{name: "untrusted without decision", wantErr: ErrNoDecision},
		{name: "untrusted with explicit no decision", decision: types.ToolCallDecisionNoDecision, wantErr: ErrNoDecision},
		{name: "untrusted rejected", decision: types.ToolCallDecisionReject, wantErr: ErrRejected},
		{name: "untrusted approved", decision: types.ToolCallDecisionApprove},
		{name: "trusted without decision", trusted: true},
		{name: "trusted ignores rejection", trusted: true, decision: types.ToolCallDecisionReject},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			def := callbackDefinition("callback", true, tt.trusted, func(context.Context, map[string]any) (string, error) {
				called = true
				return "result", nil
			})
			call := types.ToolCall{
				ToolCallID: "call-1",
				Function:   "callback",
				Decision:   tt.decision,
			}

			result, err := Run(t.Context(), def, call)

			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
				assert.ErrorContains(t, err, "call-1")
				assert.ErrorContains(t, err, "callback")
				assert.False(t, called)
				assert.Empty(t, result)
				return
			}
			require.NoError(t, err)
			assert.True(t, called)
			assert.Equal(t, "result", result)
		})
	}
}

func TestRunRejectsMalformedDefinition(t *testing.T) {
	callback := func(context.Context, map[string]any) (string, error) {
		return "", nil
	}
	tests := []struct {
		name    string
		def     types.ToolDefinition
		wantErr string
	}{
		{
			name: "callback and command",
			def: types.ToolDefinition{
				Trusted:  true,
				Callback: callback,
				Command:  &types.ToolCommand{Command: []string{"echo"}},
			},
			wantErr: "both callback and command are set",
		},
		{
			name:    "no callback or command",
			def:     types.ToolDefinition{Trusted: true},
			wantErr: "neither callback nor command are set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Run(t.Context(), tt.def, types.ToolCall{})

			require.Error(t, err)
			assert.ErrorContains(t, err, tt.wantErr)
			assert.Empty(t, result)
		})
	}
}

func TestRunCallback(t *testing.T) {
	t.Run("decodes arguments and passes the context", func(t *testing.T) {
		type contextKey string
		ctx := context.WithValue(t.Context(), contextKey("key"), "context value")
		var gotParams map[string]any
		var gotContextValue any
		def := callbackDefinition("callback", true, true, func(ctx context.Context, params map[string]any) (string, error) {
			gotParams = params
			gotContextValue = ctx.Value(contextKey("key"))
			return "callback result", nil
		})

		result, err := Run(ctx, def, types.ToolCall{Args: `{"name":"Ada","count":2,"nested":{"active":true}}`})

		require.NoError(t, err)
		assert.Equal(t, "callback result", result)
		assert.Equal(t, "context value", gotContextValue)
		assert.Equal(t, map[string]any{
			"name":   "Ada",
			"count":  float64(2),
			"nested": map[string]any{"active": true},
		}, gotParams)
	})

	t.Run("passes an empty map when arguments are empty", func(t *testing.T) {
		var gotParams map[string]any
		def := callbackDefinition("callback", true, true, func(_ context.Context, params map[string]any) (string, error) {
			gotParams = params
			return "", nil
		})

		_, err := Run(t.Context(), def, types.ToolCall{})

		require.NoError(t, err)
		assert.NotNil(t, gotParams)
		assert.Empty(t, gotParams)
	})

	t.Run("rejects invalid arguments without invoking callback", func(t *testing.T) {
		called := false
		def := callbackDefinition("callback", true, true, func(context.Context, map[string]any) (string, error) {
			called = true
			return "", nil
		})

		result, err := Run(t.Context(), def, types.ToolCall{Args: "{"})

		require.Error(t, err)
		assert.ErrorContains(t, err, "cannot unmarshall params")
		assert.False(t, called)
		assert.Empty(t, result)
	})

	t.Run("returns callback errors", func(t *testing.T) {
		callbackErr := errors.New("callback failed")
		def := callbackDefinition("callback", true, true, func(context.Context, map[string]any) (string, error) {
			return "partial result", callbackErr
		})

		result, err := Run(t.Context(), def, types.ToolCall{})

		assert.ErrorIs(t, err, callbackErr)
		assert.Equal(t, "partial result", result)
	})
}

func TestRunCommand(t *testing.T) {
	t.Run("returns command output", func(t *testing.T) {
		def := types.ToolDefinition{
			Trusted: true,
			Tool:    types.Tool{Name: "printf"},
			Command: &types.ToolCommand{Command: []string{"printf", "%s", "{{.Params.value}}"}},
		}

		result, err := Run(t.Context(), def, types.ToolCall{Args: `{"value":"hello"}`})

		require.NoError(t, err)
		assert.Equal(t, "hello", result)
	})

	t.Run("wraps command builder errors", func(t *testing.T) {
		def := types.ToolDefinition{
			Trusted: true,
			Tool:    types.Tool{Name: "invalid"},
			Command: &types.ToolCommand{},
		}

		result, err := Run(t.Context(), def, types.ToolCall{Args: `{}`})

		require.Error(t, err)
		assert.ErrorContains(t, err, "cannot create tool call builder")
		assert.ErrorContains(t, err, "tool call does not have a command")
		assert.Empty(t, result)
	})

	t.Run("includes stderr when a command fails", func(t *testing.T) {
		def := types.ToolDefinition{
			Trusted: true,
			Tool:    types.Tool{Name: "failing"},
			Command: &types.ToolCommand{Command: []string{"sh", "-c", "printf 'failure details' >&2; exit 7"}},
		}

		result, err := Run(t.Context(), def, types.ToolCall{Args: `{}`})

		require.Error(t, err)
		assert.ErrorContains(t, err, "running tool 'failing' failed")
		assert.ErrorContains(t, err, "exit status 7")
		assert.ErrorContains(t, err, "failure details")
		assert.Empty(t, result)
	})

	t.Run("reports a missing executable", func(t *testing.T) {
		def := types.ToolDefinition{
			Trusted: true,
			Tool:    types.Tool{Name: "missing"},
			Command: &types.ToolCommand{Command: []string{"harnesser-test-command-that-does-not-exist"}},
		}

		result, err := Run(t.Context(), def, types.ToolCall{Args: `{}`})

		require.Error(t, err)
		assert.ErrorContains(t, err, "running tool 'missing' failed")
		assert.Empty(t, result)
	})
}
