package toolset

import (
	"context"
	"errors"
	"testing"

	"github.com/c00/harnesser/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubToolsLoader struct {
	defs []types.ToolDefinition
	err  error
}

func (l stubToolsLoader) Load() ([]types.ToolDefinition, error) {
	return l.defs, l.err
}

func TestNewRegistry(t *testing.T) {
	t.Run("creates an empty registry without a loader", func(t *testing.T) {
		registry, err := NewRegistry(nil)

		require.NoError(t, err)
		assert.Empty(t, registry.GetTools())
	})

	t.Run("loads definitions", func(t *testing.T) {
		def := callbackDefinition("loaded", true, true, nil)

		registry, err := NewRegistry(stubToolsLoader{defs: []types.ToolDefinition{def}})

		require.NoError(t, err)
		got, err := registry.GetToolDef("loaded")
		require.NoError(t, err)
		assert.Equal(t, def, got)
	})

	t.Run("wraps loader errors", func(t *testing.T) {
		loadErr := errors.New("load failed")

		registry, err := NewRegistry(stubToolsLoader{err: loadErr})

		require.Error(t, err)
		assert.ErrorIs(t, err, loadErr)
		assert.ErrorContains(t, err, "cannot load tools")
		assert.Nil(t, registry)
	})
}

func TestToolRegistryDefinitions(t *testing.T) {
	registry := NewEmptyRegistry()
	enabled := callbackDefinition("enabled", true, true, nil)
	disabled := callbackDefinition("disabled", false, true, nil)
	registry.SetToolDefs(enabled, disabled)

	t.Run("gets a definition", func(t *testing.T) {
		got, err := registry.GetToolDef("enabled")

		require.NoError(t, err)
		assert.Equal(t, enabled, got)
	})

	t.Run("reports a missing definition", func(t *testing.T) {
		got, err := registry.GetToolDef("missing")

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrNotFound)
		assert.ErrorContains(t, err, "missing")
		assert.Equal(t, types.ToolDefinition{}, got)
	})

	t.Run("returns only enabled tools", func(t *testing.T) {
		assert.Equal(t, types.Tools{enabled.Tool}, registry.GetTools())
	})

	t.Run("updates a definition with the same name", func(t *testing.T) {
		updated := callbackDefinition("enabled", false, false, nil)
		registry.SetToolDefs(updated)

		got, err := registry.GetToolDef("enabled")
		require.NoError(t, err)
		assert.Equal(t, updated, got)
		assert.Empty(t, registry.GetTools())
	})

	t.Run("removes a definition", func(t *testing.T) {
		registry.RemoveToolDef("disabled")
		registry.RemoveToolDef("does-not-exist")

		_, err := registry.GetToolDef("disabled")
		assert.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("clears all definitions", func(t *testing.T) {
		registry.SetToolDefs(enabled, disabled)

		registry.Clear()

		assert.Empty(t, registry.GetTools())
		_, err := registry.GetToolDef("enabled")
		assert.ErrorIs(t, err, ErrNotFound)
	})
}

func TestToolRegistryLoad(t *testing.T) {
	registry := NewEmptyRegistry()
	existing := callbackDefinition("existing", true, true, nil)
	loaded := callbackDefinition("loaded", true, true, nil)
	registry.SetToolDefs(existing)

	err := registry.Load(stubToolsLoader{defs: []types.ToolDefinition{loaded}})

	require.NoError(t, err)
	_, err = registry.GetToolDef("existing")
	assert.NoError(t, err)
	_, err = registry.GetToolDef("loaded")
	assert.NoError(t, err)

	loadErr := errors.New("load failed")
	err = registry.Load(stubToolsLoader{err: loadErr})
	require.Error(t, err)
	assert.ErrorIs(t, err, loadErr)
	assert.ErrorContains(t, err, "cannot load tools")
}

func TestToolRegistrySetCallback(t *testing.T) {
	t.Run("sets a callback on an existing definition", func(t *testing.T) {
		registry := NewEmptyRegistry()
		registry.SetToolDefs(callbackDefinition("callback", true, true, nil))
		callback := func(context.Context, map[string]any) (string, error) {
			return "called", nil
		}

		require.NoError(t, registry.SetCallback("callback", callback))
		result, err := registry.Run(t.Context(), types.ToolCall{Function: "callback"})

		require.NoError(t, err)
		assert.Equal(t, "called", result)
	})

	t.Run("reports a missing definition", func(t *testing.T) {
		err := NewEmptyRegistry().SetCallback("missing", func(context.Context, map[string]any) (string, error) {
			return "", nil
		})

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrNotFound)
		assert.ErrorContains(t, err, "missing")
	})

	t.Run("rejects a callback for a command definition", func(t *testing.T) {
		registry := NewEmptyRegistry()
		registry.SetToolDefs(types.ToolDefinition{
			Tool:    types.Tool{Name: "command"},
			Command: &types.ToolCommand{Command: []string{"echo"}},
		})

		err := registry.SetCallback("command", func(context.Context, map[string]any) (string, error) {
			return "", nil
		})

		require.Error(t, err)
		assert.ErrorContains(t, err, "does not take a callback")
	})
}

func TestToolRegistryRun(t *testing.T) {
	t.Run("runs a registered tool", func(t *testing.T) {
		registry := NewEmptyRegistry()
		registry.SetToolDefs(callbackDefinition("callback", true, true, func(context.Context, map[string]any) (string, error) {
			return "result", nil
		}))

		result, err := registry.Run(t.Context(), types.ToolCall{Function: "callback"})

		require.NoError(t, err)
		assert.Equal(t, "result", result)
	})

	t.Run("wraps a missing tool error", func(t *testing.T) {
		result, err := NewEmptyRegistry().Run(t.Context(), types.ToolCall{Function: "missing"})

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrNotFound)
		assert.ErrorContains(t, err, "cannot run tool")
		assert.Empty(t, result)
	})
}

func callbackDefinition(name string, enabled, trusted bool, callback types.ToolCallback) types.ToolDefinition {
	return types.ToolDefinition{
		Enabled:  enabled,
		Trusted:  trusted,
		Tool:     types.Tool{Name: name, Description: name + " description"},
		Callback: callback,
	}
}
