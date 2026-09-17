package configure

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	esperruntime "github.com/esper-io/esper-cli/internal/runtime"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

type configView struct {
	TenantName   string `json:"tenant_name"`
	EnterpriseID string `json:"enterprise_id"`
	APIKey       string `json:"api_key"`
}

func NewCommand(options *esperruntime.GlobalOptions) *cobra.Command {
	command := &cobra.Command{
		Use:   "configure",
		Short: "Configure the Esper tenant name, enterprise ID, and API key",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			return runConfigure(command, options)
		},
	}
	command.AddCommand(&cobra.Command{
		Use:   "show",
		Short: "Show the configured tenant name, enterprise ID, and redacted API key",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			return runShow(command, options)
		},
	})
	return command
}

func runConfigure(command *cobra.Command, options *esperruntime.GlobalOptions) error {
	tenantName := strings.TrimSpace(options.Environment)
	apiKey := strings.TrimSpace(options.APIKey)
	input := command.InOrStdin()
	reader := bufio.NewReader(input)
	var err error
	if tenantName == "" {
		tenantName, err = prompt(reader, command.ErrOrStderr(), "Tenant name")
		if err != nil {
			return esperruntime.NewError(esperruntime.CategoryUsage, err)
		}
	}
	enterpriseID, err := prompt(reader, command.ErrOrStderr(), "Enterprise ID")
	if err != nil {
		return esperruntime.NewError(esperruntime.CategoryUsage, err)
	}
	if apiKey == "" {
		apiKey, err = promptAPIKey(input, reader, command.ErrOrStderr())
		if err != nil {
			return esperruntime.NewError(esperruntime.CategoryUsage, err)
		}
	}

	store, state, err := loadState()
	if err != nil {
		return err
	}
	state.Config.Environment = tenantName
	state.Config.EnterpriseID = enterpriseID
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
	view := configView{TenantName: state.Config.Environment, EnterpriseID: state.Config.EnterpriseID, APIKey: redact(state.Config.APIKey)}
	if options.JSON {
		return json.NewEncoder(command.OutOrStdout()).Encode(view)
	}
	_, err = fmt.Fprintf(command.OutOrStdout(), "tenant_name: %s\nenterprise_id: %s\napi_key: %s\n", view.TenantName, view.EnterpriseID, view.APIKey)
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

func promptAPIKey(input io.Reader, reader *bufio.Reader, output io.Writer) (string, error) {
	file, ok := input.(*os.File)
	if !ok || !term.IsTerminal(int(file.Fd())) {
		return prompt(reader, output, "API key")
	}
	if _, err := fmt.Fprint(output, "API key: "); err != nil {
		return "", err
	}
	value, err := term.ReadPassword(int(file.Fd()))
	if _, writeErr := fmt.Fprintln(output); writeErr != nil {
		return "", writeErr
	}
	if err != nil {
		return "", fmt.Errorf("read API key: %w", err)
	}
	return requiredPromptValue(string(value), "API key")
}

func prompt(reader *bufio.Reader, output io.Writer, label string) (string, error) {
	if _, err := fmt.Fprintf(output, "%s: ", label); err != nil {
		return "", err
	}
	value, err := reader.ReadString('\n')
	if err != nil && len(value) == 0 {
		return "", fmt.Errorf("read %s: %w", strings.ToLower(label), err)
	}
	return requiredPromptValue(value, label)
}

func requiredPromptValue(value, label string) (string, error) {
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
