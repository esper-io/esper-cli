package runtime

import "testing"

func TestResolveCredentialsPrecedence(t *testing.T) {
	t.Setenv(EnvironmentVariable, "environment-variable")
	t.Setenv(APIKeyVariable, "environment-key")

	tests := []struct {
		name        string
		environment string
		apiKey      string
		wantEnv     string
		wantKey     string
	}{
		{name: "explicit", environment: "explicit", apiKey: "explicit-key", wantEnv: "explicit", wantKey: "explicit-key"},
		{name: "environment", wantEnv: "environment-variable", wantKey: "environment-key"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			credentials, err := ResolveCredentials(
				Config{Environment: "config", APIKey: "config-key", EnterpriseID: "enterprise"},
				test.environment,
				test.apiKey,
			)
			if err != nil {
				t.Fatalf("ResolveCredentials() error = %v", err)
			}
			if credentials.Environment != test.wantEnv || credentials.APIKey != test.wantKey {
				t.Fatalf("ResolveCredentials() = %#v", credentials)
			}
		})
	}
}

func TestResolveCredentialsRequiresConfiguration(t *testing.T) {
	t.Setenv(EnvironmentVariable, "")
	t.Setenv(APIKeyVariable, "")
	_, err := ResolveCredentials(Config{}, "", "")
	if err == nil || ExitCode(err) != 3 {
		t.Fatalf("ResolveCredentials() error = %v, exit = %d", err, ExitCode(err))
	}
}

func TestCredentialsBaseURL(t *testing.T) {
	tests := []struct {
		environment string
		want        string
	}{
		{environment: "develop", want: "https://develop-api.esper.cloud/api"},
		{environment: "http://localhost:8000/api/", want: "http://localhost:8000/api"},
	}
	for _, test := range tests {
		if got := (Credentials{Environment: test.environment}).BaseURL(); got != test.want {
			t.Fatalf("BaseURL(%q) = %q, want %q", test.environment, got, test.want)
		}
	}
}
