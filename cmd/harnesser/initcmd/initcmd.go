package initcmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

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
}

func run(cmd *cobra.Command, args []string) error {
	forUser, _ := cmd.Flags().GetBool("user")

	targetConfigFile := filepath.Join(config.DirName, config.ConfigFile)
	path := filepath.Join(config.DirName)

	if forUser {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("cannot get user home dir: %w", err)
		}
		targetConfigFile = filepath.Join(home, targetConfigFile)
		path = filepath.Join(home, path)
	}

	_, err := os.Stat(targetConfigFile)
	if err == nil || !os.IsNotExist(err) {
		return fmt.Errorf("harnesser already initialized")
	}

	err = config.Write(config.Defaults(), forUser)
	if err != nil {
		return fmt.Errorf("cannot init config file: %w", err)
	}

	// Create dirs
	err = errors.Join(
		ensureDir(filepath.Join(path, config.ToolsDir)),
		ensureDir(filepath.Join(path, config.HistoryDir)),
		ensureDir(filepath.Join(path, config.PromptsDir)),
	)

	fmt.Printf("Configuration setup at %v\n", path)

	return nil
}

func ensureDir(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("cannot create directory '%v': %w", dir, err)
	}
	return nil
}
