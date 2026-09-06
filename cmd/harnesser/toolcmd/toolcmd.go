package toolcmd

import (
	"github.com/c00/harnesser/cmd/harnesser/toolcmd/readfilecmd"
	"github.com/c00/harnesser/cmd/harnesser/toolcmd/writefilecmd"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:           "tool",
	Short:         "Built in tools for use by the agent. To add them to your tools folder, use the `create-yamls` sub command.",
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	Cmd.AddCommand(
		readfilecmd.Cmd,
		writefilecmd.Cmd,
	)
}
