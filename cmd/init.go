package cmd

import (
	"fmt"
	"os"

	"github.com/KirisameLonnet/ashpipe/internal/config"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize an ashpipe workspace in the current directory",
	Long: `Creates .ashpipe/config.yaml in the current directory.

Run this once per workspace, then use 'ashpipe add' to define portals.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}

		// Warn if already inside an existing workspace.
		if root, err := config.FindRoot(cwd); err == nil && root != cwd {
			fmt.Fprintf(os.Stderr,
				"[ashpipe] Warning: already inside workspace at %s\n"+
					"         Initializing a nested workspace anyway.\n",
				root,
			)
		}

		if err := config.Init(cwd); err != nil {
			return err
		}

		fmt.Printf("Initialized ashpipe workspace in %s/.ashpipe/\n\n", cwd)
		fmt.Println("Next steps:")
		fmt.Println("  1. Add a portal:")
		fmt.Println("       ashpipe add <portal-name> <user@host:/remote/path>")
		fmt.Println("  2. Set up shell hook (add to ~/.zshrc or ~/.bashrc):")
		fmt.Println("       eval \"$(ashpipe hook zsh)\"")
		fmt.Println("  3. Connect by entering the portal directory:")
		fmt.Println("       cd <portal-name>/")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
