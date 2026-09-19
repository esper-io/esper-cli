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

func TestRetiredInstallationListUsesOnlyNewerCommand(t *testing.T) {
	root := NewRootCommand()
	command, _, err := root.Find([]string{"api", "legacy", "installdevice", "list"})
	if err == nil && command.CommandPath() == "espercli api legacy installdevice list" {
		t.Fatal("retired legacy installation-list command is still exposed")
	}
	for _, path := range []string{"installdevice list", "device-eventfeed list", "event-feed list", "api legacy device get", "device get"} {
		command, _, err := root.Find(strings.Fields(path))
		if err != nil || command.CommandPath() != "espercli "+path {
			t.Errorf("retained command %q not reachable: %v", path, err)
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
