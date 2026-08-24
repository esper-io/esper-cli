package cmd

import (
	"github.com/esper-io/esper-cli/internal/cmd/generated"
	esperruntime "github.com/esper-io/esper-cli/internal/runtime"
	"github.com/spf13/cobra"
)

type GlobalOptions = esperruntime.GlobalOptions

func NewRootCommand() *cobra.Command {
	options := &GlobalOptions{}
	command := &cobra.Command{
		Use:           "espercli",
		Short:         "Manage Esper resources",
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	flags := command.PersistentFlags()
	flags.BoolVarP(&options.JSON, "json", "j", false, "write raw API JSON")
	flags.BoolVarP(&options.Yes, "yes", "y", false, "skip destructive-operation confirmation")
	flags.BoolVarP(&options.Verbose, "verbose", "v", false, "write verbose diagnostics to stderr")
	flags.BoolVar(&options.NoColor, "no-color", false, "disable colored output")
	flags.StringVar(&options.Environment, "environment", "", "Esper environment (overrides ESPER_ENVIRONMENT)")
	flags.StringVar(&options.APIKey, "api-key", "", "Esper API key (overrides ESPER_API_KEY)")
	generated.AddCommands(command, options)
	return command
}
