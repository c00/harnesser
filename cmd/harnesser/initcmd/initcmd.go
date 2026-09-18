package initcmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/c00/harnesser/cmd/internal/secrets"
	"github.com/c00/harnesser/cmd/internal/standardtools"
	"github.com/c00/harnesser/config"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:           "init",
	Short:         "Initialize Harnesser (locally or for the user)",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          run,
}

func init() {
	Cmd.Flags().BoolP("user", "u", false, "Initialize configs in the user folder (~/.harnesser)")
	Cmd.Flags().BoolP("standard-tools", "t", true, "Add the standard tools for web searching, file management and running commands")
}

func run(cmd *cobra.Command, args []string) error {
	forUser, _ := cmd.Flags().GetBool("user")
	addTools, _ := cmd.Flags().GetBool("standard-tools")

	targetConfigFile := filepath.Join(config.DirName, config.ConfigFile)
	path := filepath.Join(config.DirName)

	if forUser {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("cannot get user home dir: %w", err)
		}
		targetConfigFile = filepath.Join(home, targetConfigFile)
		path = filepath.Join(home, path)

		err = secrets.ApiKeySetup(cmd.Context())
		if err != nil {
			return fmt.Errorf("cannot setup api key: %w", err)
		}
	}

	_, err := os.Stat(targetConfigFile)
	if err == nil || !os.IsNotExist(err) {
		fmt.Println("harnesser already initialized")
		return nil
	}

	err = config.Write(config.Defaults(), forUser)
	if err != nil {
		return fmt.Errorf("cannot init config file: %w", err)
	}

	toolsDir := filepath.Join(path, config.ToolsDir)
	// Create dirs
	err = errors.Join(
		ensureDir(toolsDir),
		ensureDir(filepath.Join(path, config.HistoryDir)),
		ensureDir(filepath.Join(path, config.PromptsDir)),
	)

	emptyToolsDir, err := isEmpty(toolsDir)
	if err != nil {
		return fmt.Errorf("cannot check tools dir: %w", err)
	}

	if addTools && emptyToolsDir {
		err := standardtools.WriteStandardTools(toolsDir)
		if err != nil {
			return fmt.Errorf("cannot write standard tools: %w", err)
		}
	}

	fmt.Printf("Configuration setup at %v\n", path)

	return nil
}

func ensureDir(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("cannot create directory '%v': %w", dir, err)
	}
	return nil
}

func isEmpty(path string) (bool, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return false, err
	}
	return len(entries) == 0, nil
}
