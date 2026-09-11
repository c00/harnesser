package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/c00/harnesser/cmd/harnesser/initcmd"
	"github.com/c00/harnesser/cmd/harnesser/startcmd"
	"github.com/c00/harnesser/cmd/harnesser/testcmd"
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
		initcmd.Cmd,
		startcmd.Cmd,
		testcmd.Cmd,
		toolcmd.Cmd,
	)
	logFile, err := setupLogging()
	if err != nil {
		fmt.Printf("cannot setup logging: %v\n", err.Error())
		os.Exit(1)
	}

	defer logFile.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	err = rootCmd.ExecuteContext(ctx)
	if err != nil {
		fmt.Printf("harnesser exited with error: %v\n", err)
		os.Exit(1)
	}
}

func setupLogging() (*os.File, error) {
	logLevel := slog.LevelWarn
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

	logDir := os.Getenv("HARNESSER_LOG_DIR")

	if logDir == "" {
		baseDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("no home dir: %w", err)
		}
		logDir = filepath.Join(baseDir, "log")
	}

	err := os.MkdirAll(logDir, 0o755)
	if err != nil {
		return nil, fmt.Errorf("cannot create log dir: %w", err)
	}

	filename := fmt.Sprintf("harnesser_log_%v.txt", time.Now().Format("20060102_150405"))

	handle, err := os.OpenFile(filepath.Join(logDir, filename), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("cannot open log file: %w", err)
	}

	baseHandler := slog.NewTextHandler(handle, &slog.HandlerOptions{Level: logLevel})
	slog.SetDefault(slog.New(baseHandler))
	slog.SetLogLoggerLevel(logLevel)

	return handle, nil
}
