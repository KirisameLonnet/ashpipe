package hook

import "fmt"

// Zsh returns the zsh hook script to be eval'd in ~/.zshrc.
func Zsh() string {
	return `
_ashpipe_hook() {
  local portal
  portal="$(ashpipe detect 2>/dev/null)"
  if [[ -n "$portal" ]]; then
    ashpipe connect "$portal"
  fi
}
autoload -Uz add-zsh-hook
add-zsh-hook chpwd _ashpipe_hook
`
}

// Bash returns the bash hook script to be eval'd in ~/.bashrc.
func Bash() string {
	return `
_ashpipe_hook() {
  local portal
  portal="$(ashpipe detect 2>/dev/null)"
  if [[ -n "$portal" ]]; then
    ashpipe connect "$portal"
  fi
}
if [[ -n "$PROMPT_COMMAND" ]]; then
  PROMPT_COMMAND="_ashpipe_hook; $PROMPT_COMMAND"
else
  PROMPT_COMMAND="_ashpipe_hook"
fi
`
}

// Fish returns the fish hook script.
func Fish() string {
	return `
function _ashpipe_hook --on-variable PWD
  set portal (ashpipe detect 2>/dev/null)
  if test -n "$portal"
    ashpipe connect $portal
  end
end
`
}

// Print outputs the hook script for the given shell.
func Print(shell string) error {
	switch shell {
	case "zsh":
		fmt.Print(Zsh())
	case "bash":
		fmt.Print(Bash())
	case "fish":
		fmt.Print(Fish())
	default:
		return fmt.Errorf("unsupported shell %q; supported: zsh, bash, fish", shell)
	}
	return nil
}
