package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/esper-io/esper-cli/internal/cmd/approval"
	"github.com/esper-io/esper-cli/internal/cmd/configure"
	contextcmd "github.com/esper-io/esper-cli/internal/cmd/context"
	"github.com/esper-io/esper-cli/internal/cmd/discover"
	"github.com/esper-io/esper-cli/internal/cmd/generated"
	"github.com/esper-io/esper-cli/internal/cmd/secureadb"
	esperruntime "github.com/esper-io/esper-cli/internal/runtime"
	"github.com/esper-io/esper-cli/internal/version"
	"github.com/spf13/cobra"
)

type GlobalOptions = esperruntime.GlobalOptions

func NewRootCommand() *cobra.Command {
	options := &GlobalOptions{}
	var startWindow bool
	command := &cobra.Command{
		Use:           "espercli",
		Short:         "Manage Esper resources",
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	command.Args = cobra.NoArgs
	command.RunE = func(command *cobra.Command, _ []string) error {
		if startWindow {
			return approval.StartWindow(command, options)
		}
		return command.Help()
	}
	command.PersistentPreRunE = func(invoked *cobra.Command, _ []string) error {
		if invoked != command && command.Flags().Changed("dangerously-skip-permissions") {
			return esperruntime.NewError(esperruntime.CategoryUsage, fmt.Errorf("--dangerously-skip-permissions must be used alone, without a subcommand"))
		}
		if err := esperruntime.ValidateApprovalMode(options.ApprovalMode); err != nil {
			return esperruntime.NewError(esperruntime.CategoryUsage, err)
		}
		return nil
	}
	command.Flags().BoolVar(&startWindow, "dangerously-skip-permissions", false, "start a human-confirmed 15-minute approval window across all tenants")
	command.SetUsageTemplate(generated.UsageTemplate("Command groups"))
	flags := command.PersistentFlags()
	flags.StringVar(&options.ApprovalMode, "approval-mode", "request", "write approval: request or agent (trust permission obtained in the agent UI)")
	flags.BoolVarP(&options.JSON, "json", "j", false, "write raw API JSON")
	flags.BoolVarP(&options.Yes, "yes", "y", false, "skip destructive y/n prompt after human approval")
	flags.BoolVarP(&options.Verbose, "verbose", "v", false, "write verbose diagnostics to stderr")
	flags.BoolVar(&options.NoColor, "no-color", false, "disable colored output")
	flags.StringVar(&options.Environment, "environment", "", "Esper environment (overrides ESPER_ENVIRONMENT)")
	flags.StringVar(&options.APIKey, "api-key", "", "Esper API key (overrides ESPER_API_KEY)")
	generated.AddCommands(command, options)
	command.AddCommand(configure.NewCommand(options))
	command.AddCommand(contextcmd.NewCommand(options))
	command.AddCommand(approval.NewCommand(options))
	command.AddCommand(discover.NewCommand(options))
	command.AddCommand(secureadb.NewCommand(options))
	addVersionCommand(command, options)
	for _, child := range command.Commands() {
		if child.Name() != "api" {
			child.SetUsageTemplate(generated.UsageTemplate("Operations"))
		}
	}
	return command
}

func addVersionCommand(root *cobra.Command, options *GlobalOptions) {
	command, _, err := root.Find([]string{"version"})
	if err != nil || command == root {
		command = &cobra.Command{Use: "version"}
		root.AddCommand(command)
	}
	command.Short = "Show build information or manage API versions"
	command.Args = cobra.NoArgs
	command.RunE = func(command *cobra.Command, _ []string) error {
		info := version.Current()
		if options.JSON {
			return json.NewEncoder(command.OutOrStdout()).Encode(info)
		}
		_, err := fmt.Fprintln(command.OutOrStdout(), info.String())
		return err
	}
}
