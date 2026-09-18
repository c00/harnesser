package history

import (
	"fmt"
	"time"

	"github.com/c00/harnesser/types"
)

type HistoryProvider interface {
	// List entry keys, newest first.
	List() ([]string, error)
	// Set the current history to this key and load it into memory.
	Select(key string) (HistoryEntry, error)
	// Get the current history entry.
	Get() HistoryEntry
	// Saves the entry.
	Save(entry HistoryEntry) error
}

type HistoryEntry struct {
	Messages types.Messages
	Key      string
	Updated  time.Time
}

// Create a new key (filename) for a history entry. Defaults to a RFC3339 timestamp.
func NewKey() string {
	return fmt.Sprintf("%v.yaml", time.Now().Format(time.RFC3339))
}
