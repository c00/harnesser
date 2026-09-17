package promptsprovider

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/c00/harnesser/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileProviderLoad(t *testing.T) {
	t.Run("loads text prompts and skips other entries", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "a.md"), []byte("prompt a"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "b.TXT"), []byte("prompt b"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "ignored.yaml"), []byte("ignored"), 0o644))
		require.NoError(t, os.Mkdir(filepath.Join(dir, "nested.md"), 0o755))

		provider := &FileProvider{path: dir}
		require.NoError(t, provider.Load())

		prompts := provider.Prompts()
		require.Len(t, prompts, 2)
		assert.Equal(t, "a.md", prompts[0].Key)
		assert.True(t, prompts[0].Active)
		assert.Equal(t, models.RoleSystem, prompts[0].Message.Role)
		assert.Equal(t, "prompt a", prompts[0].Message.Content.String())
		assert.Equal(t, "b.TXT", prompts[1].Key)
		assert.True(t, prompts[1].Active)
		assert.Equal(t, models.RoleSystem, prompts[1].Message.Role)
		assert.Equal(t, "prompt b", prompts[1].Message.Content.String())
	})

	t.Run("replaces prompts when reloaded", func(t *testing.T) {
		dir := t.TempDir()
		firstPath := filepath.Join(dir, "first.md")
		require.NoError(t, os.WriteFile(firstPath, []byte("first prompt"), 0o644))

		provider := &FileProvider{path: dir}
		require.NoError(t, provider.Load())
		require.Len(t, provider.Prompts(), 1)

		require.NoError(t, os.Remove(firstPath))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "second.txt"), []byte("second prompt"), 0o644))
		require.NoError(t, provider.Load())

		require.Len(t, provider.Prompts(), 1)
		assert.Equal(t, "second.txt", provider.Prompts()[0].Key)
		assert.Equal(t, "second prompt", provider.Prompts()[0].Message.Content.String())
	})

	t.Run("fails when prompt cannot be read", func(t *testing.T) {
		dir := t.TempDir()
		promptPath := filepath.Join(dir, "broken.md")
		require.NoError(t, os.Symlink(filepath.Join(dir, "missing-target"), promptPath))
		provider := &FileProvider{path: dir}

		err := provider.Load()

		require.Error(t, err)
		assert.ErrorContains(t, err, "cannot read prompt")
		assert.ErrorContains(t, err, promptPath)
	})
}
