package runtime

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestApprovalWindowFixedLifetime(t *testing.T) {
	now := time.Now().UTC()
	store := &ApprovalStore{Path: filepath.Join(t.TempDir(), "approvals.json"), Now: func() time.Time { return now }}
	spec := ApprovalSpec{BaseURL: "https://first.test", Method: "DELETE", Path: "/things/1"}
	request, _, err := store.Request(spec)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Approve(request.ID); err != nil {
		t.Fatal(err)
	}
	first, err := store.StartWindow()
	if err != nil || !first.Active || first.Window.ExpiresAt.Sub(first.Window.StartedAt) != 15*time.Minute {
		t.Fatalf("start = %+v, %v", first, err)
	}
	for _, offset := range []time.Duration{time.Minute, 15*time.Minute - time.Nanosecond} {
		now = first.Window.StartedAt.Add(offset)
		repeated, err := store.StartWindow()
		if err != nil || !repeated.Active || *repeated.Window != *first.Window {
			t.Fatalf("repeat renewed window: %+v, %v", repeated, err)
		}
		status, err := store.WindowStatus()
		if err != nil || !status.Active || *status.Window != *first.Window {
			t.Fatalf("status = %+v, %v", status, err)
		}
	}
	now = first.Window.ExpiresAt
	status, err := store.WindowStatus()
	if err != nil || status.Active {
		t.Fatalf("window active at expiry: %+v, %v", status, err)
	}
	for range 2 {
		if status, err = store.StopWindow(); err != nil || status.Active || status.Window != nil {
			t.Fatalf("stop = %+v, %v", status, err)
		}
	}
	// Window operations must not prune, consume, or otherwise change requests.
	document, err := store.load()
	if err != nil || len(document.Requests) != 1 || !document.Requests[0].ConsumedAt.IsZero() {
		t.Fatalf("requests changed: %+v, %v", document, err)
	}
	info, err := os.Stat(store.Path)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("store permissions = %v, %v", info, err)
	}
}

func TestApprovalModesPreservePendingRequestsAndWindow(t *testing.T) {
	t.Setenv(CredentialsFileEnvironment, filepath.Join(t.TempDir(), "creds.json"))
	store, err := NewApprovalStore()
	if err != nil {
		t.Fatal(err)
	}
	spec := ApprovalSpec{BaseURL: "https://first.test", Method: "POST", Path: "/things"}
	request, _, err := store.Request(spec)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Approve(request.ID); err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"agent", "window"} {
		if mode == "window" {
			if _, err := store.StartWindow(); err != nil {
				t.Fatal(err)
			}
		}
		before, _ := os.ReadFile(store.Path)
		for _, baseURL := range []string{spec.BaseURL, "https://other-tenant.test"} {
			input := spec
			input.BaseURL = baseURL
			approvalMode := "agent"
			if mode == "window" {
				approvalMode = "request"
			}
			decision, err := RequireApproval(input, approvalMode)
			if err != nil || !decision.Approved || !decision.SkipDestructiveConfirmation {
				t.Fatalf("%s decision = %+v, %v", mode, decision, err)
			}
		}
		// Agent mode while a window is active must not extend it either.
		if _, err := RequireApproval(spec, "agent"); err != nil {
			t.Fatal(err)
		}
		after, _ := os.ReadFile(store.Path)
		if !bytes.Equal(before, after) {
			t.Fatalf("%s changed persisted permission", mode)
		}
	}
	if _, err := store.StopWindow(); err != nil {
		t.Fatal(err)
	}
	decision, err := RequireApproval(spec, "request")
	if err != nil || !decision.Approved || decision.SkipDestructiveConfirmation || decision.Request.ID != request.ID {
		t.Fatalf("normal approval after stop = %+v, %v", decision, err)
	}
	decision, err = RequireApproval(spec, "request")
	if err != nil || decision.Approved {
		t.Fatalf("approval reused: %+v, %v", decision, err)
	}
}

func TestAgentApprovalDoesNotPersistOrAuthorizeNextInvocation(t *testing.T) {
	t.Setenv(CredentialsFileEnvironment, filepath.Join(t.TempDir(), "creds.json"))
	spec := ApprovalSpec{Method: "POST", Path: "/things"}
	decision, err := RequireApproval(spec, "agent")
	if err != nil || !decision.Approved {
		t.Fatalf("agent = %+v, %v", decision, err)
	}
	path, _ := DefaultApprovalPath()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("agent persisted approvals: %v", err)
	}
	if _, err := RequireApproval(spec, "invalid"); err == nil {
		t.Fatal("invalid mode accepted")
	}
	decision, err = RequireApproval(spec, "")
	if err != nil || decision.Approved {
		t.Fatalf("unflagged invocation = %+v, %v", decision, err)
	}
	// Explicitly selecting another store does not inherit the first store's window.
	store, _ := NewApprovalStore()
	if _, err := store.StartWindow(); err != nil {
		t.Fatal(err)
	}
	t.Setenv(CredentialsFileEnvironment, filepath.Join(t.TempDir(), "creds.json"))
	decision, err = RequireApproval(spec, "request")
	if err != nil || decision.Approved {
		t.Fatalf("separate store inherited window: %+v, %v", decision, err)
	}
}

func TestApprovalWindowInvalidStateNeverAuthorizes(t *testing.T) {
	now := time.Now().UTC()
	for _, data := range []string{
		`{"requests":[]}`, `{`, `{"window":{"started_at":"bad"}}`,
		fmt.Sprintf(`{"window":{"started_at":%q,"expires_at":%q}}`, now.Add(time.Minute).Format(time.RFC3339Nano), now.Add(16*time.Minute).Format(time.RFC3339Nano)),
		fmt.Sprintf(`{"window":{"started_at":%q,"expires_at":%q}}`, now.Format(time.RFC3339Nano), now.Add(time.Hour).Format(time.RFC3339Nano)),
		`{"window":{"started_at":"0001-01-01T00:00:00Z","expires_at":"0001-01-01T00:15:00Z"}}`,
	} {
		t.Run(data, func(t *testing.T) {
			t.Setenv(CredentialsFileEnvironment, filepath.Join(t.TempDir(), "creds.json"))
			store, _ := NewApprovalStore()
			if err := os.WriteFile(store.Path, []byte(data), 0o600); err != nil {
				t.Fatal(err)
			}
			decision, _ := RequireApproval(ApprovalSpec{Method: "DELETE", Path: "/thing"}, "request")
			if decision.Approved || decision.SkipDestructiveConfirmation {
				t.Fatalf("invalid state authorized: %+v", decision)
			}
		})
	}
}

func TestExpiredOrUnreadableWindowDoesNotAuthorize(t *testing.T) {
	for _, unreadable := range []bool{false, true} {
		t.Run(fmt.Sprintf("unreadable=%v", unreadable), func(t *testing.T) {
			t.Setenv(CredentialsFileEnvironment, filepath.Join(t.TempDir(), "creds.json"))
			store, _ := NewApprovalStore()
			if unreadable {
				if err := os.Mkdir(store.Path, 0o700); err != nil {
					t.Fatal(err)
				}
			} else {
				store.Now = func() time.Time { return time.Now().Add(-16 * time.Minute) }
				if _, err := store.StartWindow(); err != nil {
					t.Fatal(err)
				}
			}
			decision, err := RequireApproval(ApprovalSpec{Method: "DELETE", Path: "/things/1"}, "request")
			if decision.Approved || decision.SkipDestructiveConfirmation || (err != nil) != unreadable {
				t.Fatalf("decision = %+v, error = %v", decision, err)
			}
		})
	}
}

func TestConcurrentWindowAndRequestsPreserveBoth(t *testing.T) {
	store := &ApprovalStore{Path: filepath.Join(t.TempDir(), "approvals.json")}
	var workers sync.WaitGroup
	for index := range 8 {
		workers.Add(1)
		go func() {
			defer workers.Done()
			if _, err := store.StartWindow(); err != nil {
				t.Error(err)
			}
			if _, _, err := store.Request(ApprovalSpec{Method: "POST", Path: fmt.Sprintf("/things/%d", index)}); err != nil {
				t.Error(err)
			}
		}()
	}
	workers.Wait()
	document, err := store.load()
	if err != nil || document.Window == nil || len(document.Requests) != 8 {
		t.Fatalf("concurrent updates lost: %+v, %v", document, err)
	}
}
