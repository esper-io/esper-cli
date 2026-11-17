package runtime

import "fmt"

type GlobalOptions struct {
	ApprovalMode string
	JSON         bool
	Yes          bool
	Verbose      bool
	NoColor      bool
	Environment  string
	APIKey       string
}

func ValidateApprovalMode(mode string) error {
	if mode != "request" && mode != "agent" {
		return fmt.Errorf("invalid --approval-mode %q: must be request or agent", mode)
	}
	return nil
}
