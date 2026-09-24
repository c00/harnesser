package history

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/c00/harnesser/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

func TestFileProvider2SaveAttachment(t *testing.T) {
	dir := t.TempDir()
	provider := NewFileProvider2(dir)
	_, err := provider.Select("chat")
	require.NoError(t, err)

	attachment := &types.Attachment{
		MimeType: "image/png",
		Filename: filepath.Join("uploads", "image.png"),
		Data:     []byte("\x89PNG\r\n\x1a\n"),
	}
	entry := HistoryEntry{
		Key: "chat",
		Messages: types.Messages{
			{
				Role: types.RoleUser,
				Content: types.MessageParts{
					{Type: types.PartImage, Attachment: attachment},
				},
			},
		},
	}

	require.NoError(t, provider.Save(entry))

	assert.Equal(t, filepath.Join("uploads", "image.png"), attachment.Filename)
	assert.Equal(t, []byte("\x89PNG\r\n\x1a\n"), attachment.Data)
	assert.Equal(t, entry, provider.Get())

	savedData, err := os.ReadFile(filepath.Join(dir, "chat", attachmentFolder, "image.png"))
	require.NoError(t, err)
	assert.Equal(t, attachment.Data, savedData)

	yamlData, err := os.ReadFile(filepath.Join(dir, "chat", messagesYaml))
	require.NoError(t, err)
	var stored types.Messages
	require.NoError(t, yaml.Unmarshal(yamlData, &stored))
	require.Len(t, stored, 1)
	require.Len(t, stored[0].Content, 1)
	require.NotNil(t, stored[0].Content[0].Attachment)
	assert.Equal(t, "image.png", stored[0].Content[0].Attachment.Filename)
	assert.Empty(t, stored[0].Content[0].Attachment.Data)

	loaded, err := provider.Select("chat")
	require.NoError(t, err)
	require.Len(t, loaded.Messages, 1)
	require.Len(t, loaded.Messages[0].Content, 1)
	require.NotNil(t, loaded.Messages[0].Content[0].Attachment)
	assert.Equal(t, "image.png", loaded.Messages[0].Content[0].Attachment.Filename)
	assert.Equal(t, attachment.Data, loaded.Messages[0].Content[0].Attachment.Data)
}

func TestFileProvider2RejectsInvalidHistoryKeys(t *testing.T) {
	tests := []string{
		"",
		".",
		"..",
		filepath.Join("..", "outside"),
		filepath.Join("nested", "chat"),
		filepath.Join(string(filepath.Separator), "absolute"),
	}

	for _, key := range tests {
		t.Run(key, func(t *testing.T) {
			provider := NewFileProvider2(t.TempDir())
			provider.currentKey = key

			_, selectErr := provider.Select(key)
			require.Error(t, selectErr)
			assert.ErrorContains(t, selectErr, "invalid history key")

			saveErr := provider.Save(HistoryEntry{Key: key})
			require.Error(t, saveErr)
			assert.ErrorContains(t, saveErr, "invalid history key")
		})
	}
}
