package toolset

import (
	"fmt"

	"github.com/c00/harnesser/types"
)

var _ ToolsProvider = (*MemoryProvider)(nil)

type MemoryProvider struct {
	toolDefs []types.ToolDefinition
}

func NewMemoryProvider() *MemoryProvider {
	return &MemoryProvider{}
}

func (p *MemoryProvider) GetTools() types.Tools {
	tools := types.Tools{}

	for _, def := range p.toolDefs {
		if def.Enabled {
			tools = append(tools, def.Tool)
		}
	}

	return tools
}

func (p *MemoryProvider) GetDefinitions() []types.ToolDefinition {
	return p.toolDefs
}

func (p *MemoryProvider) GetDefinition(name string) (types.ToolDefinition, error) {
	for _, def := range p.toolDefs {
		if def.Tool.Name == name {
			return def, nil
		}

	}

	return types.ToolDefinition{}, fmt.Errorf("%w: %v", ErrNotFound, name)
}

func (p *MemoryProvider) SetDefinitions(def []types.ToolDefinition) {
	p.toolDefs = def
}

func (p *MemoryProvider) AddDefinition(def types.ToolDefinition) {
	if p.toolDefs == nil {
		p.toolDefs = []types.ToolDefinition{}
	}

	p.toolDefs = append(p.toolDefs, def)
}
