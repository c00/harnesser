package systemprompts

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/c00/harnesser/types"
)

var _ PromptsReader = (*FileProvider)(nil)

// FileProvider implements PromptsReader
type FileProvider struct {
	path    string
	prompts []Prompt
}

// NewFileProvider creates a file provider and attempts to load the prompts
func NewFileProvider(path string) (*FileProvider, error) {
	p := &FileProvider{
		path: path,
	}
	err := p.Load()
	if err != nil {
		return nil, fmt.Errorf("cannot load prompts: %w", err)
	}
	return p, nil
}

func (p *FileProvider) Prompts() Prompts {
	return p.prompts
}

// Load will load the files from the directory. Will create the dir if it does not exist.
func (p *FileProvider) Load() error {
	p.prompts = []Prompt{}

	if err := os.MkdirAll(p.path, 0o755); err != nil {
		return fmt.Errorf("cannot create directory '%v': %w", p.path, err)
	}

	entries, err := os.ReadDir(p.path)
	if err != nil {
		return fmt.Errorf("cannot read dir: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext != ".md" && ext != ".txt" {
			continue
		}

		path := filepath.Join(p.path, entry.Name())
		prompt, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("cannot read prompt %q: %w", path, err)
		}
		p.prompts = append(p.prompts, Prompt{
			Active:  true,
			Key:     entry.Name(),
			Message: types.NewSystemMessage(string(prompt)),
		})
	}

	return nil
}
