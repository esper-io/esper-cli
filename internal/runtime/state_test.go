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
			Device:     &ActiveResource{ID: "device-1", Name: "kiosk"},
			Group:      &ActiveResource{ID: "group-1", Name: "stores"},
			Enterprise: &ActiveResource{ID: "enterprise-1"},
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

func TestContextResourceForParameter(t *testing.T) {
	tests := []struct {
		parameter string
		resource  string
		ok        bool
	}{
		{parameter: "enterprise_id", resource: ContextEnterprise, ok: true},
		{parameter: "device_id", resource: ContextDevice, ok: true},
		{parameter: "deviceId", resource: ContextDevice, ok: true},
		{parameter: "group_id", resource: ContextGroup, ok: true},
		{parameter: "devicegroup_id", resource: ContextGroup, ok: true},
		{parameter: "application_id", resource: ContextApp, ok: true},
		{parameter: "app_id", resource: ContextApp, ok: true},
		{parameter: "appId", resource: ContextApp, ok: true},
		{parameter: "device", ok: false},
		{parameter: "tenant_id", ok: false},
	}
	for _, test := range tests {
		t.Run(test.parameter, func(t *testing.T) {
			resource, ok := ContextResourceForParameter(test.parameter)
			if resource != test.resource || ok != test.ok {
				t.Fatalf("ContextResourceForParameter(%q) = %q, %v", test.parameter, resource, ok)
			}
		})
	}
}

func TestActiveContextResources(t *testing.T) {
	var active ActiveContext
	for _, resource := range ContextResources {
		value := &ActiveResource{ID: resource + "-1"}
		if err := active.SetResource(resource, value); err != nil {
			t.Fatal(err)
		}
		if got := active.Resource(resource); !reflect.DeepEqual(got, value) {
			t.Fatalf("Resource(%q) = %#v", resource, got)
		}
	}
	if err := active.SetResource("unknown", nil); err == nil {
		t.Fatal("SetResource() accepted an unknown resource")
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
