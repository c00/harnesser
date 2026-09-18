package decidecmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/c00/harnesser/cmd/internal/setup"
	"github.com/c00/harnesser/types"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:           "decide toolCallID1 [toolCallID2 ...]",
	Aliases:       []string{"approve"},
	Short:         "aprove or reject a tool call",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          run,
	Args:          cobra.MinimumNArgs(1),
}

func init() {
	Cmd.Flags().StringP("thread", "t", "", "The history thread to load. Can be relative to the histdir, relative to the pwd or an absolute path")
	Cmd.Flags().BoolP("continue", "c", false, "Continue the last conversation")
	Cmd.Flags().BoolP("approve", "a", true, "approve tool calls")
}

func run(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	agent, err := setup.Setup(cmd)
	if err != nil {
		return fmt.Errorf("cannot setup agent: %w", err)
	}

	approve, _ := cmd.Flags().GetBool("approve")
	decision := types.ToolCallDecisionReject
	if approve {
		decision = types.ToolCallDecisionApprove
	}

	// Run
	for _, arg := range args {
		err = agent.ConfirmToolCall(ctx, arg, decision)
		if err != nil {
			return fmt.Errorf("cannot get response: %w", err)
		}
	}

	return nil
}

func chooseLatest(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}

	if len(entries) == 0 {
		return "", errors.New("no history files")
	}

	return entries[len(entries)-1].Name(), nil
}
