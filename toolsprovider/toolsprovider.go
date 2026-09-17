package toolsprovider

import "github.com/c00/harnesser/models"

type ToolsProvider interface {
	GetTools() models.Tools
	GetDefinitions() []models.ToolDefinition
	GetDefinition(name string) (models.ToolDefinition, error)
}
