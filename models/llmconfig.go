package models

type ReasoningLevel string

const (
	ReasoningLow    ReasoningLevel = "low"
	ReasoningMedium ReasoningLevel = "medium"
	ReasoningHigh   ReasoningLevel = "high"
)

type LlmConfig struct {
	Name            string
	Provider        string
	Models          []string
	MaxOutputTokens int
	Reasoning       ReasoningLevel
}
