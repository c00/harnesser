package historyprovider

import (
	"testing"
	"time"

	"github.com/c00/harnesser/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryProviderList(t *testing.T) {
	tests := []struct {
		name    string
		entries []HistoryEntry
		want    []string
	}{
		{
			name: "empty provider",
			want: []string{},
		},
		{
			name: "entries newest first",
			entries: []HistoryEntry{
				{Key: "old", Updated: time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)},
				{Key: "new", Updated: time.Date(2025, time.January, 3, 0, 0, 0, 0, time.UTC)},
				{Key: "middle", Updated: time.Date(2025, time.January, 2, 0, 0, 0, 0, time.UTC)},
			},
			want: []string{"new", "middle", "old"},
		},
		{
			name: "equal timestamps retain insertion order",
			entries: []HistoryEntry{
				{Key: "first", Updated: time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)},
				{Key: "second", Updated: time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)},
			},
			want: []string{"first", "second"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := NewMemoryProvider()
			provider.entries = tt.entries

			keys, err := provider.List()

			require.NoError(t, err)
			assert.Equal(t, tt.want, keys)
		})
	}
}

func TestMemoryProviderSelect(t *testing.T) {
	t.Run("returns an empty entry for a new key", func(t *testing.T) {
		provider := NewMemoryProvider()

		entry, err := provider.Select("new")

		require.NoError(t, err)
		assert.Equal(t, HistoryEntry{Key: "new", Messages: models.Messages{}}, entry)
	})

	t.Run("loads a saved entry", func(t *testing.T) {
		provider := NewMemoryProvider()
		saved := HistoryEntry{
			Key:      "saved",
			Messages: testMessages("question", "answer"),
			Updated:  time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC),
		}
		require.NoError(t, selectAndSave(provider, saved))
		_, err := provider.Select("other")
		require.NoError(t, err)

		entry, err := provider.Select("saved")

		require.NoError(t, err)
		assert.Equal(t, saved, entry)
	})
}

func TestMemoryProviderSave(t *testing.T) {
	t.Run("adds the selected entry", func(t *testing.T) {
		provider := NewMemoryProvider()
		entry := HistoryEntry{Key: "chat", Messages: testMessages("hello", "hi")}
		_, err := provider.Select(entry.Key)
		require.NoError(t, err)

		require.NoError(t, provider.Save(entry))

		assert.Equal(t, entry, provider.currentEntry)
		assert.Equal(t, []HistoryEntry{entry}, provider.entries)
	})

	t.Run("replaces an existing entry", func(t *testing.T) {
		provider := NewMemoryProvider()
		original := HistoryEntry{Key: "chat", Messages: testMessages("old", "answer")}
		require.NoError(t, selectAndSave(provider, original))
		updated := HistoryEntry{Key: "chat", Messages: testMessages("new", "answer")}

		require.NoError(t, provider.Save(updated))

		assert.Equal(t, updated, provider.currentEntry)
		assert.Equal(t, []HistoryEntry{updated}, provider.entries)
	})

	t.Run("rejects an entry with a different key", func(t *testing.T) {
		provider := NewMemoryProvider()
		_, err := provider.Select("selected")
		require.NoError(t, err)

		err = provider.Save(HistoryEntry{Key: "other"})

		require.Error(t, err)
		assert.ErrorContains(t, err, "history key is not the same")
		assert.Empty(t, provider.entries)
	})
}

func selectAndSave(provider *MemoryProvider, entry HistoryEntry) error {
	if _, err := provider.Select(entry.Key); err != nil {
		return err
	}

	return provider.Save(entry)
}
