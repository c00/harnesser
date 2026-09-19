package toolset

import (
	"context"
	"fmt"
	"sync"

	"github.com/c00/harnesser/types"
)

type ToolRegistry struct {
	tools    map[string]types.ToolDefinition
	toolsMut sync.RWMutex
}

func NewEmptyRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools: map[string]types.ToolDefinition{},
	}
}

func NewRegistry(loader ToolsLoader) (*ToolRegistry, error) {
	reg := &ToolRegistry{
		tools: map[string]types.ToolDefinition{},
	}

	if loader != nil {
		err := reg.Load(loader)
		if err != nil {
			return nil, fmt.Errorf("cannot load tools: %w", err)
		}
	}

	return reg, nil
}

func (r *ToolRegistry) Clear() {
	r.toolsMut.Lock()
	defer r.toolsMut.Unlock()

	r.tools = map[string]types.ToolDefinition{}
}

// Load will add all the tools from the loader to the registry.
// It does not clear the registry first.
func (r *ToolRegistry) Load(p ToolsLoader) error {
	defs, err := p.Load()
	if err != nil {
		return fmt.Errorf("cannot load tools: %w", err)
	}

	r.SetToolDefs(defs...)

	return nil
}

// Set callback for an already existing tool definition
// Useful for padding out defs loaded through yaml with predefined callbacks.
func (r *ToolRegistry) SetCallback(name string, cb types.ToolCallback) error {
	r.toolsMut.Lock()
	defer r.toolsMut.Unlock()

	def, ok := r.tools[name]
	if !ok {
		return fmt.Errorf("%w: %v", ErrNotFound, name)
	}

	if def.Command != nil {
		return fmt.Errorf("tool '%v' does not take a callback", name)
	}

	def.Callback = cb
	r.tools[name] = def

	return nil
}

// Add or update Tool Definitions
func (r *ToolRegistry) SetToolDefs(defs ...types.ToolDefinition) {
	r.toolsMut.Lock()
	defer r.toolsMut.Unlock()

	for _, def := range defs {
		r.tools[def.Tool.Name] = def
	}
}

// Get Tool Definition. Returns an error if not found
func (r *ToolRegistry) GetToolDef(name string) (types.ToolDefinition, error) {
	r.toolsMut.RLock()
	defer r.toolsMut.RUnlock()

	def, ok := r.tools[name]
	if !ok {
		return types.ToolDefinition{}, fmt.Errorf("%w: %v", ErrNotFound, name)
	}

	return def, nil
}

// Remove tool definition. Silently returns if the tool does not exist
func (r *ToolRegistry) RemoveToolDef(name string) {
	r.toolsMut.Lock()
	defer r.toolsMut.Unlock()

	delete(r.tools, name)
}

// Get all active tools as types.Tool
func (r *ToolRegistry) GetTools() types.Tools {
	r.toolsMut.RLock()
	defer r.toolsMut.RUnlock()

	var tools types.Tools
	for _, def := range r.tools {
		if def.Enabled {
			tools = append(tools, def.Tool)
		}
	}

	return tools
}

// Shorthand to execute a tool call
func (r *ToolRegistry) Run(ctx context.Context, tc types.ToolCall) (string, error) {
	td, err := r.GetToolDef(tc.Function)
	if err != nil {
		return "", fmt.Errorf("cannot run tool: %w", err)
	}

	return Run(ctx, td, tc)
}
