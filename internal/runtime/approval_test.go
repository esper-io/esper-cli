package runtime

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestApprovalStoreRequiresExactSingleUseApproval(t *testing.T) {
	now := time.Date(2026, time.August, 26, 12, 0, 0, 0, time.UTC)
	store := &ApprovalStore{Path: filepath.Join(t.TempDir(), "approvals.json"), Now: func() time.Time { return now }}
	spec := ApprovalSpec{BaseURL: "https://example.test/api", Method: "POST", Path: "/v2/things", Query: map[string][]string{"device": {"device-1"}}, Body: []byte(`{"token":"secret","name":"fixture"}`)}
	request, created, err := store.Request(spec)
	if err != nil || !created {
		t.Fatalf("Request() = %#v, %v, want new request", request, err)
	}
	data, err := os.ReadFile(store.Path)
	if err != nil || strings.Contains(string(data), "secret") || strings.Contains(string(data), "device-1") {
		t.Fatalf("approval file leaked request values: %q, %v", data, err)
	}
	if info, err := os.Stat(store.Path); err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("approval permissions = %v, %v", info, err)
	}
	if _, err := store.Approve(request.ID); err != nil {
		t.Fatal(err)
	}
	if _, approved, err := store.Consume(spec); err != nil || !approved {
		t.Fatalf("Consume() = approved=%v, err=%v", approved, err)
	}
	if _, approved, err := store.Consume(spec); err != nil || approved {
		t.Fatalf("second Consume() = approved=%v, err=%v", approved, err)
	}
	changed := spec
	changed.Body = []byte(`{"token":"secret","name":"changed"}`)
	second, created, err := store.Request(changed)
	if err != nil || !created || second.ID == request.ID {
		t.Fatalf("changed Request() = %#v, created=%v, err=%v", second, created, err)
	}
	changed = spec
	changed.Query = map[string][]string{"device": {"device-2"}}
	third, created, err := store.Request(changed)
	if err != nil || !created || third.ID == request.ID || third.ID == second.ID {
		t.Fatalf("query-changed Request() = %#v, created=%v, err=%v", third, created, err)
	}
}

func TestApprovalStoreExpiresAndConsumesOnceUnderContention(t *testing.T) {
	now := time.Date(2026, time.August, 26, 12, 0, 0, 0, time.UTC)
	store := &ApprovalStore{Path: filepath.Join(t.TempDir(), "approvals.json"), Now: func() time.Time { return now }}
	spec := ApprovalSpec{BaseURL: "https://example.test", Method: "PATCH", Path: "/things/1"}
	request, _, err := store.Request(spec)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Approve(request.ID); err != nil {
		t.Fatal(err)
	}
	var group sync.WaitGroup
	approved := 0
	var mutex sync.Mutex
	for range 8 {
		group.Add(1)
		go func() {
			defer group.Done()
			_, ok, err := store.Consume(spec)
			if err != nil {
				t.Errorf("Consume() error = %v", err)
				return
			}
			if ok {
				mutex.Lock()
				approved++
				mutex.Unlock()
			}
		}()
	}
	group.Wait()
	if approved != 1 {
		t.Fatalf("approved consumes = %d, want 1", approved)
	}
	now = now.Add(approvalLifetime + time.Second)
	if _, err := store.Show(request.ID); err == nil {
		t.Fatal("expired approval remained visible")
	}
}

func TestApprovalStoreShowDoesNotRewriteUnchangedDocument(t *testing.T) {
	store := &ApprovalStore{Path: filepath.Join(t.TempDir(), "approvals.json")}
	request, _, err := store.Request(ApprovalSpec{BaseURL: "https://example.test", Method: "POST", Path: "/things"})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(store.Path, 0o640); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Show(request.ID); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(store.Path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o640 {
		t.Fatalf("Show() rewrote approval file permissions to %o", info.Mode().Perm())
	}
}

func TestRequireTerminalRejectsPipes(t *testing.T) {
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	defer writer.Close()
	if err := RequireTerminal(reader); err == nil {
		t.Fatal("pipe was accepted as terminal")
	}
}
