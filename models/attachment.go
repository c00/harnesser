package models

import (
	"strings"
	"time"
)

type Attachment struct {
	MimeType string `json:"mime_type"` // "image", "audio"
	// Filename will be unique. May contain directory parts
	Filename    string    `json:"filename"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

func (a Attachment) IsImage() bool {
	return strings.HasPrefix(a.MimeType, "image/")
}

func (a Attachment) IsVideo() bool {
	return strings.HasPrefix(a.MimeType, "video/")
}
