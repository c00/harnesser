package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/c00/harnesser/cmd/harnesser/decidecmd"
	"github.com/c00/harnesser/cmd/harnesser/initcmd"
	"github.com/c00/harnesser/cmd/harnesser/onecmd"
	"github.com/c00/harnesser/cmd/harnesser/startcmd"
	"github.com/c00/harnesser/cmd/harnesser/stepcmd"
	"github.com/c00/harnesser/cmd/harnesser/toolcmd"
	"github.com/c00/harnesser/version"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:           "harnesser",
	Short:         "Simple AI Harness cli",
	SilenceUsage:  true,
	SilenceErrors: true,
	Version:       version.Version,
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

	logLevel := slog.LevelDebug
	logLevelStr := strings.ToLower(os.Getenv("HARNESSER_LOG_LEVEL"))

	switch logLevelStr {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	}

	baseHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel})
	slog.SetDefault(slog.New(baseHandler))
	slog.SetLogLoggerLevel(logLevel)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	err := rootCmd.ExecuteContext(ctx)
	if err != nil {
		fmt.Printf("harnesser exited with error: %v\n", err)
		os.Exit(1)
	}
}
