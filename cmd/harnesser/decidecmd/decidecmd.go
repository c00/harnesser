package decidecmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/c00/harnesser/config"
	"github.com/c00/harnesser/llm/openrouter"
	"github.com/c00/harnesser/models"
	"github.com/c00/harnesser/runner"
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

	// Read config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("cannot read configuration: %w", err)
	}

	promptsDir := filepath.Join(cfg.DataDir, "prompts")
	toolsDir := filepath.Join(cfg.DataDir, "tools")
	historyDir := filepath.Join(cfg.DataDir, "history")

	// Initialize components
	// TODO some secrets management solution
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		return errors.New("no openrouter api key set")
	}

	// TODO abstract this away to support multiple llm backends
	provider := openrouter.New(ctx, cfg.LlmConfig, apiKey)

	// Create runner
	historyFile, _ := cmd.Flags().GetString("thread")
	// Here we need a file, otherwise this command makes no sense. So if historyFile is empty, we will choose the latest file in the history folder
	if historyFile == "" {
		var err error
		historyFile, err = chooseLatest(historyDir)
		if err != nil {
			return fmt.Errorf("cannot find latest file: %w", err)
		}
	}

	if cont, _ := cmd.Flags().GetBool("continue"); cont {
		// set history file to the last file
		lastHistoryFile, err := chooseLatest(historyDir)
		if err != nil {
			return fmt.Errorf("cannot get last file from history: %w", err)
		}

		historyFile = lastHistoryFile
	}

	agent, err := runner.NewRunner(provider, promptsDir, historyDir, toolsDir, historyFile)
	if err != nil {
		return fmt.Errorf("cannot create runner: %w", err)
	}

	err = agent.LoadHistory()
	if err != nil {
		return fmt.Errorf("cannot load history: %w", err)
	}

	approve, _ := cmd.Flags().GetBool("approve")
	decision := models.ToolCallDecisionReject
	if approve {
		decision = models.ToolCallDecisionApprove
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
