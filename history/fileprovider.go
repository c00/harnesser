package history

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"

	"github.com/c00/harnesser/types"
	"go.yaml.in/yaml/v4"
)

const messagesYaml = "messages.yaml"
const attachmentFolder = "attachments"

var _ HistoryProvider = (*FileProvider)(nil)

// FileProvider stores history in a folder rather than a singular file
type FileProvider struct {
	currentKey   string
	path         string
	currentEntry HistoryEntry
	logger       *slog.Logger
}

// NewFileProvider creates a file provider and attempts to load the prompts
func NewFileProvider(path string) *FileProvider {
	return &FileProvider{
		path:   path,
		logger: slog.Default().With("provider", "FileProvider2", "module", "history"),
	}

}

// List entries newest first.
func (p *FileProvider) List() ([]string, error) {
	if err := os.MkdirAll(p.path, 0o755); err != nil {
		return nil, fmt.Errorf("cannot create directory '%v': %w", p.path, err)
	}

	folders, err := os.ReadDir(p.path)
	if err != nil {
		return nil, fmt.Errorf("cannot read dir: %w", err)
	}

	entries := []HistoryEntry{}

	for _, entry := range folders {
		if !entry.IsDir() {
			p.logger.Warn("Skipping file in history dir", "file", entry.Name())
			continue
		}

		// find messages.yaml
		messagesYaml := filepath.Join(p.path, entry.Name(), messagesYaml)

		info, err := os.Stat(messagesYaml)
		if err != nil {
			p.logger.Warn("no messages.yaml found for history folder", "folder", entry.Name())
			continue
		}

		entries = append(entries, HistoryEntry{
			Key:     entry.Name(),
			Updated: info.ModTime(),
		})
	}

	sort.SliceStable(entries, func(i, j int) bool {
		return entries[i].Updated.After(entries[j].Updated)
	})

	entryKeys := make([]string, 0, len(entries))
	for _, entry := range entries {
		entryKeys = append(entryKeys, entry.Key)
	}

	return entryKeys, nil
}

func (p *FileProvider) Get() HistoryEntry {
	return p.currentEntry
}

// Set the current history to this key.
func (p *FileProvider) Select(key string) (HistoryEntry, error) {
	if err := validateHistoryKey(key); err != nil {
		return HistoryEntry{Key: key}, fmt.Errorf("invalid history key %v: %w", key, err)
	}

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
	if err := validateHistoryKey(entry.Key); err != nil {
		return fmt.Errorf("invalid history key %v: %w", entry.Key, err)
	}

	fullPath := filepath.Join(p.path, entry.Key, attachmentFolder)
	if err := os.MkdirAll(fullPath, 0o755); err != nil {
		return fmt.Errorf("cannot create directory '%v': %w", p.path, err)
	}

	msgs := make(types.Messages, 0, len(entry.Messages))

	// Save attachments separately
	for _, original := range entry.Messages {
		m := original
		m.Content = append(types.MessageParts(nil), original.Content...)

		for cIdx, part := range m.Content {
			if part.Attachment == nil {
				continue
			}

			part.Attachment.EnsureID()

			// Save the attachment as a separate file
			attCopy := *part.Attachment
			attachmentPath := filepath.Join(p.path, p.currentKey, attachmentFolder)

			err := attCopy.Save(attachmentPath)
			if err != nil {
				return fmt.Errorf("cannot write attachment to file: %w", err)
			}

			attCopy.Data = nil
			part.Attachment = &attCopy
			m.Content[cIdx] = part
		}

		msgs = append(msgs, m)
	}

	data, err := yaml.Marshal(msgs)
	if err != nil {
		return fmt.Errorf("cannot encode history: %w", err)
	}

	path := filepath.Join(p.path, p.currentKey, messagesYaml)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("cannot write history %q: %w", path, err)
	}

	p.currentEntry = entry
	return nil
}

// Load will load the current entry into memory
func (p *FileProvider) load() error {
	p.currentEntry = HistoryEntry{Key: p.currentKey, Messages: types.Messages{}}

	path := filepath.Join(p.path, p.currentKey, messagesYaml)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("cannot read messages %q: %w", path, err)
	}

	if err := yaml.Unmarshal(data, &p.currentEntry.Messages); err != nil {
		return fmt.Errorf("cannot parse history %q: %w", path, err)
	}

	// Load attachments
	attachmentPath := filepath.Join(p.path, p.currentKey, attachmentFolder)
	for mIdx, m := range p.currentEntry.Messages {
		anyAttachments := false

		for cIdx, part := range m.Content {
			if part.Attachment == nil {
				continue
			}

			err := part.Attachment.Load(attachmentPath)
			if err != nil {
				return fmt.Errorf("cannot load attachment: %w", err)
			}

			anyAttachments = true
			m.Content[cIdx] = part
		}

		if anyAttachments {
			p.currentEntry.Messages[mIdx] = m
		}
	}

	return nil
}

func validateHistoryKey(key string) error {
	if key == "." || !filepath.IsLocal(key) || filepath.Base(key) != key {
		return fmt.Errorf("invalid history key %q", key)
	}

	return nil
}
