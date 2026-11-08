package picker

import (
	"os"
	"strings"
	"testing"
)

func TestSelectSingleItemRunsTui(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	forceTTY(t)
	inReader, inWriter, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	t.Cleanup(func() {
		_ = inReader.Close()
		_ = inWriter.Close()
	})
	out, readOut := pipeOut(t)

	if _, err := inWriter.WriteString("\r"); err != nil {
		t.Fatalf("write enter: %v", err)
	}

	result, item, err := SelectWithOptions([]string{"only"}, Options{
		In:  inReader,
		Out: out,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result != Selected || item != "only" {
		t.Errorf("result = %v item = %q, want Selected and only", result, item)
	}

	readOut()
}

func TestSelectNonTerminalInputListsItems(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	out, readOut := pipeOut(t)

	result, item, err := SelectWithOptions([]string{"alpha", "beta"}, Options{
		In:  strings.NewReader("1\n"),
		Out: out,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result != Listed || item != "" {
		t.Errorf("result = %v item = %q, want Listed and no item", result, item)
	}

	if got := readOut(); !strings.Contains(got, "alpha\nbeta") {
		t.Errorf("expected the item list, got %q", got)
	}
}
