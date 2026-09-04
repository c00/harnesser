package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/c00/harnesser/cmd/harnesser/decidecmd"
	"github.com/c00/harnesser/cmd/harnesser/initcmd"
	"github.com/c00/harnesser/cmd/harnesser/onecmd"
	"github.com/c00/harnesser/cmd/harnesser/startcmd"
	"github.com/c00/harnesser/cmd/harnesser/stepcmd"
	"github.com/c00/harnesser/cmd/harnesser/toolcmd"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:           "harnesser",
	Short:         "Simple AI Harness cli",
	SilenceUsage:  true,
	SilenceErrors: true,
}

func main() {
	// Add subcommands
	rootCmd.AddCommand(
		onecmd.Cmd,
		stepcmd.Cmd,
		initcmd.Cmd,
		decidecmd.Cmd,
		startcmd.Cmd,
		toolcmd.Cmd,
	)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	err := rootCmd.ExecuteContext(ctx)
	if err != nil {
		fmt.Printf("harnesser exited with error: %v\n", err)
		os.Exit(1)
	}
}
