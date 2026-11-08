// Command kon is a Go port of kubie, a more powerful alternative to
// kubectx and kubens.
package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/khanghua1505/kon/cmd"
)

type exitCoder interface {
	ExitCode() int
}

func main() {
	root := cmd.NewRootCmd()
	cmd.RegisterCommands(root)

	if err := root.Execute(); err != nil {
		var coder exitCoder
		if errors.As(err, &coder) {
			os.Exit(coder.ExitCode())
		}

		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
