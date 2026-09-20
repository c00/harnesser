package types

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

type Attachment struct {
	MimeType string
	// Complete file name
	Filename string
	// Raw bytes of the file
	Data        []byte
	Title       string
	Description string
	CreatedAt   time.Time
}

func (a Attachment) IsImage() bool {
	return strings.HasPrefix(a.MimeType, "image/")
}

func (a Attachment) IsVideo() bool {
	return strings.HasPrefix(a.MimeType, "video/")
}

// Loads the filename from disk and sets data. Also sets the MimeType.
func (a *Attachment) Load() error {
	if a.Filename == "" {
		return fmt.Errorf("cannot load attachment: filename is empty")
	}

	data, err := os.ReadFile(a.Filename)
	if err != nil {
		return fmt.Errorf("cannot read attachment %q: %w", a.Filename, err)
	}
	a.Data = data
	a.MimeType = http.DetectContentType(data)

	return nil
}

// Base64 returns the attachment data as a base64 data URL.
func (a Attachment) Base64() (string, error) {
	if len(a.Data) == 0 {
		return "", fmt.Errorf("attachment data is empty")
	}
	if a.MimeType == "" {
		return "", fmt.Errorf("attachment MIME type is empty")
	}

	encoded := base64.StdEncoding.EncodeToString(a.Data)
	return fmt.Sprintf("data:%s;base64,%s", a.MimeType, encoded), nil
}
