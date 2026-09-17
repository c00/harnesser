package toolsprovider

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/c00/harnesser/models"
	"go.yaml.in/yaml/v4"
)

var _ ToolsProvider = (*FileProvider)(nil)
var ErrNotFound = errors.New("tool not found")

type FileProvider struct {
	path     string
	toolDefs []models.ToolDefinition
}

func NewFileProvider(path string) (*FileProvider, error) {
	p := &FileProvider{
		path: path,
	}
	err := p.Load()
	if err != nil {
		return nil, fmt.Errorf("cannot load tools: %w", err)
	}
	return p, nil
}

func (p *FileProvider) GetTools() models.Tools {
	tools := models.Tools{}

	for _, def := range p.toolDefs {
		if def.Enabled {
			tools = append(tools, def.Tool)
		}
	}

	return tools
}

func (p *FileProvider) GetDefinitions() []models.ToolDefinition {
	return p.toolDefs
}

func (p *FileProvider) GetDefinition(name string) (models.ToolDefinition, error) {
	for _, def := range p.toolDefs {
		if def.Tool.Name == name {
			return def, nil
		}

	}

	return models.ToolDefinition{}, fmt.Errorf("%w: %v", ErrNotFound, name)
}

// Load will load the files from the directory. Will create the dir if it does not exist.
func (p *FileProvider) Load() error {
	p.toolDefs = []models.ToolDefinition{}

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
		if ext != ".yaml" && ext != ".yml" {
			continue
		}

		path := filepath.Join(p.path, entry.Name())
		tool, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("cannot read tool %q: %w", path, err)
		}

		def := models.ToolDefinition{}
		err = yaml.Unmarshal(tool, &def)
		if err != nil {
			return fmt.Errorf("cannot unmarshall tool %v: %w", path, err)
		}

		p.toolDefs = append(p.toolDefs, def)
	}

	return nil
}
