package picker

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// promptList shows the items as a numbered list and reads the chosen number
// from the input. An invalid number is reported and asked again.
func promptList(items []string, opts Options) (Result, string, error) {
	for i, item := range items {
		if _, err := fmt.Fprintf(opts.Out, "  %d) %s\n", i+1, item); err != nil {
			return Cancelled, "", fmt.Errorf("could not print the item list: %w", err)
		}
	}

	reader := bufio.NewReader(opts.In)

	for {
		if _, err := fmt.Fprintf(opts.Out, "Select an item [1-%d]: ", len(items)); err != nil {
			return Cancelled, "", fmt.Errorf("could not print the prompt: %w", err)
		}

		line, err := reader.ReadString('\n')
		if err != nil && line == "" {
			if errors.Is(err, io.EOF) {
				return Cancelled, "", nil
			}

			return Cancelled, "", fmt.Errorf("could not read the selection: %w", err)
		}

		answer := strings.TrimSpace(line)

		choice, convErr := strconv.Atoi(answer)
		if convErr != nil || choice < 1 || choice > len(items) {
			_, _ = fmt.Fprintf(opts.Out, "Invalid selection: %s\n", answer)

			if err != nil {
				// The input is exhausted, there is nothing left to read.
				return Cancelled, "", nil
			}

			continue
		}

		return Selected, items[choice-1], nil
	}
}
