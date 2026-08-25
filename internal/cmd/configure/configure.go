package configure

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	esperruntime "github.com/esper-io/esper-cli/internal/runtime"
	"github.com/spf13/cobra"
)

type configView struct {
	Environment string `json:"environment"`
	APIKey      string `json:"api_key"`
}

func NewCommand(options *esperruntime.GlobalOptions) *cobra.Command {
	command := &cobra.Command{
		Use:   "configure",
		Short: "Configure the Esper environment and API key",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			return runConfigure(command, options)
		},
	}
	command.AddCommand(&cobra.Command{
		Use:   "show",
		Short: "Show the configured environment and redacted API key",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			return runShow(command, options)
		},
	})
	return command
}

func runConfigure(command *cobra.Command, options *esperruntime.GlobalOptions) error {
	environment := strings.TrimSpace(options.Environment)
	apiKey := strings.TrimSpace(options.APIKey)
	reader := bufio.NewReader(command.InOrStdin())
	var err error
	if environment == "" {
		environment, err = prompt(reader, command.ErrOrStderr(), "Environment")
		if err != nil {
			return esperruntime.NewError(esperruntime.CategoryUsage, err)
		}
	}
	if apiKey == "" {
		apiKey, err = prompt(reader, command.ErrOrStderr(), "API key")
		if err != nil {
			return esperruntime.NewError(esperruntime.CategoryUsage, err)
		}
	}

	store, state, err := loadState()
	if err != nil {
		return err
	}
	state.Config.Environment = environment
	state.Config.APIKey = apiKey
	if err := store.Save(state); err != nil {
		return esperruntime.NewError(esperruntime.CategoryAuth, err)
	}
	_, err = fmt.Fprintln(command.OutOrStdout(), "Configuration saved.")
	return err
}

func runShow(command *cobra.Command, options *esperruntime.GlobalOptions) error {
	_, state, err := loadState()
	if err != nil {
		return err
	}
	if state.Config.Environment == "" || state.Config.APIKey == "" {
		return esperruntime.NewError(esperruntime.CategoryAuth, fmt.Errorf("Esper credentials are not configured"))
	}
	view := configView{Environment: state.Config.Environment, APIKey: redact(state.Config.APIKey)}
	if options.JSON {
		return json.NewEncoder(command.OutOrStdout()).Encode(view)
	}
	_, err = fmt.Fprintf(command.OutOrStdout(), "environment: %s\napi_key: %s\n", view.Environment, view.APIKey)
	return err
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

func prompt(reader *bufio.Reader, output io.Writer, label string) (string, error) {
	if _, err := fmt.Fprintf(output, "%s: ", label); err != nil {
		return "", err
	}
	value, err := reader.ReadString('\n')
	if err != nil && len(value) == 0 {
		return "", fmt.Errorf("read %s: %w", strings.ToLower(label), err)
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("%s cannot be empty", strings.ToLower(label))
	}
	return value, nil
}

func redact(value string) string {
	if len(value) <= 4 {
		return strings.Repeat("*", len(value))
	}
	visible := value[len(value)-4:]
	masked := len(value) - 4
	if masked < 4 {
		masked = 4
	}
	return strings.Repeat("*", masked) + visible
}
