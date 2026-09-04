package readfilecmd

import (
	"bytes"
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:           "read-file name",
	Example:       `read-file README.md --start=1 --lines=100`,
	Short:         "Read a (part of) a file.",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          run,
	Args:          cobra.ExactArgs(1),
}

func init() {
	Cmd.Flags().IntP("start", "s", 1, "Starting line. The first line is 1.")
	Cmd.Flags().IntP("max-lines", "l", 100, "The maximum number of lines to include.")
	Cmd.Flags().IntP("max-bytes", "b", 1024, "The maximum number of bytes to include.")
	Cmd.Flags().Bool("line-numbers", false, "Set to true to include line numbers in the output.")
}

func run(cmd *cobra.Command, args []string) error {
	startLine, _ := cmd.Flags().GetInt("start")
	maxLines, _ := cmd.Flags().GetInt("max-lines")
	maxBytes, _ := cmd.Flags().GetInt("max-bytes")
	includeNrs, _ := cmd.Flags().GetBool("line-numbers")

	filename := args[0]

	data, err := readFile(filename, startLine, maxLines, maxBytes, includeNrs)
	if err != nil {
		return fmt.Errorf("cannot read file '%v': %w", filename, err)
	}

	fmt.Print(string(data))

	return nil
}

func readFile(filename string, startLine, maxLines, maxBytes int, includeLineNr bool) ([]byte, error) {
	if startLine < 1 {
		return nil, fmt.Errorf("start line must be at least 1")
	}
	if maxLines < 0 {
		return nil, fmt.Errorf("maximum number of lines cannot be negative")
	}
	if maxBytes < 0 {
		return nil, fmt.Errorf("maximum number of bytes cannot be negative")
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	lines := splitLines(data)
	if startLine > len(lines) {
		return nil, fmt.Errorf("start line %d exceeds file length of %d lines", startLine, len(lines))
	}

	endLine := startLine - 1 + maxLines
	if endLine > len(lines) {
		endLine = len(lines)
	}
	selected := lines[startLine-1 : endLine]

	var output bytes.Buffer
	if includeLineNr && len(selected) > 0 {
		width := len(strconv.Itoa(endLine))
		for i, line := range selected {
			fmt.Fprintf(&output, "%*d ", width, startLine+i)
			output.Write(line)
		}
	} else {
		for _, line := range selected {
			output.Write(line)
		}
	}

	if output.Len() <= maxBytes {
		return output.Bytes(), nil
	}

	result := append([]byte(nil), output.Bytes()[:maxBytes]...)
	if len(result) > 0 && result[len(result)-1] != '\n' {
		result = append(result, '\n')
	}
	result = append(result, "[output truncated: maximum byte count reached]\n"...)
	return result, nil
}

// splitLines returns physical file lines and retains their line endings.
func splitLines(data []byte) [][]byte {
	if len(data) == 0 {
		return nil
	}

	lines := bytes.SplitAfter(data, []byte{'\n'})
	if len(lines[len(lines)-1]) == 0 {
		lines = lines[:len(lines)-1]
	}
	return lines
}
