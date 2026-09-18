package startcmd

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/c00/harnesser/cmd/internal/setup"
	"github.com/c00/harnesser/types"
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
	if initialPrompt != "" {
		agent.AddMessage(types.NewUserTextMessage(initialPrompt))
	}

	// Initialize UI
	var p *tea.Program
	model := initialModel(ctx, agent, initialPrompt != "", func(msg tea.Msg) {
		p.Send(msg)
	})

	p = tea.NewProgram(model, tea.WithContext(ctx))
	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("cannot run program: %w", err)
	}
	if model, ok := finalModel.(tuiModel); ok && model.err != nil {
		return model.err
	}
	return nil

}
