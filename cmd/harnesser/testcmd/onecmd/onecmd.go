package onecmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/c00/harnesser/cmd/internal/setup"
	"github.com/c00/harnesser/types"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:           "one",
	Short:         "Run a single-turn inference",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          run,
}

func init() {
	Cmd.Flags().StringP("thread", "t", "", "The history thread to load. Can be relative to the current dir or an absolute path")
	Cmd.Flags().BoolP("continue", "c", false, "Continue the last conversation")
}

func run(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	agent, err := setup.Setup(cmd)
	if err != nil {
		return fmt.Errorf("cannot setup agent: %w", err)
	}

	// Build prompt
	prompt := strings.Join(args, " ")

	if prompt == "" {
		return fmt.Errorf("missing user prompt")
	}

	// Run
	resp, err := agent.RunPrompt(ctx, types.NewUserTextMessage(prompt))
	if err != nil {
		return fmt.Errorf("cannot get response: %w", err)
	}

	// Print response
	fmt.Println(resp.Content.String())

	// Print tool calls if any
	for _, tc := range resp.ToolCalls {
		fmt.Println(tc.String())
	}

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
