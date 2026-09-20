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

	filename := filepath.Join(t.TempDir(), "image.png")
	require.NoError(t, os.WriteFile(filename, []byte("\x89PNG\r\n\x1a\n"), 0o600))
	attachment := Attachment{Filename: filename}

	require.NoError(t, attachment.Load())
	assert.Equal(t, "image/png", attachment.MimeType)
	assert.Equal(t, []byte("\x89PNG\r\n\x1a\n"), attachment.Data)
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
