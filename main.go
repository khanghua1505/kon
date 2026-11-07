// Command kubie is a Go port of kubie, a more powerful alternative to
// kubectx and kubens.
package main

import (
	"fmt"
	"os"

	"github.com/khanghua1505/kubie/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
