package picker

import (
	"os/exec"
	"testing"
)

func TestSelectWithRealFzf(t *testing.T) {
	if _, err := exec.LookPath("fzf"); err != nil {
		t.Skip("fzf not installed")
	}

	t.Setenv("FZF_DEFAULT_OPTS", "--filter=one")

	result, item, err := SelectWithOptions([]string{"one", "two", "three"}, Options{Mode: ModeFzf})
	if err != nil {
		t.Fatalf("SelectWithOptions() error = %v", err)
	}

	if result != Selected {
		t.Fatalf("result = %v, want Selected", result)
	}

	if item != "one" {
		t.Errorf("item = %q, want %q", item, "one")
	}
}
