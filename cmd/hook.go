package cmd

import (
	"github.com/KirisameLonnet/ashpipe/internal/hook"
	"github.com/spf13/cobra"
)

var hookCmd = &cobra.Command{
	Use:   "hook <shell>",
	Short: "Print shell hook to add to your shell config",
	Long: `Prints a shell hook that enables automatic portal detection on cd.

Add the output of this command to your shell's startup config:

  zsh  (~/.zshrc, or programs.zsh.initExtra in home-manager):
    eval "$(ashpipe hook zsh)"

  bash (~/.bashrc):
    eval "$(ashpipe hook bash)"

  fish (~/.config/fish/config.fish):
    ashpipe hook fish | source

Nix / home-manager users: do not echo directly into ~/.zshrc.
Add the line above to your home-manager configuration instead.`,
	Args:      cobra.ExactArgs(1),
	ValidArgs: []string{"zsh", "bash", "fish"},
	RunE: func(cmd *cobra.Command, args []string) error {
		return hook.Print(args[0])
	},
}

func init() {
	rootCmd.AddCommand(hookCmd)
}
