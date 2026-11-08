package picker

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/khanghua1505/kon/internal/settings"
)

// forceTTY makes every stream look like a terminal for the duration of the
// test, so the prompt branches of the fallback chain run without a real TTY.
func forceTTY(t *testing.T) {
	t.Helper()

	original := isTTY
	isTTY = func(*os.File) bool { return true }
	t.Cleanup(func() { isTTY = original })
}

// promptOut returns a pipe-backed output file plus a reader that must be
// called once after the interaction to collect everything written to it.
func promptOut(t *testing.T) (*os.File, func() string) {
	t.Helper()

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}

	var captured strings.Builder
	done := make(chan struct{})
	go func() {
		_, _ = io.Copy(&captured, reader)
		close(done)
	}()

	t.Cleanup(func() {
		_ = writer.Close()
		_ = reader.Close()
	})

	return writer, func() string {
		_ = writer.Close()
		<-done

		return captured.String()
	}
}

func TestSelectListsItemsWhenNotATerminal(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	var out strings.Builder

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

	forceTTY(t)
	out, readOut := promptOut(t)

	result, item, err := SelectWithOptions([]string{"alpha", "beta"}, Options{
		Tui: settings.TuiSettings{Enabled: false},
		In:  strings.NewReader("2\n"),
		Out: out,
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

	if got := readOut(); !strings.Contains(got, "1) beta") || !strings.Contains(got, "2) alpha") {
		t.Fatalf("expected the reversed menu, got %q", got)
	}
}

func TestSelectTerminalPromptRepromptsOnInvalidNumber(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	forceTTY(t)
	out, readOut := promptOut(t)

	result, item, err := SelectWithOptions([]string{"alpha", "beta"}, Options{
		Tui: settings.TuiSettings{Enabled: false},
		In:  strings.NewReader("abc\n9\n1\n"),
		Out: out,
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

	if got := strings.Count(readOut(), "Invalid selection:"); got != 2 {
		t.Fatalf("expected 2 invalid selections, got %d", got)
	}
}

func TestSelectTerminalPromptEndsOnEOF(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	forceTTY(t)
	out, readOut := promptOut(t)

	result, item, err := SelectWithOptions([]string{"alpha", "beta"}, Options{
		Tui: settings.TuiSettings{Enabled: false},
		In:  strings.NewReader(""),
		Out: out,
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

	readOut()
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
