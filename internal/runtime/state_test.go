package runtime

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestStateStoreRoundTripAndPermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "creds.json")
	store := &StateStore{Path: path}
	want := State{
		Config: Config{Environment: "develop", APIKey: "secret", EnterpriseID: "enterprise-1"},
		Active: ActiveContext{
			Device: &ActiveResource{ID: "device-1", Name: "kiosk"},
			Group:  &ActiveResource{ID: "group-1", Name: "stores"},
		},
	}

	if err := store.Save(want); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	got, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Load() = %#v, want %#v", got, want)
	}

	directoryInfo, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	if permissions := directoryInfo.Mode().Perm(); permissions != 0o700 {
		t.Fatalf("directory permissions = %o, want 700", permissions)
	}
	fileInfo, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if permissions := fileInfo.Mode().Perm(); permissions != 0o600 {
		t.Fatalf("file permissions = %o, want 600", permissions)
	}
}

func TestStateStoreLoadMissing(t *testing.T) {
	store := &StateStore{Path: filepath.Join(t.TempDir(), "missing.json")}
	got, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !reflect.DeepEqual(got, State{}) {
		t.Fatalf("Load() = %#v, want empty state", got)
	}
}

func TestDefaultCredentialsPathOverride(t *testing.T) {
	want := filepath.Join(t.TempDir(), "custom.json")
	t.Setenv(CredentialsFileEnvironment, want)
	got, err := DefaultCredentialsPath()
	if err != nil {
		t.Fatalf("DefaultCredentialsPath() error = %v", err)
	}
	if got != want {
		t.Fatalf("DefaultCredentialsPath() = %q, want %q", got, want)
	}
}
