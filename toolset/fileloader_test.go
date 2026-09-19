package toolset

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/c00/harnesser/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileLoaderLoad(t *testing.T) {
	t.Run("creates a missing directory", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "tools")

		defs, err := NewFileLoader(path).Load()

		require.NoError(t, err)
		assert.Empty(t, defs)
		info, err := os.Stat(path)
		require.NoError(t, err)
		assert.True(t, info.IsDir())
	})

	t.Run("loads YAML files and skips other entries", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "first.yaml"), []byte(`
enabled: true
trusted: false
tool:
  name: first
  description: first tool
  parameters:
    type: object
command:
  command: [echo, first]
`), 0o600))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "second.YML"), []byte(`
enabled: false
trusted: true
tool:
  name: second
  description: second tool
command:
  command: [echo, second]
  argsFrom: values
  allowedEnv: [PATH]
`), 0o600))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "ignored.txt"), []byte("ignored"), 0o600))
		require.NoError(t, os.Mkdir(filepath.Join(dir, "nested.yaml"), 0o700))

		defs, err := NewFileLoader(dir).Load()

		require.NoError(t, err)
		require.Len(t, defs, 2)
		assert.Equal(t, types.ToolDefinition{
			Enabled: true,
			Tool: types.Tool{
				Name:        "first",
				Description: "first tool",
				Parameters:  map[string]any{"type": "object"},
			},
			Command: &types.ToolCommand{Command: []string{"echo", "first"}},
		}, defs[0])
		assert.Equal(t, types.ToolDefinition{
			Trusted: true,
			Tool: types.Tool{
				Name:        "second",
				Description: "second tool",
			},
			Command: &types.ToolCommand{
				Command:    []string{"echo", "second"},
				ArgsFrom:   "values",
				AllowedEnv: []string{"PATH"},
			},
		}, defs[1])
	})

	t.Run("returns an error for invalid YAML", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "broken.yaml")
		require.NoError(t, os.WriteFile(path, []byte("tool: ["), 0o600))

		defs, err := NewFileLoader(dir).Load()

		require.Error(t, err)
		assert.ErrorContains(t, err, "cannot unmarshall tool")
		assert.ErrorContains(t, err, path)
		assert.Nil(t, defs)
	})

	t.Run("returns an error when a tool cannot be read", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "missing.yaml")
		require.NoError(t, os.Symlink(filepath.Join(dir, "missing-target"), path))

		defs, err := NewFileLoader(dir).Load()

		require.Error(t, err)
		assert.ErrorContains(t, err, "cannot read tool")
		assert.ErrorContains(t, err, path)
		assert.Nil(t, defs)
	})

	t.Run("returns an error when the directory cannot be created", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "file")
		require.NoError(t, os.WriteFile(path, nil, 0o600))

		defs, err := NewFileLoader(path).Load()

		require.Error(t, err)
		assert.ErrorContains(t, err, "cannot create directory")
		assert.Nil(t, defs)
	})
}
