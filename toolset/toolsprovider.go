package toolset

import "github.com/c00/harnesser/types"

type ToolsLoader interface {
	Load() ([]types.ToolDefinition, error)
}
