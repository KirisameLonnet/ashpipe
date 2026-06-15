package cmd

import (
	"fmt"
	"os"

	"github.com/KirisameLonnet/ashpipe/internal/detect"
	"github.com/spf13/cobra"
)

var detectCmd = &cobra.Command{
	Use:    "detect",
	Short:  "Detect if the current directory is inside a portal (used by shell hook)",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		result, err := detect.FromCwd()
		if err != nil {
			return err
		}
		if result == nil {
			// Not in a portal — print nothing, exit 0.
			return nil
		}
		// Print the portal name so the shell hook can pass it to `ashpipe connect`.
		fmt.Fprintln(os.Stdout, result.PortalName)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(detectCmd)
}
