package historyprovider

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/c00/harnesser/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

func TestFileProviderList(t *testing.T) {
	t.Run("creates a missing directory", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "history")
		provider := &FileProvider{path: path}

		entries, err := provider.List()

		require.NoError(t, err)
		assert.Empty(t, entries)
		info, err := os.Stat(path)
		require.NoError(t, err)
		assert.True(t, info.IsDir())
	})

	t.Run("returns YAML files newest first", func(t *testing.T) {
		dir := t.TempDir()
		oldPath := filepath.Join(dir, "old.yaml")
		newPath := filepath.Join(dir, "new.YML")
		require.NoError(t, os.WriteFile(oldPath, []byte("[]"), 0o644))
		require.NoError(t, os.WriteFile(newPath, []byte("[]"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "ignored.txt"), []byte("ignored"), 0o644))
		require.NoError(t, os.Mkdir(filepath.Join(dir, "nested.yaml"), 0o755))

		oldTime := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
		newTime := oldTime.Add(time.Hour)
		require.NoError(t, os.Chtimes(oldPath, oldTime, oldTime))
		require.NoError(t, os.Chtimes(newPath, newTime, newTime))

		provider := &FileProvider{path: dir}
		entries, err := provider.List()

		require.NoError(t, err)
		assert.Equal(t, []string{"new.YML", "old.yaml"}, entries)
	})
}

func TestFileProviderSelect(t *testing.T) {
	t.Run("loads messages from the selected YAML file", func(t *testing.T) {
		dir := t.TempDir()
		messages := testMessages("question", "answer")
		data, err := yaml.Marshal(messages)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(dir, "chat.yaml"), data, 0o644))
		provider := &FileProvider{path: dir}

		entry, err := provider.Select("chat.yaml")

		require.NoError(t, err)
		assert.Equal(t, "chat.yaml", entry.Key)
		assert.Equal(t, messages, entry.Messages)
	})

	t.Run("returns an error for invalid YAML", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "broken.yaml"), []byte("- ["), 0o644))
		provider := &FileProvider{path: dir}

		entry, err := provider.Select("broken.yaml")

		require.Error(t, err)
		assert.ErrorContains(t, err, "cannot load file")
		assert.ErrorContains(t, err, "cannot parse history")
		assert.Equal(t, HistoryEntry{Key: "broken.yaml"}, entry)
	})

	t.Run("returns an empty entry when the selected file is missing", func(t *testing.T) {
		provider := &FileProvider{path: t.TempDir()}

		entry, err := provider.Select("missing.yaml")

		require.NoError(t, err)
		assert.Equal(t, HistoryEntry{Key: "missing.yaml", Messages: models.Messages{}}, entry)
	})
}

func TestFileProviderSave(t *testing.T) {
	t.Run("writes messages to the selected file", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), "history")
		provider := &FileProvider{
			currentKey: "chat.yaml",
			path:       dir,
		}
		entry := HistoryEntry{
			Key: "chat.yaml",
			Messages: testMessages("hello", "hi"),
		}

		require.NoError(t, provider.Save(entry))
		assert.Equal(t, entry, provider.currentEntry)

		data, err := os.ReadFile(filepath.Join(dir, "chat.yaml"))
		require.NoError(t, err)
		var saved models.Messages
		require.NoError(t, yaml.Unmarshal(data, &saved))
		assert.Equal(t, entry.Messages, saved)
	})

	t.Run("rejects an entry with a different key", func(t *testing.T) {
		dir := t.TempDir()
		provider := &FileProvider{
			currentKey: "selected.yaml",
			path:       dir,
		}

		err := provider.Save(HistoryEntry{Key: "other.yaml"})

		require.Error(t, err)
		assert.ErrorContains(t, err, "history key is not the same")
		_, statErr := os.Stat(filepath.Join(dir, "other.yaml"))
		assert.ErrorIs(t, statErr, os.ErrNotExist)
	})
}

func testMessages(user, assistant string) models.Messages {
	createdAt := time.Date(2025, time.January, 1, 12, 0, 0, 0, time.UTC)
	return models.Messages{
		{
			Role:      models.RoleUser,
			Content:   models.MessageParts{{Type: models.PartText, Text: user}},
			ToolCalls: []models.ToolCall{},
			CreatedAt: createdAt,
			Reasoning: models.ReasoningParts{},
		},
		{
			Role:      models.RoleAssistant,
			Content:   models.MessageParts{{Type: models.PartText, Text: assistant}},
			ToolCalls: []models.ToolCall{},
			CreatedAt: createdAt,
			Reasoning: models.ReasoningParts{},
		},
	}
}
