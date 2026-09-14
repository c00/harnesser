package createtoolscmd

import (
	"fmt"
	"path/filepath"

	"github.com/c00/harnesser/cmd/internal/standardtools"
	"github.com/c00/harnesser/config"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:           "create-standard-tools",
	Aliases:       []string{"c", "create", "create-tools"},
	Short:         "Creates the standard tools in the current .harnesser/tools folder.",
	Long:          `Creates the standard tools in the current .harnesser/tools folder. This includes tools for searching and scraping the web, editing and managing files and running commands.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          run,
	Args:          cobra.NoArgs,
}

func init() {
	Cmd.Flags().BoolP("user", "u", false, "Initialize configs in the user folder (~/.harnesser)")
}

func run(cmd *cobra.Command, args []string) error {
	toolsDir := filepath.Join(config.DirName, config.ToolsDir)

	err := standardtools.WriteStandardTools(toolsDir)
	if err != nil {
		return fmt.Errorf("cannot write standard tools: %w", err)
	}

	fmt.Printf("Tools have been written to %v\n", toolsDir)

	return nil
}
