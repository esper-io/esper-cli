package approval

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	esperruntime "github.com/esper-io/esper-cli/internal/runtime"
	"github.com/spf13/cobra"
)

func NewCommand(options *esperruntime.GlobalOptions) *cobra.Command {
	command := &cobra.Command{Use: "approval", Short: "Review and approve pending API writes"}
	command.AddCommand(&cobra.Command{
		Use:   "show <approval-id>",
		Short: "Show a sanitized pending approval request",
		Args:  cobra.ExactArgs(1),
		RunE: func(command *cobra.Command, args []string) error {
			request, err := load(args[0])
			if err != nil {
				return esperruntime.NewError(esperruntime.CategoryUsage, err)
			}
			return write(command.OutOrStdout(), options.JSON, request)
		},
	})
	command.AddCommand(&cobra.Command{
		Use:   "approve <approval-id>",
		Short: "Approve one pending API write from an interactive terminal",
		Args:  cobra.ExactArgs(1),
		RunE: func(command *cobra.Command, args []string) error {
			if err := esperruntime.RequireTerminal(command.InOrStdin()); err != nil {
				return esperruntime.NewError(esperruntime.CategoryUsage, err)
			}
			request, err := load(args[0])
			if err != nil {
				return esperruntime.NewError(esperruntime.CategoryUsage, err)
			}
			if err := write(command.ErrOrStderr(), false, request); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(command.ErrOrStderr(), "Type approve %s to approve this exact request: ", request.ID); err != nil {
				return err
			}
			answer, err := bufio.NewReader(command.InOrStdin()).ReadString('\n')
			if err != nil && len(answer) == 0 {
				return esperruntime.NewError(esperruntime.CategoryUsage, fmt.Errorf("read approval: %w", err))
			}
			if strings.TrimSpace(answer) != "approve "+request.ID {
				return esperruntime.NewError(esperruntime.CategoryUsage, fmt.Errorf("approval cancelled"))
			}
			store, err := esperruntime.NewApprovalStore()
			if err != nil {
				return esperruntime.NewError(esperruntime.CategoryAuth, err)
			}
			request, err = store.Approve(request.ID)
			if err != nil {
				return esperruntime.NewError(esperruntime.CategoryUsage, err)
			}
			return write(command.OutOrStdout(), options.JSON, request)
		},
	})
	return command
}

func load(id string) (esperruntime.ApprovalRequest, error) {
	store, err := esperruntime.NewApprovalStore()
	if err != nil {
		return esperruntime.ApprovalRequest{}, err
	}
	return store.Show(id)
}

func write(writer io.Writer, asJSON bool, request esperruntime.ApprovalRequest) error {
	if asJSON {
		return json.NewEncoder(writer).Encode(request)
	}
	additional := make([]string, 0, len(request.AdditionalTargets))
	for _, target := range request.AdditionalTargets {
		additional = append(additional, target.Method+" "+target.Path)
	}
	_, err := fmt.Fprintf(writer, "Approval %s\n  %s %s\n  Additional writes: %s\n  Query keys: %s\n  Body fields: %s\n  Body SHA-256: %s\n  Expires: %s\n", request.ID, request.Method, request.Path, display(additional), display(request.QueryKeys), display(request.BodyFields), displayValue(request.BodySHA256), request.ExpiresAt.Format("2006-01-02T15:04:05Z07:00"))
	return err
}

func display(values []string) string {
	if len(values) == 0 {
		return "<none>"
	}
	return strings.Join(values, ", ")
}

func displayValue(value string) string {
	if value == "" {
		return "<none>"
	}
	return value
}
