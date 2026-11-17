package approval

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	esperruntime "github.com/esper-io/esper-cli/internal/runtime"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func newWindowCommand(options *esperruntime.GlobalOptions) *cobra.Command {
	window := &cobra.Command{Use: "window", Short: "Inspect or stop the shared 15-minute approval window"}
	for _, action := range []string{"status", "stop"} {
		short := "Show the approval window status"
		if action == "stop" {
			short = "Stop the approval window"
		}
		window.AddCommand(&cobra.Command{
			Use: action, Short: short, Args: cobra.NoArgs,
			RunE: func(command *cobra.Command, _ []string) error {
				store, err := esperruntime.NewApprovalStore()
				if err != nil {
					return esperruntime.NewError(esperruntime.CategoryAuth, err)
				}
				var status esperruntime.ApprovalWindowStatus
				if action == "stop" {
					status, err = store.StopWindow()
				} else {
					status, err = store.WindowStatus()
				}
				if err != nil {
					return esperruntime.NewError(esperruntime.CategoryAuth, err)
				}
				return writeWindow(command.OutOrStdout(), options.JSON, status)
			},
		})
	}
	return window
}

// StartWindow is only exposed by the standalone root flag, never by agent mode.
func StartWindow(command *cobra.Command, options *esperruntime.GlobalOptions) error {
	file, ok := command.InOrStdin().(*os.File)
	if !ok || !term.IsTerminal(int(file.Fd())) {
		return esperruntime.NewError(esperruntime.CategoryUsage, fmt.Errorf("--dangerously-skip-permissions must be run by a person from an interactive terminal"))
	}
	store, err := esperruntime.NewApprovalStore()
	if err != nil {
		return esperruntime.NewError(esperruntime.CategoryAuth, err)
	}
	status, err := store.WindowStatus()
	if err != nil {
		return esperruntime.NewError(esperruntime.CategoryAuth, err)
	}
	if !status.Active {
		if err := confirmWindow(command.InOrStdin(), command.ErrOrStderr()); err != nil {
			return esperruntime.NewError(esperruntime.CategoryUsage, err)
		}
		status, err = store.StartWindow()
		if err != nil {
			return esperruntime.NewError(esperruntime.CategoryAuth, err)
		}
	}
	return writeWindow(command.OutOrStdout(), options.JSON, status)
}

func confirmWindow(reader io.Reader, writer io.Writer) error {
	const phrase = "start unrestricted window"
	if _, err := fmt.Fprintf(writer, "WARNING: For 15 minutes, skip all API-write approvals and destructive confirmations across ALL tenants using this local approval store, including Secure ADB. Credentials, required inputs, and API permissions still apply.\nType %q to continue: ", phrase); err != nil {
		return err
	}
	answer, err := bufio.NewReader(reader).ReadString('\n')
	if err != nil {
		return fmt.Errorf("read window confirmation: %w", err)
	}
	if strings.TrimSpace(answer) != phrase {
		return fmt.Errorf("approval window cancelled")
	}
	return nil
}

func writeWindow(writer io.Writer, asJSON bool, status esperruntime.ApprovalWindowStatus) error {
	if asJSON {
		return json.NewEncoder(writer).Encode(status)
	}
	state := "inactive"
	if status.Active {
		state = "active"
	}
	if _, err := fmt.Fprintf(writer, "Approval window: %s\n", state); err != nil {
		return err
	}
	if status.Window != nil {
		_, err := fmt.Fprintf(writer, "  Started: %s\n  Expires: %s\n", status.Window.StartedAt.Format(time.RFC3339Nano), status.Window.ExpiresAt.Format(time.RFC3339Nano))
		return err
	}
	return nil
}
