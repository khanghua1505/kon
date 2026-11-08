package picker

import (
	"bytes"
	"strings"
	"testing"
)

func TestSelectListsItemsWhenNotATerminal(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	var out bytes.Buffer

	result, item, err := SelectWithOptions([]string{"beta", "alpha"}, Options{Out: &out})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result != Listed {
		t.Fatalf("expected Listed, got %v", result)
	}

	if item != "" {
		t.Fatalf("expected no item, got %q", item)
	}

	if got := out.String(); got != "alpha\nbeta\n" {
		t.Fatalf("expected the sorted items, got %q", got)
	}
}

func TestSelectTerminalPromptReadsNumber(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	var out bytes.Buffer

	result, item, err := SelectWithOptions([]string{"alpha", "beta"}, Options{
		Mode: ModeTerminal,
		In:   strings.NewReader("2\n"),
		Out:  &out,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result != Selected {
		t.Fatalf("expected Selected, got %v", result)
	}

	if item != "alpha" {
		t.Fatalf("expected alpha, got %q", item)
	}

	if !strings.Contains(out.String(), "1) beta") || !strings.Contains(out.String(), "2) alpha") {
		t.Fatalf("expected the reversed menu, got %q", out.String())
	}
}

func TestSelectTerminalPromptRepromptsOnInvalidNumber(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	var out bytes.Buffer

	result, item, err := SelectWithOptions([]string{"alpha", "beta"}, Options{
		Mode: ModeTerminal,
		In:   strings.NewReader("abc\n9\n1\n"),
		Out:  &out,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result != Selected {
		t.Fatalf("expected Selected, got %v", result)
	}

	if item != "beta" {
		t.Fatalf("expected beta, got %q", item)
	}

	if got := strings.Count(out.String(), "Invalid selection:"); got != 2 {
		t.Fatalf("expected 2 invalid selections, got %d in %q", got, out.String())
	}
}

func TestSelectTerminalPromptEndsOnEOF(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	var out bytes.Buffer

	result, item, err := SelectWithOptions([]string{"alpha", "beta"}, Options{
		Mode: ModeTerminal,
		In:   strings.NewReader(""),
		Out:  &out,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result != Cancelled {
		t.Fatalf("expected Cancelled, got %v", result)
	}

	if item != "" {
		t.Fatalf("expected no item, got %q", item)
	}
}

func TestSelectReturnsSingleItemImmediately(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	result, item, err := Select([]string{"only"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result != Selected {
		t.Fatalf("expected Selected, got %v", result)
	}

	if item != "only" {
		t.Fatalf("expected only, got %q", item)
	}
}

func TestSelectWithoutItemsFails(t *testing.T) {
	if _, _, err := Select(nil); err == nil {
		t.Fatal("expected an error for an empty item list")
	}
}
