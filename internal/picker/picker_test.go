package picker

import (
	"io"
	"os"
	"strings"
	"testing"
)

// forceTTY makes every stream look like a terminal for the duration of the
// test, so the TUI branch runs without a real TTY.
func forceTTY(t *testing.T) {
	t.Helper()

	original := isTTY
	isTTY = func(*os.File) bool { return true }
	t.Cleanup(func() { isTTY = original })
}

// pipeOut returns a pipe-backed output file plus a reader that must be
// called once after the interaction to collect everything written to it.
func pipeOut(t *testing.T) (*os.File, func() string) {
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

func TestSelectSingleItemListsWhenNotATerminal(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	var out strings.Builder

	result, item, err := SelectWithOptions([]string{"only"}, Options{Out: &out})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result != Listed || item != "" {
		t.Fatalf("expected Listed and no item, got %v %q", result, item)
	}

	if got := out.String(); got != "only\n" {
		t.Fatalf("expected the single item listed, got %q", got)
	}
}

func TestSelectWithoutItemsFails(t *testing.T) {
	if _, _, err := Select(nil); err == nil {
		t.Fatal("expected an error for an empty item list")
	}
}
