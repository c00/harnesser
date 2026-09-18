package history

import (
	"fmt"
	"sort"

	"github.com/c00/harnesser/types"
)

var _ HistoryProvider = (*MemoryProvider)(nil)

type MemoryProvider struct {
	currentKey   string
	currentEntry HistoryEntry
	entries      []HistoryEntry
}

func NewMemoryProvider() *MemoryProvider {
	return &MemoryProvider{entries: []HistoryEntry{}}
}

// List entries newest first.
func (p *MemoryProvider) List() ([]string, error) {
	sort.SliceStable(p.entries, func(i, j int) bool {
		return p.entries[i].Updated.After(p.entries[j].Updated)
	})

	keys := make([]string, 0, len(p.entries))
	for _, file := range p.entries {
		keys = append(keys, file.Key)
	}

	return keys, nil
}

func (p *MemoryProvider) Get() HistoryEntry {
	return p.currentEntry
}

// Set the current history to this key.
func (p *MemoryProvider) Select(key string) (HistoryEntry, error) {
	p.currentKey = key

	err := p.load()
	if err != nil {
		return HistoryEntry{Key: key}, fmt.Errorf("cannot load entry: %w", err)
	}

	return p.currentEntry, nil
}

// Saves the entry to the selected key.
func (p *MemoryProvider) Save(entry HistoryEntry) error {
	if entry.Key != p.currentKey {
		return fmt.Errorf("history key is not the same")
	}

	p.currentEntry = entry

	for idx, e := range p.entries {
		if e.Key == entry.Key {
			p.entries[idx] = entry
			return nil
		}
	}

	p.entries = append(p.entries, entry)

	return nil
}

// Load will load the current entry into memory
func (p *MemoryProvider) load() error {
	for _, e := range p.entries {
		if e.Key == p.currentKey {
			p.currentEntry = e
			return nil
		}
	}

	p.currentEntry = HistoryEntry{Key: p.currentKey, Messages: types.Messages{}}

	return nil
}
