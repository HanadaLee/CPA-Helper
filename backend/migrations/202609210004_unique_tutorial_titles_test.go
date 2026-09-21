package migrations

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestUniqueTutorialTitlePreservesExistingNumberedNames(t *testing.T) {
	used := map[string]bool{}
	reserved := map[string]bool{"Codex": true, "Codex (2)": true}
	for i, title := range []string{"Codex", "Codex", "Codex (2)", "Codex"} {
		want := []string{"Codex", "Codex (3)", "Codex (2)", "Codex (4)"}[i]
		if got := uniqueTutorialTitle(title, used, reserved); got != want {
			t.Fatalf("title = %q, want %q", got, want)
		}
	}
}

func TestUniqueTutorialTitleKeepsUnicodeWithinLimit(t *testing.T) {
	title := strings.Repeat("中", 120)
	used := map[string]bool{title: true}
	got := uniqueTutorialTitle(title, used, map[string]bool{title: true})
	if !utf8.ValidString(got) || utf8.RuneCountInString(got) != 120 || !strings.HasSuffix(got, " (2)") {
		t.Fatalf("invalid numbered title: %q", got)
	}
}
