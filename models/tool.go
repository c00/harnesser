package models

import (
	"encoding/json"
	"fmt"
)

const (
	ToolCallDecisionApprove    ToolCallDecision = "approve"
	ToolCallDecisionReject     ToolCallDecision = "reject"
	ToolCallDecisionNoDecision ToolCallDecision = "no-decision"
)

type ToolCallDecision string

// ToolDefinition is a Tool plus its information on how to execute it
type ToolDefinition struct {
	// Enabled determines whether the tool will be available for the LLM
	Enabled bool `yaml:"enabled"`
	// Trusted: true will automatically run the command if it invoked. If false, will ask the user for confirmation
	Trusted bool `yaml:"trusted"`
	Tool    Tool `yaml:"tool"`
	// Command specifies the command and arguments
	Command []string `yaml:"command"`
	// ArgsFrom specified the parameter where the args are set. Should point to a string array
	ArgsFrom string `yaml:"argsFrom"`
	// AllowedEnv are the environment variables that will be forwarded to the command
	AllowedEnv []string `yaml:"allowedEnv"`
}

// Tool defines a function the LLM can invoke.
type Tool struct {
	Name        string         `json:"name" yaml:"name"`
	Description string         `json:"description" yaml:"description"`
	Parameters  map[string]any `json:"parameters" yaml:"parameters"` // JSON Schema
}

type Tools []Tool

// ToolCall represents a request from the LLM to execute a function.
type ToolCall struct {
	// The ID given by the LLM
	ToolCallID string           `json:"tool_call_id" yaml:"tool_call_id"`
	Function   string           `json:"function" yaml:"function"`
	Args       string           `json:"args" yaml:"args"` // JSON string
	Decision   ToolCallDecision `json:"-" yaml:"decision"`
}

func (ts Tools) List() []string {
	parts := []string{}
	for _, t := range ts {
		parts = append(parts, t.Name)
	}

	return parts
}

// PrettyArgs returns an indented version of the Args JSON string.
func (tc ToolCall) PrettyArgs() string {
	var obj any
	if err := json.Unmarshal([]byte(tc.Args), &obj); err != nil {
		return tc.Args
	}
	pretty, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return tc.Args
	}
	return string(pretty)
}

func (tc ToolCall) String() string {
	argString := "no arguments"
	if tc.Args != "" && tc.Args != "{}" {
		argString = "with arguments"
	}
	return fmt.Sprintf("%v (%v): %v", tc.Function, tc.ToolCallID, argString)
}
