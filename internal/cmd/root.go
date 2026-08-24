package cmd

import (
	"github.com/spf13/cobra"
)

type GlobalOptions struct {
	JSON        bool
	Yes         bool
	Verbose     bool
	NoColor     bool
	Environment string
	APIKey      string
}

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
	return command
}
