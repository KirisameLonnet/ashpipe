package hook

import (
	"fmt"
	"strings"
)

// Zsh returns the zsh hook script, using exe as the absolute path to the binary.
func Zsh(exe string) string {
	q := posixShellQuote(exe)
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
`, q, q)
}

// Bash returns the bash hook script.
func Bash(exe string) string {
	q := posixShellQuote(exe)
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
`, q, q)
}

// Fish returns the fish hook script.
func Fish(exe string) string {
	q := fishShellQuote(exe)
	return fmt.Sprintf(`
function _ashpipe_hook --on-variable PWD
  set portal (%s detect 2>/dev/null)
  if test -n "$portal"
    %s connect "$portal"
    # Session ended — step back out of the portal directory.
    cd ..
  end
end
`, q, q)
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

func posixShellQuote(s string) string {
	if s == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func fishShellQuote(s string) string {
	if s == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(s, `'`, `\'`) + "'"
}
