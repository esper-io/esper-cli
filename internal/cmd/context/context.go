package context

import (
	"encoding/json"
	"fmt"
	"strings"

	esperruntime "github.com/esper-io/esper-cli/internal/runtime"
	"github.com/spf13/cobra"
)

type contextView struct {
	Device     *esperruntime.ActiveResource `json:"device"`
	App        *esperruntime.ActiveResource `json:"app"`
	Group      *esperruntime.ActiveResource `json:"group"`
	Enterprise *esperruntime.ActiveResource `json:"enterprise"`
}

func NewCommand(options *esperruntime.GlobalOptions) *cobra.Command {
	command := &cobra.Command{Use: "context", Short: "Manage active Esper resource context"}
	set := &cobra.Command{
		Use:       "set <resource> <id>",
		Short:     "Set an active resource",
		Args:      cobra.ExactArgs(2),
		ValidArgs: esperruntime.ContextResources,
		RunE: func(command *cobra.Command, args []string) error {
			return runSet(command, options, args[0], args[1])
		},
	}
	get := &cobra.Command{
		Use:       "get [resource]",
		Short:     "Get one or all active resources",
		Args:      cobra.MaximumNArgs(1),
		ValidArgs: esperruntime.ContextResources,
		RunE: func(command *cobra.Command, args []string) error {
			resource := ""
			if len(args) == 1 {
				resource = args[0]
			}
			return runGet(command, options, resource)
		},
	}
	clear := &cobra.Command{
		Use:       "clear <resource>|--all",
		Short:     "Clear one or all active resources",
		Args:      cobra.MaximumNArgs(1),
		ValidArgs: esperruntime.ContextResources,
		RunE: func(command *cobra.Command, args []string) error {
			all, _ := command.Flags().GetBool("all")
			return runClear(command, options, args, all)
		},
	}
	clear.Flags().Bool("all", false, "clear all active resources")
	command.AddCommand(set, get, clear)
	return command
}

func runSet(command *cobra.Command, options *esperruntime.GlobalOptions, resource, id string) error {
	if err := validateResource(resource); err != nil {
		return err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return esperruntime.NewError(esperruntime.CategoryUsage, fmt.Errorf("context id cannot be empty"))
	}
	store, state, err := loadState()
	if err != nil {
		return err
	}
	if err := state.Active.SetResource(resource, &esperruntime.ActiveResource{ID: id}); err != nil {
		return esperruntime.NewError(esperruntime.CategoryUsage, err)
	}
	if err := store.Save(state); err != nil {
		return esperruntime.NewError(esperruntime.CategoryAuth, err)
	}
	return writeContext(command, options, state.Active, resource)
}

func runGet(command *cobra.Command, options *esperruntime.GlobalOptions, resource string) error {
	if resource != "" {
		if err := validateResource(resource); err != nil {
			return err
		}
	}
	_, state, err := loadState()
	if err != nil {
		return err
	}
	return writeContext(command, options, state.Active, resource)
}

func runClear(command *cobra.Command, options *esperruntime.GlobalOptions, args []string, all bool) error {
	if all == (len(args) == 1) {
		return esperruntime.NewError(esperruntime.CategoryUsage, fmt.Errorf("set exactly one resource or --all"))
	}
	store, state, err := loadState()
	if err != nil {
		return err
	}
	resource := ""
	if all {
		state.Active = esperruntime.ActiveContext{}
	} else {
		resource = args[0]
		if err := validateResource(resource); err != nil {
			return err
		}
		if err := state.Active.SetResource(resource, nil); err != nil {
			return esperruntime.NewError(esperruntime.CategoryUsage, err)
		}
	}
	if err := store.Save(state); err != nil {
		return esperruntime.NewError(esperruntime.CategoryAuth, err)
	}
	return writeContext(command, options, state.Active, resource)
}

func loadState() (*esperruntime.StateStore, esperruntime.State, error) {
	store, err := esperruntime.NewStateStore()
	if err != nil {
		return nil, esperruntime.State{}, esperruntime.NewError(esperruntime.CategoryAuth, err)
	}
	state, err := store.Load()
	if err != nil {
		return nil, esperruntime.State{}, esperruntime.NewError(esperruntime.CategoryAuth, err)
	}
	return store, state, nil
}

func validateResource(resource string) error {
	for _, candidate := range esperruntime.ContextResources {
		if resource == candidate {
			return nil
		}
	}
	return esperruntime.NewError(esperruntime.CategoryUsage, fmt.Errorf("unknown context resource %q (valid: %s)", resource, strings.Join(esperruntime.ContextResources, ", ")))
}

func writeContext(command *cobra.Command, options *esperruntime.GlobalOptions, active esperruntime.ActiveContext, resource string) error {
	if options.JSON {
		if resource == "" {
			return json.NewEncoder(command.OutOrStdout()).Encode(view(active))
		}
		return json.NewEncoder(command.OutOrStdout()).Encode(map[string]*esperruntime.ActiveResource{resource: active.Resource(resource)})
	}
	if resource != "" {
		_, err := fmt.Fprintf(command.OutOrStdout(), "%s: %s\n", resource, resourceID(active.Resource(resource)))
		return err
	}
	for _, name := range esperruntime.ContextResources {
		if _, err := fmt.Fprintf(command.OutOrStdout(), "%s: %s\n", name, resourceID(active.Resource(name))); err != nil {
			return err
		}
	}
	return nil
}

func view(active esperruntime.ActiveContext) contextView {
	return contextView{Device: active.Device, App: active.Application, Group: active.Group, Enterprise: active.Enterprise}
}

func resourceID(resource *esperruntime.ActiveResource) string {
	if resource == nil || resource.ID == "" {
		return "<unset>"
	}
	return resource.ID
}
