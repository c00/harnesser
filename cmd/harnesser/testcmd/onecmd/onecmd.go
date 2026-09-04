package onecmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/c00/harnesser/config"
	"github.com/c00/harnesser/llm/openrouter"
	"github.com/c00/harnesser/models"
	"github.com/c00/harnesser/runner"
	"github.com/c00/harnesser/secrets"
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

	// Read config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("cannot read configuration: %w", err)
	}

	promptsDir := filepath.Join(cfg.DataDir, "prompts")
	toolsDir := filepath.Join(cfg.DataDir, "tools")
	historyDir := filepath.Join(cfg.DataDir, "history")

	// Initialize components
	apiKey, err := secrets.GetSecret(secrets.OpenrouterKeyName)
	if err != nil {
		return fmt.Errorf("cannot get openrouter key: %w", err)
	}

	// TODO abstract this away to support multiple llm backends
	provider := openrouter.New(ctx, cfg.LlmConfig, apiKey)

	// Create runner
	historyFile, _ := cmd.Flags().GetString("thread")

	if cont, _ := cmd.Flags().GetBool("continue"); cont {
		// set history file to the last file
		lastHistoryFile, err := chooseLatest(historyDir)
		if err != nil {
			return fmt.Errorf("cannot get last file from history: %w", err)
		}

		historyFile = lastHistoryFile
		fmt.Println("loaded last file", historyFile)
	}

	agent, err := runner.NewRunner(provider, promptsDir, historyDir, toolsDir, historyFile)
	if err != nil {
		return fmt.Errorf("cannot create runner: %w", err)
	}

	err = agent.LoadHistory()
	if err != nil {
		return fmt.Errorf("cannot load history: %w", err)
	}

	// Build prompt
	prompt := strings.Join(args, " ")

	if prompt == "" {
		return fmt.Errorf("missing user prompt")
	}

	// Run
	resp, err := agent.RunPrompt(ctx, models.NewUserTextMessage(prompt))
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
