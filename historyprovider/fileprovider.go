package historyprovider

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/c00/harnesser/models"
	"go.yaml.in/yaml/v4"
)

var _ HistoryProvider = (*FileProvider)(nil)

type FileProvider struct {
	currentKey   string
	path         string
	currentEntry HistoryEntry
}

// NewFileProvider creates a file provider and attempts to load the prompts
func NewFileProvider(path string) *FileProvider {
	return &FileProvider{
		path: path,
	}

}

// List entries newest first.
func (p *FileProvider) List() ([]string, error) {
	if err := os.MkdirAll(p.path, 0o755); err != nil {
		return nil, fmt.Errorf("cannot create directory '%v': %w", p.path, err)
	}

	entries, err := os.ReadDir(p.path)
	if err != nil {
		return nil, fmt.Errorf("cannot read dir: %w", err)
	}

	files := []HistoryEntry{}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext != ".yaml" && ext != ".yml" {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			return nil, fmt.Errorf("cannot read file info for %q: %w", entry.Name(), err)
		}

		files = append(files, HistoryEntry{
			Key:     entry.Name(),
			Updated: info.ModTime(),
		})
	}

	sort.SliceStable(files, func(i, j int) bool {
		return files[i].Updated.After(files[j].Updated)
	})

	filenames := make([]string, 0, len(files))
	for _, file := range files {
		filenames = append(filenames, file.Key)
	}

	return filenames, nil
}

func (p *FileProvider) Get() HistoryEntry {
	return p.currentEntry
}

// Set the current history to this key.
func (p *FileProvider) Select(key string) (HistoryEntry, error) {
	p.currentKey = key

	err := p.load()
	if err != nil {
		return HistoryEntry{Key: key}, fmt.Errorf("cannot load file: %w", err)
	}

	return p.currentEntry, nil
}

// Saves the entry to the selected key.
func (p *FileProvider) Save(entry HistoryEntry) error {
	if entry.Key != p.currentKey {
		return fmt.Errorf("history key is not the same")
	}

	if err := os.MkdirAll(p.path, 0o755); err != nil {
		return fmt.Errorf("cannot create directory '%v': %w", p.path, err)
	}

	data, err := yaml.Marshal(entry.Messages)
	if err != nil {
		return fmt.Errorf("cannot encode history: %w", err)
	}

	path := filepath.Join(p.path, p.currentKey)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("cannot write history %q: %w", path, err)
	}

	p.currentEntry = entry
	return nil
}

// Load will load the current entry into memory
func (p *FileProvider) load() error {
	p.currentEntry = HistoryEntry{Key: p.currentKey, Messages: models.Messages{}}

	path := filepath.Join(p.path, p.currentKey)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("cannot read history %q: %w", path, err)
	}

	if err := yaml.Unmarshal(data, &p.currentEntry.Messages); err != nil {
		return fmt.Errorf("cannot parse history %q: %w", path, err)
	}

	return nil
}
