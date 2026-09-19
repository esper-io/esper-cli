package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestIOSCommandsAreSeparatedFromCrossPlatformCommands(t *testing.T) {
	root := NewRootCommand()
	for _, path := range []string{
		"ios-app list", "ios-itunesapp list",
		"ios-webclip create", "ios-webclip get", "ios-webclip list", "ios-webclip delete",
		"ios-provisioning-profile create", "ios-provisioning-profile get", "ios-provisioning-profile list",
		"ios-provisioning-profile-version get", "ios-provisioning-profile-version list", "ios-provisioning-profile-version delete",
		"tenant-app create", "tenant-app list", "tenant-app-version get",
		"device-app list", "seamless create", "seamless upload", "api legacy app list",
	} {
		command, _, err := root.Find(strings.Fields(path))
		if err != nil || command.CommandPath() != "espercli "+path {
			t.Errorf("command %q not reachable: %v", path, err)
		}
	}
	for _, path := range []string{
		"app list", "itunesapp list", "webclip list", "provisioning-profile get",
		"provisioning-profile-version get", "ios-tenant-app list", "ios-seamless create",
	} {
		command, _, err := root.Find(strings.Fields(path))
		if err == nil && command.CommandPath() == "espercli "+path {
			t.Errorf("unexpected command %q", path)
		}
	}
	versions, _, err := root.Find([]string{"version", "list"})
	if err != nil {
		t.Fatal(err)
	}
	if versions.Flags().Lookup("provisioning-profile") != nil {
		t.Fatal("mixed version list still exposes the iOS provisioning-profile route")
	}
	for _, scope := range []string{"blueprint", "tenant-app"} {
		if versions.Flags().Lookup(scope) == nil {
			t.Errorf("mixed version list lost --%s", scope)
		}
	}
}

func TestLegacyCommandsWithNewerEndpointsAreRemoved(t *testing.T) {
	root := NewRootCommand()
	for _, path := range []string{
		"api legacy installdevice list",
		"api legacy application list", "api legacy application get", "api legacy application delete",
		"api legacy app-version get", "api legacy app-version patch", "api legacy app-version delete",
		"api legacy version list", "api legacy device list", "api legacy device get", "api legacy install list",
		"api legacy blueprint list", "api legacy blueprint create", "api legacy blueprint get", "api legacy blueprint delete",
		"device-eventfeed list", "group-eventfeed list",
	} {
		command, _, err := root.Find(strings.Fields(path))
		if err == nil && command.CommandPath() == "espercli "+path {
			t.Errorf("removed command %q is still exposed", path)
		}
	}
	for _, path := range []string{"installdevice list", "user list", "api legacy app list", "api legacy status get", "device-group list", "device-group create"} {
		command, _, err := root.Find(strings.Fields(path))
		if err != nil || command.CommandPath() != "espercli "+path {
			t.Errorf("retained exception %q not reachable: %v", path, err)
		}
	}
}

func TestIOSCatalogHelpDoesNotCallDeviceAppsAnOlderGeneration(t *testing.T) {
	root := NewRootCommand()
	var output bytes.Buffer
	root.SetOut(&output)
	root.SetErr(&output)
	root.SetArgs([]string{"ios-app", "list", "--help"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(output.String(), "api legacy app list") {
		t.Fatal("iOS catalog help incorrectly advertises device inventory as an older version")
	}
}
