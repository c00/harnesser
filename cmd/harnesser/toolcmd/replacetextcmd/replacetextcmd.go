package replacetextcmd

import (
	"bytes"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:           "replace-text name search replacement",
	Example:       `replace-text README.md "old text" "new text" --expected-matches=1`,
	Short:         "Replace literal text in a file when the number of matches is as expected.",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          run,
	Args:          cobra.ExactArgs(3),
}

func init() {
	Cmd.Flags().Int("expected-matches", 1, "The number of occurrences that must be found before editing the file.")
}

func run(cmd *cobra.Command, args []string) error {
	expectedMatches, _ := cmd.Flags().GetInt("expected-matches")

	filename := args[0]
	search := args[1]
	replacement := args[2]

	replaced, err := replaceText(filename, search, replacement, expectedMatches)
	if err != nil {
		return fmt.Errorf("cannot replace text in file '%v': %w", filename, err)
	}

	fmt.Printf("%d occurrences replaced.\n", replaced)

	return nil
}

func replaceText(filename, search, replacement string, expectedMatches int) (int, error) {
	if search == "" {
		return 0, fmt.Errorf("search text cannot be empty")
	}
	if expectedMatches < 0 {
		return 0, fmt.Errorf("expected matches cannot be negative")
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		return 0, err
	}

	searchBytes := []byte(search)
	matches := bytes.Count(data, searchBytes)
	if matches != expectedMatches {
		return 0, fmt.Errorf("expected %d matches, found %d", expectedMatches, matches)
	}

	if matches == 0 {
		return 0, nil
	}

	result := bytes.ReplaceAll(data, searchBytes, []byte(replacement))
	if err := os.WriteFile(filename, result, 0o644); err != nil {
		return 0, err
	}

	return matches, nil
}
