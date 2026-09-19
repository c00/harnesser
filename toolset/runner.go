package toolset

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
	"time"

	"github.com/c00/harnesser/toolcallbuilder"
	"github.com/c00/harnesser/types"
)

var ErrNoDecision = errors.New("no decision for tool call")
var ErrRejected = errors.New("tool call was rejected by user")

// Run a tool def and tool call
func Run(ctx context.Context, td types.ToolDefinition, tc types.ToolCall) (string, error) {
	// Can we run this tool?
	if !td.Trusted {
		switch tc.Decision {
		case "":
			return "", fmt.Errorf("tool %v, function %v: %w", tc.ToolCallID, tc.Function, ErrNoDecision)
		case types.ToolCallDecisionNoDecision:
			return "", fmt.Errorf("tool %v, function %v: %w", tc.ToolCallID, tc.Function, ErrNoDecision)
		case types.ToolCallDecisionReject:
			return "", fmt.Errorf("tool %v, function %v: %w", tc.ToolCallID, tc.Function, ErrRejected)
		}
	}

	if td.Command != nil && td.Callback != nil {
		return "", fmt.Errorf("malformed tooldefinition: both callback and command are set. Should be either, not both")
	}
	if td.Command == nil && td.Callback == nil {
		return "", fmt.Errorf("malformed tooldefinition: neither callback nor command are set")
	}

	if td.Command != nil {
		return runCmd(ctx, td, tc)
	}

	// Run callback instead
	// json marshall args to map[string]any
	params := map[string]any{}
	if tc.Args != "" {
		err := json.Unmarshal([]byte(tc.Args), params)
		if err != nil {
			return "", fmt.Errorf("cannot unmarshall params: %w", err)
		}
	}
	return td.Callback(ctx, params)

}

func runCmd(ctx context.Context, td types.ToolDefinition, tc types.ToolCall) (string, error) {
	// build context
	// Give it a cancelable context, with a timeout
	execCtx, cancel := context.WithTimeout(ctx, time.Second*20)
	defer cancel()

	builder, err := toolcallbuilder.NewToolCallCmdBuilder(tc, td)
	if err != nil {
		return "", fmt.Errorf("cannot create tool call builder: %w", err)
	}

	// TODO limit env vars
	cmd := exec.CommandContext(execCtx, builder.CommandName(), builder.CommandArgs()...)

	slog.Debug("Executing command", "command", fmt.Sprintf("%v", builder.CommandString()))

	var stderr strings.Builder
	cmd.Stderr = &stderr

	data, err := cmd.Output()
	if err != nil {
		// If the process finishes with a non-zero exit code, return an error
		stderrText := strings.TrimSpace(stderr.String())
		if stderrText != "" {
			return "", fmt.Errorf("running tool '%v' failed: %w: %s", td.Tool.Name, err, stderrText)
		}

		return "", fmt.Errorf("running tool '%v' failed: %w", td.Tool.Name, err)
	}

	return string(data), nil
}
