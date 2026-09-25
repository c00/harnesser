package types

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"time"

	gonanoid "github.com/matoous/go-nanoid/v2"
)

type Attachment struct {
	// Unique stable ID
	ID       string
	MimeType string
	// OriginalFilename for reference
	OriginalFilename string
	// Raw bytes of the file
	Data        []byte
	Title       string
	Description string
	CreatedAt   time.Time
}

// EnsureID sets an ID if none exists yet
func (a *Attachment) EnsureID() {
	if a.ID != "" {
		return
	}

	id, err := gonanoid.New(12)
	if err != nil {
		id := rand.Text()
		a.ID = id
	} else {
		a.ID = id
	}
}

func (a Attachment) IsImage() bool {
	return strings.HasPrefix(a.MimeType, "image/")
}

func (a Attachment) IsVideo() bool {
	return strings.HasPrefix(a.MimeType, "video/")
}

// Loads the filename from disk and sets data. Also sets the MimeType.
func (a *Attachment) Load(path string) error {
	if a.ID == "" {
		return fmt.Errorf("cannot load attachment: filename is empty")
	}

	fullName := filepath.Join(path, a.Filename())

	data, err := os.ReadFile(fullName)
	if err != nil {
		return fmt.Errorf("cannot read attachment %q: %w", fullName, err)
	}

	a.Data = data

	return nil
}

func (a *Attachment) Filename() string {
	a.EnsureID()
	return fmt.Sprintf("%v%v", a.ID, getExtension(a.MimeType))
}

// Loads the filename from disk and sets data. Also sets the MimeType.
func (a *Attachment) Save(path string) error {
	a.EnsureID()

	fullName := filepath.Join(path, a.Filename())

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

func getExtension(mimetype string) string {
	ext := ".bin"

	// Prefer common extensions over obscure ones (like .jfif for image/jpeg)
	preferredExtensions := map[string]string{
		"image/jpeg": ".jpg",
		"image/png":  ".png",
		"image/gif":  ".gif",
		"text/plain": ".txt",
	}
	if preferred, ok := preferredExtensions[mimetype]; ok {
		ext = preferred
	} else {
		exts, err := mime.ExtensionsByType(mimetype)
		if err != nil || len(exts) == 0 {
			ext = ".bin"
		} else {
			ext = exts[0]
		}
	}
	return ext
}
