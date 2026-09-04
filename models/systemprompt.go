package models

type CacheMode string

const (
	CacheModeOff   CacheMode = "off"
	CacheModeShort CacheMode = "short" // 5 min ephemeral
	CacheModeLong  CacheMode = "long"  // 1 hour ephemeral
)

type SystemPrompt struct {
	Priority int    `json:"priority"`
	Label    string `json:"label"`
	Content  string `json:"content"`
}
