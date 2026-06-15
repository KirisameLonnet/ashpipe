package cmd

import (
	"github.com/KirisameLonnet/ashpipe/internal/hook"
	"github.com/spf13/cobra"
)

var hookCmd = &cobra.Command{
	Use:   "hook <shell>",
	Short: "Print shell hook to eval in your shell config",
	Long: `Add to your shell config:
  zsh:  eval "$(ashpipe hook zsh)"
  bash: eval "$(ashpipe hook bash)"
  fish: ashpipe hook fish | source`,
	Args:      cobra.ExactArgs(1),
	ValidArgs: []string{"zsh", "bash", "fish"},
	RunE: func(cmd *cobra.Command, args []string) error {
		return hook.Print(args[0])
	},
}

func init() {
	rootCmd.AddCommand(hookCmd)
}
