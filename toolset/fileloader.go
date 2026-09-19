package toolset

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/c00/harnesser/types"
	"go.yaml.in/yaml/v4"
)

var _ ToolsLoader = (*FileLoader)(nil)
var ErrNotFound = errors.New("tool not found")

type FileLoader struct {
	path string
}

func NewFileLoader(path string) *FileLoader {
	return &FileLoader{
		path: path,
	}
}

// Load will load the files from the directory. Will create the dir if it does not exist.
func (p *FileLoader) Load() ([]types.ToolDefinition, error) {
	toolDefs := []types.ToolDefinition{}

	if err := os.MkdirAll(p.path, 0o755); err != nil {
		return nil, fmt.Errorf("cannot create directory '%v': %w", p.path, err)
	}

	entries, err := os.ReadDir(p.path)
	if err != nil {
		return nil, fmt.Errorf("cannot read dir: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext != ".yaml" && ext != ".yml" {
			continue
		}

		path := filepath.Join(p.path, entry.Name())
		tool, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("cannot read tool %q: %w", path, err)
		}

		def := types.ToolDefinition{}
		err = yaml.Unmarshal(tool, &def)
		if err != nil {
			return nil, fmt.Errorf("cannot unmarshall tool %v: %w", path, err)
		}

		toolDefs = append(toolDefs, def)
	}

	return toolDefs, nil
}
