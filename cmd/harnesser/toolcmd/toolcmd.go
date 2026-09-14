package toolcmd

import (
	createtoolscmd "github.com/c00/harnesser/cmd/harnesser/toolcmd/createtools"
	"github.com/c00/harnesser/cmd/harnesser/toolcmd/readfilecmd"
	"github.com/c00/harnesser/cmd/harnesser/toolcmd/replacetextcmd"
	"github.com/c00/harnesser/cmd/harnesser/toolcmd/writefilecmd"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:           "tool",
	Short:         "Built in tools for use by the agent. To add them to your tools folder, use the `create-standard-tools` sub command.",
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	Cmd.AddCommand(
		readfilecmd.Cmd,
		replacetextcmd.Cmd,
		writefilecmd.Cmd,
		createtoolscmd.Cmd,
	)
}
