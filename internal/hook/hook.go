package hook

import "fmt"

// Zsh returns the zsh hook script, using exe as the absolute path to the binary.
func Zsh(exe string) string {
	return fmt.Sprintf(`
_ashpipe_hook() {
  local portal
  portal="$(%s detect 2>/dev/null)"
  if [[ -n "$portal" ]]; then
    %s connect "$portal"
    # Session ended — step back out of the portal directory.
    cd ..
  fi
}
autoload -Uz add-zsh-hook
add-zsh-hook chpwd _ashpipe_hook
`, exe, exe)
}

// Bash returns the bash hook script.
func Bash(exe string) string {
	return fmt.Sprintf(`
_ashpipe_hook() {
  local portal
  portal="$(%s detect 2>/dev/null)"
  if [[ -n "$portal" ]]; then
    %s connect "$portal"
    # Session ended — step back out of the portal directory.
    cd ..
  fi
}
if [[ -n "$PROMPT_COMMAND" ]]; then
  PROMPT_COMMAND="_ashpipe_hook; $PROMPT_COMMAND"
else
  PROMPT_COMMAND="_ashpipe_hook"
fi
`, exe, exe)
}

// Fish returns the fish hook script.
func Fish(exe string) string {
	return fmt.Sprintf(`
function _ashpipe_hook --on-variable PWD
  set portal (%s detect 2>/dev/null)
  if test -n "$portal"
    %s connect $portal
    # Session ended — step back out of the portal directory.
    cd ..
  end
end
`, exe, exe)
}

// Print outputs the hook script for the given shell, embedding the binary path.
func Print(shell, exe string) error {
	switch shell {
	case "zsh":
		fmt.Print(Zsh(exe))
	case "bash":
		fmt.Print(Bash(exe))
	case "fish":
		fmt.Print(Fish(exe))
	default:
		return fmt.Errorf("unsupported shell %q; supported: zsh, bash, fish", shell)
	}
	return nil
}
