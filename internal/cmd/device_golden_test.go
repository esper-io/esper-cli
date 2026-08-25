package cmd

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	esperruntime "github.com/esper-io/esper-cli/internal/runtime"
)

func TestDeviceCommandsGoldenFixtures(t *testing.T) {
	tests := []struct {
		name       string
		arguments  []string
		requestURI string
		fixture    string
		golden     string
		status     int
	}{
		{name: "list success", arguments: []string{"device", "list", "--json"}, requestURI: "/v2/devices/", fixture: "list-success.json", golden: "list-success.golden", status: http.StatusOK},
		{name: "get success", arguments: []string{"device", "get", "device-1", "--json"}, requestURI: "/v2/devices/device-1", fixture: "get-success.json", golden: "get-success.golden", status: http.StatusOK},
		{name: "list API error", arguments: []string{"device", "list", "--json"}, requestURI: "/v2/devices/", fixture: "api-error.json", status: http.StatusNotFound},
		{name: "get API error", arguments: []string{"device", "get", "device-1", "--json"}, requestURI: "/v2/devices/device-1", fixture: "api-error.json", status: http.StatusNotFound},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := readDeviceFixture(t, test.fixture)
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				if request.URL.RequestURI() != test.requestURI {
					t.Errorf("request URI = %q, want %q", request.URL.RequestURI(), test.requestURI)
				}
				if request.Header.Get("Authorization") != "Bearer fixture-key" {
					t.Errorf("authorization = %q", request.Header.Get("Authorization"))
				}
				writer.Header().Set("Content-Type", "application/json")
				writer.WriteHeader(test.status)
				_, _ = writer.Write(response)
			}))
			defer server.Close()
			t.Setenv(esperruntime.EnvironmentVariable, server.URL)
			t.Setenv(esperruntime.APIKeyVariable, "fixture-key")

			command := NewRootCommand()
			var output bytes.Buffer
			command.SetOut(&output)
			command.SetErr(&output)
			command.SetArgs(test.arguments)
			err := command.Execute()
			if test.status >= 400 {
				var apiError *esperruntime.APIError
				if !errors.As(err, &apiError) || apiError.StatusCode != test.status {
					t.Fatalf("Execute() error = %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			want := string(readDeviceFixture(t, test.golden))
			if output.String() != want {
				t.Fatalf("output mismatch\nwant:\n%s\ngot:\n%s", want, output.String())
			}
		})
	}
}

func TestDeviceListAllUnwrapsAppsEnvelope(t *testing.T) {
	first := readDeviceFixture(t, "list-all-first-page.json")
	second := readDeviceFixture(t, "list-all-second-page.json")
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/v2/devices/" {
			t.Errorf("request path = %q", request.URL.Path)
		}
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Query().Get("offset") == "1" {
			_, _ = writer.Write(second)
			return
		}
		response := strings.ReplaceAll(string(first), "NEXT_URL", server.URL+"/v2/devices/?offset=1")
		_, _ = writer.Write([]byte(response))
	}))
	defer server.Close()
	t.Setenv(esperruntime.EnvironmentVariable, server.URL)
	t.Setenv(esperruntime.APIKeyVariable, "fixture-key")

	command := NewRootCommand()
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetErr(&output)
	command.SetArgs([]string{"device", "list", "--all", "--json"})
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	want := string(readDeviceFixture(t, "list-all-success.golden"))
	if output.String() != want {
		t.Fatalf("output mismatch\nwant:\n%s\ngot:\n%s", want, output.String())
	}
}

func TestDeviceGetHumanUnwrapsAppsEnvelope(t *testing.T) {
	response := readDeviceFixture(t, "get-success.json")
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write(response)
	}))
	defer server.Close()
	t.Setenv(esperruntime.EnvironmentVariable, server.URL)
	t.Setenv(esperruntime.APIKeyVariable, "fixture-key")

	command := NewRootCommand()
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetErr(&output)
	command.SetArgs([]string{"device", "get", "device-1"})
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	for _, expected := range []string{"id", "device-1", "name", "Kiosk One", "state", "ONLINE"} {
		if !strings.Contains(output.String(), expected) {
			t.Fatalf("human output does not contain %q:\n%s", expected, output.String())
		}
	}
	if strings.Contains(output.String(), "content") || strings.Contains(output.String(), "Device retrieved") {
		t.Fatalf("human output retained response envelope:\n%s", output.String())
	}
}

func readDeviceFixture(t *testing.T, name string) []byte {
	t.Helper()
	path := filepath.Join("..", "..", "spec", "fixtures", "device", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
