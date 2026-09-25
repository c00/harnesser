package types

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAttachmentLoad(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	attachment := Attachment{ID: "attachment-id", MimeType: "image/png", OriginalFilename: "image.png"}
	filename := filepath.Join(dir, attachment.Filename())
	require.NoError(t, os.WriteFile(filename, []byte("\x89PNG\r\n\x1a\n"), 0o600))

	require.NoError(t, attachment.Load(dir))
	assert.Equal(t, "image/png", attachment.MimeType)
	assert.Equal(t, []byte("\x89PNG\r\n\x1a\n"), attachment.Data)
}

func TestAttachmentEnsureID(t *testing.T) {
	t.Parallel()

	attachment := Attachment{}
	attachment.EnsureID()

	require.NotEmpty(t, attachment.ID)
	id := attachment.ID
	attachment.EnsureID()
	assert.Equal(t, id, attachment.ID)
}

func TestAttachmentSave(t *testing.T) {
	t.Parallel()

	t.Run("writes data and creates parent directories", func(t *testing.T) {
		t.Parallel()

		dir := filepath.Join(t.TempDir(), "nested", "directory")
		attachment := Attachment{ID: "attachment-id", MimeType: "image/png", OriginalFilename: "image.png", Data: []byte("image")}

		require.NoError(t, attachment.Save(dir))

		got, err := os.ReadFile(filepath.Join(dir, "attachment-id.png"))
		require.NoError(t, err)
		assert.Equal(t, attachment.Data, got)
	})

	tests := []struct {
		name      string
		setup     func(t *testing.T) (Attachment, string)
		wantError string
	}{
		{
			name: "cannot create parent directory",
			setup: func(t *testing.T) (Attachment, string) {
				parent := filepath.Join(t.TempDir(), "parent")
				require.NoError(t, os.WriteFile(parent, []byte("file"), 0o600))
				return Attachment{ID: "attachment-id", MimeType: "image/png", OriginalFilename: "image.png", Data: []byte("image")}, parent
			},
			wantError: "cannot create directory:",
		},
		{
			name: "cannot write file",
			setup: func(t *testing.T) (Attachment, string) {
				dir := t.TempDir()
				require.NoError(t, os.Mkdir(filepath.Join(dir, "attachment-id.png"), 0o755))
				return Attachment{ID: "attachment-id", MimeType: "image/png", OriginalFilename: "image.png", Data: []byte("image")}, dir
			},
			wantError: "cannot write file:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			attachment, path := tt.setup(t)
			err := attachment.Save(path)

			require.Error(t, err)
			assert.ErrorContains(t, err, tt.wantError)
		})
	}
}

func TestAttachmentBase64(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		attachment Attachment
		want       string
		wantError  string
	}{
		{
			name:       "data URL",
			attachment: Attachment{MimeType: "image/png", Data: []byte("image")},
			want:       "data:image/png;base64,aW1hZ2U=",
		},
		{
			name:       "empty data",
			attachment: Attachment{MimeType: "image/png"},
			wantError:  "attachment data is empty",
		},
		{
			name:       "empty MIME type",
			attachment: Attachment{Data: []byte("image")},
			wantError:  "attachment MIME type is empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := tt.attachment.Base64()
			if tt.wantError != "" {
				require.EqualError(t, err, tt.wantError)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
