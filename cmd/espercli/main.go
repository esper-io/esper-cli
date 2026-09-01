package main

import (
	"fmt"
	"os"

	"github.com/esper-io/esper-cli/internal/cmd"
	esperruntime "github.com/esper-io/esper-cli/internal/runtime"
)

func main() {
	if err := cmd.NewRootCommand().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, esperruntime.FormatError(err))
		os.Exit(esperruntime.ExitCode(err))
	}
}
