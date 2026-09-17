package toolsprovider

import (
	"fmt"

	"github.com/c00/harnesser/models"
)

var _ ToolsProvider = (*MemoryProvider)(nil)

type MemoryProvider struct {
	toolDefs []models.ToolDefinition
}

func NewMemoryProvider() *MemoryProvider {
	return &MemoryProvider{}
}

func (p *MemoryProvider) GetTools() models.Tools {
	tools := models.Tools{}

	for _, def := range p.toolDefs {
		if def.Enabled {
			tools = append(tools, def.Tool)
		}
	}

	return tools
}

func (p *MemoryProvider) GetDefinitions() []models.ToolDefinition {
	return p.toolDefs
}

func (p *MemoryProvider) GetDefinition(name string) (models.ToolDefinition, error) {
	for _, def := range p.toolDefs {
		if def.Tool.Name == name {
			return def, nil
		}

	}

	return models.ToolDefinition{}, fmt.Errorf("%w: %v", ErrNotFound, name)
}

func (p *MemoryProvider) SetDefinitions(def []models.ToolDefinition) {
	p.toolDefs = p.toolDefs
}

func (p *MemoryProvider) AddDefinition(def models.ToolDefinition) {
	if p.toolDefs == nil {
		p.toolDefs = []models.ToolDefinition{}
	}

	p.toolDefs = append(p.toolDefs, def)
}
