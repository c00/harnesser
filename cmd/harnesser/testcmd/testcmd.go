package testcmd

import (
	"github.com/c00/harnesser/cmd/harnesser/testcmd/decidecmd"
	"github.com/c00/harnesser/cmd/harnesser/testcmd/onecmd"
	"github.com/c00/harnesser/cmd/harnesser/testcmd/stepcmd"
	"github.com/c00/harnesser/cmd/harnesser/testcmd/tooldefcmd"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:           "test",
	Short:         "Subcommands to test individual steps and tools",
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	Cmd.PersistentFlags().StringP("thread", "t", "", "The history thread to load. Can be relative to the current dir or an absolute path")
	Cmd.PersistentFlags().BoolP("continue", "c", false, "Continue the last conversation")

	Cmd.AddCommand(
		decidecmd.Cmd,
		onecmd.Cmd,
		stepcmd.Cmd,
		tooldefcmd.Cmd,
	)
}
