// package setup has helpers for setting up the runner
package setup

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/c00/harnesser/config"
	"github.com/c00/harnesser/landlock"
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

	// landlock
	if cfg.Landlock.Active {
		slog.Debug("Activating LSM Landlock")

		roDirs := []string{}
		roDirs = append(roDirs, cfg.Landlock.ExtraRODirs...)
		roDirs = append(roDirs, landlock.PathToDirs()...)
		roDirs = append(roDirs, landlock.RequiredRODirs()...)

		rwDirs := []string{}
		rwDirs = append(rwDirs, cfg.Landlock.ExtraRWDirs...)

		// Current working directory
		wd, _ := os.Getwd()
		home, _ := os.UserHomeDir()
		// I don't want the whole home folder to be accessible. Seems like it could cause issues.
		// A lot of secrets may be there, also writing could destroy a lot of things.
		if wd != "" && wd != home {
			rwDirs = append(rwDirs, wd)
		}

		// User config dir
		if home != "" {
			homeConfig := filepath.Join(home, config.DirName)
			if wd == home {
				// Add writable config dir
				rwDirs = append(rwDirs, homeConfig)
			} else {
				// Add readable config dir
				roDirs = append(roDirs, homeConfig)
			}
		}

		err := landlock.Landlock(roDirs, rwDirs)
		if err != nil {
			return nil, fmt.Errorf("cannot landlock: %w", err)
		}
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
