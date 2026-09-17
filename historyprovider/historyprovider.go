package historyprovider

import (
	"time"

	"github.com/c00/harnesser/models"
)

type HistoryProvider interface {
	// List entry keys, newest first.
	List() ([]string, error)
	// Set the current history to this key and load it into memory.
	Select(key string) (HistoryEntry, error)
	// Saves the entry.
	Save(entry HistoryEntry) error
}

type HistoryEntry struct {
	Messages models.Messages
	Key      string
	Updated  time.Time
}
