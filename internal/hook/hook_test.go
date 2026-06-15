package hook_test

import (
	"strings"
	"testing"

	"github.com/KirisameLonnet/ashpipe/internal/hook"
)

func TestZshHookContainsEssentials(t *testing.T) {
	out := hook.Zsh()
	for _, want := range []string{
		"_ashpipe_hook",
		"ashpipe detect",
		"ashpipe connect",
		"add-zsh-hook",
		"chpwd",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("zsh hook missing %q", want)
		}
	}
}

func TestBashHookContainsEssentials(t *testing.T) {
	out := hook.Bash()
	for _, want := range []string{
		"_ashpipe_hook",
		"ashpipe detect",
		"ashpipe connect",
		"PROMPT_COMMAND",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("bash hook missing %q", want)
		}
	}
}

func TestFishHookContainsEssentials(t *testing.T) {
	out := hook.Fish()
	for _, want := range []string{
		"ashpipe detect",
		"ashpipe connect",
		"PWD",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("fish hook missing %q", want)
		}
	}
}

func TestPrintUnsupportedShell(t *testing.T) {
	if err := hook.Print("powershell"); err == nil {
		t.Error("expected error for unsupported shell")
	}
}

func TestPrintSupportedShells(t *testing.T) {
	for _, shell := range []string{"zsh", "bash", "fish"} {
		if err := hook.Print(shell); err != nil {
			t.Errorf("Print(%q): %v", shell, err)
		}
	}
}
