package sftp

import (
	"strings"
	"testing"
)

func TestApplyEditBasic(t *testing.T) {
	result, err := applyEdit("hello world\nfoo bar\n", "foo bar", "baz qux")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result, "baz qux") {
		t.Errorf("replacement not applied: %q", result)
	}
	if strings.Contains(result, "foo bar") {
		t.Errorf("old string still present: %q", result)
	}
}

func TestApplyEditNotFound(t *testing.T) {
	_, err := applyEdit("hello world", "nonexistent", "x")
	if err == nil {
		t.Error("expected error when old_string not found")
	}
}

func TestApplyEditAmbiguous(t *testing.T) {
	_, err := applyEdit("foo foo", "foo", "bar")
	if err == nil {
		t.Error("expected error when old_string appears multiple times")
	}
}

func TestApplyEditMultiline(t *testing.T) {
	content := "func hello() {\n\treturn 1\n}\n"
	updated, err := applyEdit(content, "return 1", "return 42")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(updated, "return 42") {
		t.Errorf("multiline edit not applied: %q", updated)
	}
}

func TestUnifiedDiffChanged(t *testing.T) {
	a := "line1\nline2\nline3\n"
	b := "line1\nchanged\nline3\n"
	diff := unifiedDiff("a.txt", "b.txt", a, b)

	if !strings.Contains(diff, "--- a.txt") {
		t.Error("missing --- header")
	}
	if !strings.Contains(diff, "+++ b.txt") {
		t.Error("missing +++ header")
	}
	if !strings.Contains(diff, "- line2") {
		t.Error("missing removed line")
	}
	if !strings.Contains(diff, "+ changed") {
		t.Error("missing added line")
	}
}

func TestUnifiedDiffIdentical(t *testing.T) {
	content := "same\ncontent\n"
	diff := unifiedDiff("a.txt", "b.txt", content, content)
	for _, line := range strings.Split(diff, "\n") {
		if strings.HasPrefix(line, "+ ") || strings.HasPrefix(line, "- ") {
			t.Errorf("identical files should produce no +/- lines, got line: %q\nfull diff:\n%s", line, diff)
		}
	}
}

func TestUnifiedDiffAddedLines(t *testing.T) {
	a := "line1\n"
	b := "line1\nline2\n"
	diff := unifiedDiff("a.txt", "b.txt", a, b)
	if !strings.Contains(diff, "+ line2") {
		t.Errorf("missing added line in diff:\n%s", diff)
	}
}
