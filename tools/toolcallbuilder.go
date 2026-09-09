package tools

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/c00/harnesser/interpolator"
	"github.com/c00/harnesser/models"
)

// ToolCallBuilder uses the definition and toolcall to create the command and its arguments.
type ToolCallBuilder struct {
	toolCallDef models.ToolDefinition
	toolCall    models.ToolCall

	commandName      string
	interpolatedArgs []string
	extraArgs        []string
}

func NewToolCallBuilder(tc models.ToolCall, td models.ToolDefinition) (*ToolCallBuilder, error) {
	b := &ToolCallBuilder{
		toolCallDef: td,
		toolCall:    tc,
	}
	err := b.build()
	if err != nil {
		return nil, err
	}
	return b, nil
}

// Return the command as a string. For logging and validation purposes.
func (b *ToolCallBuilder) CommandString() string {
	return fmt.Sprintf("%v %v %v", b.commandName, strings.Join(b.interpolatedArgs, " "), strings.Join(b.extraArgs, " "))
}

func (b *ToolCallBuilder) CommandName() string {
	return b.commandName
}

func (b *ToolCallBuilder) CommandArgs() []string {
	allArgs := append([]string{}, b.interpolatedArgs...)
	allArgs = append(allArgs, b.extraArgs...)

	return allArgs
}

func (b *ToolCallBuilder) build() error {
	if b.extraArgs != nil && b.interpolatedArgs != nil {
		// Already built.
		return nil
	}

	if len(b.toolCallDef.Command) == 0 {
		return errors.New("tool call does not have a command")
	}

	args, data, err := interpolator.InterpolatedCommand(b.toolCall, b.toolCallDef)
	if err != nil {
		return fmt.Errorf("cannot interpolate command: %w", err)
	}

	args = slices.DeleteFunc(args, func(s string) bool {
		return s == ""
	})

	b.commandName = args[0]
	b.interpolatedArgs = args[1:]

	if b.toolCallDef.ArgsFrom != "" {
		extraArgs := getStringSlice(data.Params, b.toolCallDef.ArgsFrom)
		extraArgs = slices.DeleteFunc(extraArgs, func(s string) bool {
			return s == ""
		})

		b.extraArgs = extraArgs
	} else {
		b.extraArgs = []string{}
	}

	return nil
}

func getStringSlice(params map[string]any, key string) []string {
	extraArgs, ok := params[key]
	if !ok {
		return []string{}
	}
	// if extra args is of type []string, return that
	if paramSlice, ok := extraArgs.([]string); ok {
		return paramSlice
	}

	// if extra args is of type []any, then iterate through that, and add every string in it to a slice and return that slice.
	if anySlice, ok := extraArgs.([]any); ok {
		result := []string{}
		for _, item := range anySlice {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	}

	return []string{}
}
