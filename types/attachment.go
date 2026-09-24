package types

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Attachment struct {
	MimeType string
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
func (a *Attachment) Load(path string) error {
	if a.Filename == "" {
		return fmt.Errorf("cannot load attachment: filename is empty")
	}

	fullName := filepath.Join(path, a.Filename)

	data, err := os.ReadFile(fullName)
	if err != nil {
		return fmt.Errorf("cannot read attachment %q: %w", a.Filename, err)
	}
	a.Data = data
	a.MimeType = http.DetectContentType(data)

	return nil
}

// CloneMetadata creates a copy of the Attachment without the data
func (a *Attachment) CloneMetadata() Attachment {
	return Attachment{
		MimeType:    a.MimeType,
		Filename:    a.Filename,
		Title:       a.Title,
		Description: a.Description,
		CreatedAt:   a.CreatedAt,
	}
}

// Loads the filename from disk and sets data. Also sets the MimeType.
func (a *Attachment) Save(path string) error {
	if a.Filename == "" {
		return fmt.Errorf("cannot save attachment: filename is empty")
	}

	fullName := filepath.Join(path, a.Filename)

	// Make dir all to ensure path exists
	err := os.MkdirAll(filepath.Dir(fullName), 0o755)
	if err != nil {
		return fmt.Errorf("cannot create directory: %w", err)
	}

	// Write file to path
	err = os.WriteFile(fullName, a.Data, 0o644)
	if err != nil {
		return fmt.Errorf("cannot write file: %w", err)
	}

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
