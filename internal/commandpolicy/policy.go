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

// Excluded removes reviewed operations from the generated command surface.
// Keep this list explicit so other public routes remain generated.
func Excluded(method, path string) bool {
	switch path {
	case "/v0/enterprise/{enterprise_id}/content/remote-file/generate_download_url/":
		return method == "POST" // Unavailable on the deployed API; see .spec/retire-download-generate.
	case "/enterprise/{enterprise_id}/application/":
		return method == "GET"
	case "/enterprise/{enterprise_id}/application/{application_id}/":
		return method == "GET" || method == "DELETE"
	case "/enterprise/{enterprise_id}/application/{application_id}/version/":
		return method == "GET"
	case "/enterprise/{enterprise_id}/application/{application_id}/version/{version_id}/":
		return method == "GET" || method == "DELETE" || method == "PATCH"
	case "/enterprise/{enterprise_id}/device/", "/enterprise/{enterprise_id}/device/{device_id}/", "/enterprise/{enterprise_id}/device/{device_id}/install/", "/enterprise/{enterprise_id}/device/{device_id}/download/eventfeed/", "/enterprise/{enterprise_id}/group/{group_id}/download/eventfeed/", "/user/":
		return method == "GET" || (path == "/user/" && method == "POST")
	case "/user/{user_id}/":
		return method == "PUT" || method == "PATCH" || method == "DELETE"
	case "/enterprise/{enterprise_id}/devicegroup/{group_id}/blueprint/":
		return method == "GET" || method == "POST"
	case "/enterprise/{enterprise_id}/devicegroup/{group_id}/blueprint/{blueprint_id}/":
		return method == "GET" || method == "PATCH" || method == "DELETE"
	case "/enterprise/{enterprise_id}/devicegroup/{group_id}/blueprint/{blueprint_id}/revisions/", "/enterprise/{enterprise_id}/devicegroup/{group_id}/blueprint/{blueprint_id}/revisions/{revision_id}/":
		return method == "GET"
	case "/enterprise/{enterprise_id}/devicegroup/{group_id}/blueprint/restore/", "/enterprise/{enterprise_id}/devicegroup/{group_id}/blueprint/upload/":
		return method == "POST"
	case "/v0/enterprise/{enterprise_id}/geofence/":
		return method == "GET"
	case "/v0/enterprise/{enterprise_id}/geofence/{geofence_id}/":
		return method == "GET" || method == "PUT" || method == "PATCH" || method == "DELETE"
	case "/v0/enterprise/{enterprise_id}/create-apply-geo-fence/":
		return method == "GET" || method == "POST"
	case "/v0/enterprise/{enterprise_id}/create-apply-geofence/{geofence_id}/":
		return method == "GET" || method == "DELETE"
	case "/v1/foundry/builds/", "/v1/foundry/device-models/", "/v1/foundry/events/":
		return method == "GET"
	case "/v1/foundry/builds/{build_id}/":
		return method == "GET" || method == "PUT"
	case "/v1/foundry/device-models/{device_model_id}/":
		return method == "PUT"
	}
	return false
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
