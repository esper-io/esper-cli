package runtime

import (
	"fmt"
	"net/http"
	"os"
	"strings"
)

const (
	EnvironmentVariable = "ESPER_ENVIRONMENT"
	APIKeyVariable      = "ESPER_API_KEY"
)

type Credentials struct {
	Environment  string
	APIKey       string
	EnterpriseID string
}

func ResolveCredentials(config Config, environment, apiKey string) (Credentials, error) {
	if environment == "" {
		environment = os.Getenv(EnvironmentVariable)
	}
	if environment == "" {
		environment = config.Environment
	}
	if apiKey == "" {
		apiKey = os.Getenv(APIKeyVariable)
	}
	if apiKey == "" {
		apiKey = config.APIKey
	}
	if environment == "" {
		return Credentials{}, NewError(CategoryAuth, fmt.Errorf("environment is not configured"))
	}
	if apiKey == "" {
		return Credentials{}, NewError(CategoryAuth, fmt.Errorf("API key is not configured"))
	}
	return Credentials{Environment: environment, APIKey: apiKey, EnterpriseID: config.EnterpriseID}, nil
}

func (credentials Credentials) BaseURL() string {
	if strings.HasPrefix(credentials.Environment, "http://") || strings.HasPrefix(credentials.Environment, "https://") {
		return strings.TrimRight(credentials.Environment, "/")
	}
	return fmt.Sprintf("https://%s-api.esper.cloud/api", credentials.Environment)
}

func Authorize(request *http.Request, apiKey string) {
	request.Header.Set("Authorization", "Bearer "+apiKey)
}
