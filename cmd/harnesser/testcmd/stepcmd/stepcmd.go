package stepcmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/c00/harnesser/cmd/internal/setup"
	"github.com/c00/harnesser/toolcallbuilder"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:           "step",
	Short:         "Run a single step of the agent process, such as tool calling, or tool result inference",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          run,
}

func init() {
	Cmd.Flags().StringP("thread", "t", "", "The history thread to load. Can be relative to the histdir, relative to the pwd or an absolute path")
	Cmd.Flags().BoolP("continue", "c", false, "Continue the last conversation")
}

func run(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	agent, err := setup.Setup(cmd)
	if err != nil {
		return fmt.Errorf("cannot setup agent: %w", err)
	}

	// Run
	resp, err := agent.RunStep(ctx)
	if err != nil {
		return fmt.Errorf("cannot get response: %w", err)
	}

	for _, msg := range resp.Messages {
		fmt.Println(msg.Content.String())
		// Print tool calls if any
		for _, tc := range msg.ToolCalls {
			fmt.Println(tc.String())
		}
	}

	if len(resp.ToApprove) > 0 {
		fmt.Println("The following tools need approval:")
		for _, pending := range resp.ToApprove {
			builder, err := toolcallbuilder.NewToolCallBuilder(pending.ToolCall, pending.ToolDefinition)
			if err != nil {
				return fmt.Errorf("cannot create new tool call builder: %w", err)
			}
			fmt.Printf("Function: %v, ToolCallID: %v\n", pending.ToolCall.Function, builder.CommandString())
		}
	}

	// Print response
	fmt.Println("Thread Status: ", resp.Type)

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
