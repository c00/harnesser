package toolcmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/c00/harnesser/cmd/internal/setup"
	"github.com/c00/harnesser/models"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:           "tool name [args_as_json]",
	Example:       `tool search_web '{"query": "The best cheese in Amsterdam"}'`,
	Short:         "run a single tool",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          run,
	Args:          cobra.MinimumNArgs(1),
}

func run(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	agent, err := setup.Setup(cmd)
	if err != nil {
		return fmt.Errorf("cannot setup agent: %w", err)
	}

	function := args[0]
	functionArgs := "{}"
	if len(args) > 1 {
		functionArgs = args[1]
	}

	msg := models.Message{
		CreatedAt: time.Now(),
		Role:      models.RoleAssistant,
		ToolCalls: []models.ToolCall{
			{ToolCallID: "tool_00001", Function: function, Args: functionArgs, Decision: models.ToolCallDecisionApprove},
		},
	}
	agent.AddMessage(msg)
	result, err := agent.RunTools(ctx)
	if err != nil {
		return fmt.Errorf("could not run tools: %w", err)
	}

	fmt.Println(msg.String())
	fmt.Println(result.String())

	return nil
}

// Load the tools from the toolDir.
func chooseLatest(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("cannot read dir: %w", err)
	}

	if len(entries) == 0 {
		return "", nil
	}

	lastEntry := entries[len(entries)-1]
	return filepath.Join(dir, lastEntry.Name()), nil
}
