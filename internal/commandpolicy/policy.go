// Package commandpolicy contains reviewed exceptions to generation-based naming.
package commandpolicy

import "strings"

// Replacement reuses the generator's canonical-alias mechanism for an endpoint
// whose newer counterpart has the same contract. See .spec/cli-api-consolidation.
func Replacement(method, path string) string {
	if method == "GET" && path == "/enterprise/{enterprise_id}/application/{application_id}/version/{version_id}/installdevices" {
		return "getInstallDevicesV1"
	}
	return ""
}

// Command names only confirmed iOS routes. Keep the original resource noun and
// generation routing intact, including unrelated commands under api legacy.
func Command(path string, command []string) []string {
	if len(command) != 2 {
		return command
	}
	noun := ""
	switch {
	case path == "/v2/apps":
		noun = "ios-app"
	case path == "/v2/itunesapps":
		noun = "ios-itunesapp"
	case path == "/v2/webclips" || strings.HasPrefix(path, "/v2/webclips/"):
		noun = "ios-webclip"
	case path == "/v2/provisioning-profiles" || strings.HasPrefix(path, "/v2/provisioning-profiles/"):
		noun = "ios-provisioning-profile"
		if strings.Contains(path, "/versions") {
			noun += "-version"
		}
	}
	if noun == "" {
		return command
	}
	return []string{noun, command[1]}
}
