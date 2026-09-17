// package setup has helpers for setting up the runner
package setup

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/c00/harnesser/config"
	"github.com/c00/harnesser/historyprovider"
	"github.com/c00/harnesser/landlock"
	"github.com/c00/harnesser/llm/openrouter"
	"github.com/c00/harnesser/promptsprovider"
	"github.com/c00/harnesser/runner"
	"github.com/c00/harnesser/secrets"
	"github.com/c00/harnesser/toolsprovider"
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

	// Initialize History
	history := historyprovider.NewFileProvider(historyDir)

	historyFile, _ := cmd.Flags().GetString("thread")
	cont, _ := cmd.Flags().GetBool("continue")
	// Use latest file to continue
	if historyFile == "" && cont {
		list, err := history.List()
		if err != nil {
			return nil, fmt.Errorf("cannot list history files: %w", err)
		}
		if len(list) > 0 {
			historyFile = list[0]
		}
	}

	// If it's still empty, create a new one
	if historyFile == "" {
		historyFile = historyprovider.NewKey()
	}
	history.Select(historyFile)

	prompts, err := promptsprovider.NewFileProvider(promptsDir)
	if err != nil {
		return nil, fmt.Errorf("cannot create prompts provider: %w", err)
	}

	tools, err := toolsprovider.NewFileProvider(toolsDir)
	if err != nil {
		return nil, fmt.Errorf("cannot create tools provider: %w", err)
	}

	agent := runner.NewRunner(provider, prompts, history, tools, historyFile)
	if err != nil {
		return nil, fmt.Errorf("cannot create runner: %w", err)
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
