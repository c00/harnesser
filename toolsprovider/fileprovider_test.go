package toolsprovider

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/c00/harnesser/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

func TestNewFileProvider(t *testing.T) {
	t.Run("creates a missing directory", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "tools")

		provider, err := NewFileProvider(path)

		require.NoError(t, err)
		assert.Empty(t, provider.GetDefinitions())
		info, err := os.Stat(path)
		require.NoError(t, err)
		assert.True(t, info.IsDir())
	})

	t.Run("wraps load errors", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "tools")
		require.NoError(t, os.WriteFile(path, []byte("not a directory"), 0o644))

		provider, err := NewFileProvider(path)

		require.Error(t, err)
		assert.Nil(t, provider)
		assert.ErrorContains(t, err, "cannot load tools")
		assert.ErrorContains(t, err, "cannot create directory")
	})
}

func TestFileProviderLoad(t *testing.T) {
	t.Run("loads YAML definitions and skips other entries", func(t *testing.T) {
		dir := t.TempDir()
		first := testToolDefinition("first", true)
		second := testToolDefinition("second", false)
		writeToolDefinition(t, filepath.Join(dir, "a.yaml"), first)
		writeToolDefinition(t, filepath.Join(dir, "b.YML"), second)
		require.NoError(t, os.WriteFile(filepath.Join(dir, "ignored.json"), []byte("{}"), 0o644))
		require.NoError(t, os.Mkdir(filepath.Join(dir, "nested.yaml"), 0o755))

		provider := &FileProvider{path: dir}
		require.NoError(t, provider.Load())

		assert.Equal(t, []models.ToolDefinition{first, second}, provider.GetDefinitions())
		assert.Equal(t, models.Tools{first.Tool}, provider.GetTools())
		definition, err := provider.GetDefinition("second")
		require.NoError(t, err)
		assert.Equal(t, second, definition)
	})

	t.Run("replaces definitions when reloaded", func(t *testing.T) {
		dir := t.TempDir()
		firstPath := filepath.Join(dir, "first.yaml")
		writeToolDefinition(t, firstPath, testToolDefinition("first", true))
		provider := &FileProvider{path: dir}
		require.NoError(t, provider.Load())
		require.Len(t, provider.GetDefinitions(), 1)

		require.NoError(t, os.Remove(firstPath))
		second := testToolDefinition("second", true)
		writeToolDefinition(t, filepath.Join(dir, "second.yml"), second)

		require.NoError(t, provider.Load())
		assert.Equal(t, []models.ToolDefinition{second}, provider.GetDefinitions())
	})

	t.Run("returns an error for invalid YAML", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "broken.yaml")
		require.NoError(t, os.WriteFile(path, []byte("tool: ["), 0o644))
		provider := &FileProvider{path: dir}

		err := provider.Load()

		require.Error(t, err)
		assert.ErrorContains(t, err, "cannot unmarshall tool")
		assert.ErrorContains(t, err, path)
	})

	t.Run("returns an error when a definition cannot be read", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "broken.yaml")
		require.NoError(t, os.Symlink(filepath.Join(dir, "missing-target"), path))
		provider := &FileProvider{path: dir}

		err := provider.Load()

		require.Error(t, err)
		assert.ErrorContains(t, err, "cannot read tool")
		assert.ErrorContains(t, err, path)
	})
}

func TestFileProviderGetDefinitionNotFound(t *testing.T) {
	provider := &FileProvider{}

	definition, err := provider.GetDefinition("missing")

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotFound)
	assert.ErrorContains(t, err, "missing")
	assert.Equal(t, models.ToolDefinition{}, definition)
}

func writeToolDefinition(t *testing.T, path string, definition models.ToolDefinition) {
	t.Helper()

	data, err := yaml.Marshal(definition)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, data, 0o644))
}
