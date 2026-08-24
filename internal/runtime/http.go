package runtime

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type HTTPClient struct {
	BaseURL   string
	APIKey    string
	Client    *http.Client
	Retry     RetryPolicy
	UserAgent string
}

type APIError struct {
	StatusCode int
	Body       []byte
}

func (err *APIError) Error() string {
	message := strings.TrimSpace(string(err.Body))
	if message == "" {
		message = http.StatusText(err.StatusCode)
	}
	return fmt.Sprintf("HTTP %d: %s", err.StatusCode, message)
}

func NewHTTPClient(credentials Credentials) *HTTPClient {
	return &HTTPClient{
		BaseURL:   credentials.BaseURL(),
		APIKey:    credentials.APIKey,
		Client:    &http.Client{Timeout: 30 * time.Second},
		Retry:     DefaultRetryPolicy(),
		UserAgent: "espercli/2",
	}
}

func (client *HTTPClient) Do(ctx context.Context, method, requestPath string, query url.Values, body []byte) ([]byte, error) {
	return client.DoWithContentType(ctx, method, requestPath, query, body, "application/json")
}

func (client *HTTPClient) DoWithContentType(ctx context.Context, method, requestPath string, query url.Values, body []byte, contentType string) ([]byte, error) {
	base := strings.TrimRight(client.BaseURL, "/")
	requestURL := base + "/" + strings.TrimLeft(requestPath, "/")
	if len(query) > 0 {
		requestURL += "?" + query.Encode()
	}
	attempts := client.Retry.MaxAttempts
	if attempts < 1 {
		attempts = 1
	}

	for attempt := 0; attempt < attempts; attempt++ {
		request, err := http.NewRequestWithContext(ctx, method, requestURL, bytes.NewReader(body))
		if err != nil {
			return nil, NewError(CategoryUsage, fmt.Errorf("build request: %w", err))
		}
		Authorize(request, client.APIKey)
		request.Header.Set("Accept", "application/json")
		request.Header.Set("User-Agent", client.UserAgent)
		if len(body) > 0 {
			request.Header.Set("Content-Type", contentType)
		}

		response, err := client.Client.Do(request)
		if err != nil {
			if attempt+1 < attempts && retryableMethod(method) {
				if sleepErr := client.Retry.Sleep(ctx, client.Retry.delay(attempt, nil)); sleepErr != nil {
					return nil, NewError(CategoryNetwork, sleepErr)
				}
				continue
			}
			return nil, NewError(CategoryNetwork, fmt.Errorf("send request: %w", err))
		}
		responseBody, readErr := io.ReadAll(response.Body)
		response.Body.Close()
		if readErr != nil {
			return nil, NewError(CategoryNetwork, fmt.Errorf("read response: %w", readErr))
		}
		if response.StatusCode >= 200 && response.StatusCode < 300 {
			return responseBody, nil
		}
		if attempt+1 < attempts && retryableMethod(method) && retryableStatus(response.StatusCode) {
			if err := client.Retry.Sleep(ctx, client.Retry.delay(attempt, response)); err != nil {
				return nil, NewError(CategoryNetwork, err)
			}
			continue
		}
		return nil, NewError(CategoryAPI, &APIError{StatusCode: response.StatusCode, Body: responseBody})
	}
	return nil, NewError(CategoryNetwork, fmt.Errorf("request attempts exhausted"))
}

func EncodeBody(value any) ([]byte, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("encode request body: %w", err)
	}
	return data, nil
}
