package toolset

import "github.com/c00/harnesser/types"

type ToolsProvider interface {
	GetTools() types.Tools
	GetDefinitions() []types.ToolDefinition
	GetDefinition(name string) (types.ToolDefinition, error)
}
