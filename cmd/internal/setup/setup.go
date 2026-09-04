// package setup has helpers for setting up the runner
package setup

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/c00/harnesser/config"
	"github.com/c00/harnesser/llm/openrouter"
	"github.com/c00/harnesser/runner"
	"github.com/c00/harnesser/secrets"
	"github.com/spf13/cobra"
)

func Setup(cmd *cobra.Command) (*runner.Runner, error) {
	ctx := cmd.Context()

	// Read config
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("cannot read configuration: %w", err)
	}

	promptsDir := filepath.Join(cfg.DataDir, "prompts")
	toolsDir := filepath.Join(cfg.DataDir, "tools")
	historyDir := filepath.Join(cfg.DataDir, "history")

	// Initialize components
	apiKey, err := secrets.GetSecret(secrets.OpenrouterKeyName)
	if err != nil {
		return nil, fmt.Errorf("cannot get openrouter key: %w", err)
	}

	// TODO abstract this away to support multiple llm backends
	provider := openrouter.New(ctx, cfg.LlmConfig, apiKey)

	// Create runner
	historyFile, _ := cmd.Flags().GetString("thread")

	if cont, _ := cmd.Flags().GetBool("continue"); cont {
		// set history file to the last file
		lastHistoryFile, err := chooseLatest(historyDir)
		if err != nil {
			return nil, fmt.Errorf("cannot get last file from history: %w", err)
		}

		historyFile = lastHistoryFile
	}

	agent, err := runner.NewRunner(provider, promptsDir, historyDir, toolsDir, historyFile)
	if err != nil {
		return nil, fmt.Errorf("cannot create runner: %w", err)
	}

	err = agent.LoadHistory()
	if err != nil {
		return nil, fmt.Errorf("cannot load history: %w", err)
	}

	return agent, nil
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
