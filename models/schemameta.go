package models

type SchemaMeta struct {
	Name        string
	Description string
}

type SchemaMarshaller interface {
	SchemaMeta() SchemaMeta
}

type LlmUsage struct {
	Model            string
	PromptTokens     int
	CompletionTokens int
	CachedTokens     int
}
