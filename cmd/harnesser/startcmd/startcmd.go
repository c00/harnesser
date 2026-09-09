package startcmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/c00/harnesser/cmd/internal/setup"
	"github.com/c00/harnesser/internal/inputscan"
	"github.com/c00/harnesser/models"
	"github.com/c00/harnesser/runner"
	"github.com/c00/harnesser/tools"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:           "start",
	Short:         "Start the agent in interactive mode",
	Long:          "Start the agent in interactive mode. The agent will loop and give you the opportunity to respond when it is done.",
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
		return fmt.Errorf("cannot setup agent runner: %w", err)
	}

	initialPrompt := strings.Join(args, " ")
	if initialPrompt == "" {
		initialPrompt = inputscan.GetInput(ctx, "You: ")
	}

	agent.AddMessage(models.NewUserTextMessage(initialPrompt))

	// Run loop
	for {
		resp, err := agent.RunStep(ctx)
		if err != nil {
			return fmt.Errorf("cannot get response: %w", err)
		}

		switch resp.Type {
		case runner.ResponseTypeNew:
			ok := askForInput(ctx, agent)
			if !ok {
				return nil
			}
		case runner.ResponseTypeDone:
			fmt.Println("Conversation is over.")
			return nil
		case runner.ResponseTypeInferenceResultNoTools:
			printMessage(resp)
			askForInput(ctx, agent)
		case runner.ResponseTypeInferenceResultWithTools:
			printMessage(resp)
		case runner.ResponseTypeToolResults:
			printMessage(resp)
		case runner.ResponseTypeAskPermission:
			// Ask permission
			err := askPermission(ctx, agent, resp)
			if err != nil {
				return fmt.Errorf("asking permission failed: %w", err)
			}
		}
	}
}

func askForInput(ctx context.Context, agent *runner.Runner) bool {
	prompt := inputscan.GetInput(ctx, "You: ")
	if prompt == "" {
		return false
	}
	agent.AddMessage(models.NewUserTextMessage(strings.TrimSpace(prompt)))

	return true
}

func askPermission(ctx context.Context, agent *runner.Runner, resp runner.Response) error {
	// Ask the user for permission to run tools
	for _, pending := range resp.ToApprove {
		tc := pending.ToolCall
		td := pending.ToolDefinition

		builder, err := tools.NewToolCallBuilder(tc, td)
		if err != nil {
			return fmt.Errorf("cannot create tool call builder: %w", err)
		}

		fmt.Printf("\nNeed approval for: %v\n\nCommand: \n%v\n", tc.Function, builder.CommandString())
		approved := inputscan.GetYesNo(ctx, "Approve this tool call?", false)

		if approved {
			err := agent.ConfirmToolCall(ctx, tc.ToolCallID, models.ToolCallDecisionApprove)
			if err != nil {
				return fmt.Errorf("cannot confirm tool call: %w", err)
			}
		} else {
			err := agent.ConfirmToolCall(ctx, tc.ToolCallID, models.ToolCallDecisionReject)
			if err != nil {
				return fmt.Errorf("cannot reject tool call: %w", err)
			}
		}
	}

	return nil
}

func printMessage(resp runner.Response) {
	if len(resp.Messages) > 0 {
		fmt.Print("Agent: ")
	}
	for _, msg := range resp.Messages {
		fmt.Println(msg.Content.String())
		// Print tool calls if any
		for _, tc := range msg.ToolCalls {
			fmt.Println(tc.String())
		}
	}
}
