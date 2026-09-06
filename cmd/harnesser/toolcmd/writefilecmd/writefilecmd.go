package writefilecmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:           "write-file name contents",
	Example:       `write-file README.md "this is clean readme file"`,
	Short:         "Write to a file. This will truncate the file and replace its contents if it already exists. There is an append flag to append instead of replace.",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          run,
	Args:          cobra.ExactArgs(2),
}

func init() {
	Cmd.Flags().Bool("append", false, "Set to true to append to the end of the file rather than replacing its contents.")
}

func run(cmd *cobra.Command, args []string) error {
	append, _ := cmd.Flags().GetBool("append")

	filename := args[0]
	contents := args[1]

	flags := os.O_CREATE | os.O_WRONLY | os.O_TRUNC
	if append {
		flags = os.O_APPEND | os.O_CREATE | os.O_WRONLY
	}

	// Open file with Append, Create (if doesn't exist), and WriteOnly flags
	f, err := os.OpenFile(filename, flags, 0644)
	if err != nil {
		return fmt.Errorf("cannot open file %v: %w", filename, err)
	}
	defer f.Close()

	written, err := f.WriteString(contents)
	if err != nil {
		return fmt.Errorf("cannot write to file: %w", err)
	}

	fmt.Printf("%v bytes written.\n", written)

	return nil
}
