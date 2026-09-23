package cmd

import (
	"strings"
	"testing"

	"github.com/esper-io/esper-cli/internal/cmd/generated"
)

func TestV1GeofenceOperationsRemainGenerated(t *testing.T) {
	want := map[string]bool{
		"geofence_createGeofence":    true,
		"geofence_listGeofences":     true,
		"geofence_getDeviceSummary":  true,
		"geofence_getBlueprintUsage": true,
		"geofence_getDevices":        true,
	}
	for _, operation := range generated.Operations() {
		if want[operation.OperationID] {
			delete(want, operation.OperationID)
		}
	}
	for operationID := range want {
		t.Errorf("current v1 geofence operation is missing: %s", operationID)
	}

	root := NewRootCommand()
	for _, path := range []string{
		"geofence create", "geofence list", "geofence-device-summary get", "geofence-blueprint list", "geofence-device list",
	} {
		command, _, err := root.Find(strings.Fields(path))
		if err != nil || command.CommandPath() != "espercli "+path {
			t.Errorf("current geofence command %q not reachable: %v", path, err)
		}
	}
}
