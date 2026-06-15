package hook_test

import (
	"strings"
	"testing"

	"github.com/KirisameLonnet/ashpipe/internal/hook"
)

const testExe = "/usr/local/bin/ashpipe"

func TestZshHookContainsEssentials(t *testing.T) {
	out := hook.Zsh(testExe)
	for _, want := range []string{
		"_ashpipe_hook",
		testExe + " detect",
		testExe + " connect",
		"add-zsh-hook",
		"chpwd",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("zsh hook missing %q", want)
		}
	}
}

func TestBashHookContainsEssentials(t *testing.T) {
	out := hook.Bash(testExe)
	for _, want := range []string{
		"_ashpipe_hook",
		testExe + " detect",
		testExe + " connect",
		"PROMPT_COMMAND",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("bash hook missing %q", want)
		}
	}
}

func TestFishHookContainsEssentials(t *testing.T) {
	out := hook.Fish(testExe)
	for _, want := range []string{
		testExe + " detect",
		testExe + " connect",
		"PWD",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("fish hook missing %q", want)
		}
	}
}

func TestPrintUnsupportedShell(t *testing.T) {
	if err := hook.Print("powershell", testExe); err == nil {
		t.Error("expected error for unsupported shell")
	}
}

func TestPrintSupportedShells(t *testing.T) {
	for _, shell := range []string{"zsh", "bash", "fish"} {
		if err := hook.Print(shell, testExe); err != nil {
			t.Errorf("Print(%q): %v", shell, err)
		}
	}
}
