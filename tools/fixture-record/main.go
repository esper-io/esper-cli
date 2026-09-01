package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

const redacted = "REDACTED"

type Recording struct {
	Request  RecordedRequest  `json:"request"`
	Response RecordedResponse `json:"response"`
}
type RecordedRequest struct {
	Method  string          `json:"method"`
	URL     string          `json:"url"`
	Headers http.Header     `json:"headers,omitempty"`
	Body    json.RawMessage `json:"body,omitempty"`
}
type RecordedResponse struct {
	Status  int             `json:"status"`
	Headers http.Header     `json:"headers,omitempty"`
	Body    json.RawMessage `json:"body,omitempty"`
}
type Options struct {
	Method, URL, Body string
	Headers           http.Header
}

type headers []string

func (value *headers) String() string          { return strings.Join(*value, ",") }
func (value *headers) Set(header string) error { *value = append(*value, header); return nil }

func main() {
	method := flag.String("method", http.MethodGet, "HTTP method")
	requestURL := flag.String("url", "", "endpoint URL")
	body := flag.String("body", "", "inline JSON request body, @path, or - for stdin")
	output := flag.String("output", "", "fixture output path")
	var rawHeaders headers
	flag.Var(&rawHeaders, "header", "request header, repeatable: Name: value")
	flag.Parse()
	if *requestURL == "" || *output == "" {
		fmt.Fprintln(os.Stderr, "usage: fixture-record --url URL --output PATH [--method METHOD] [--body VALUE] [--header NAME:VALUE]")
		os.Exit(2)
	}
	headerMap, err := parseHeaders(rawHeaders)
	if err != nil {
		fatal(err)
	}
	recording, err := Record(http.DefaultClient, Options{Method: *method, URL: *requestURL, Body: *body, Headers: headerMap}, os.Stdin)
	if err != nil {
		fatal(err)
	}
	data, err := json.MarshalIndent(recording, "", "  ")
	if err != nil {
		fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(*output), 0o755); err != nil {
		fatal(err)
	}
	if err := os.WriteFile(*output, append(data, '\n'), 0o600); err != nil {
		fatal(err)
	}
}

func Record(client *http.Client, options Options, stdin io.Reader) (Recording, error) {
	body, err := readBody(options.Body, stdin)
	if err != nil {
		return Recording{}, err
	}
	request, err := http.NewRequest(options.Method, options.URL, bytes.NewReader(body))
	if err != nil {
		return Recording{}, fmt.Errorf("build request: %w", err)
	}
	request.Header = options.Headers.Clone()
	if len(body) > 0 && request.Header.Get("Content-Type") == "" {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := client.Do(request)
	if err != nil {
		return Recording{}, fmt.Errorf("send request: %w", err)
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return Recording{}, fmt.Errorf("read response: %w", err)
	}
	return Recording{
		Request:  RecordedRequest{Method: request.Method, URL: redactURL(request.URL), Headers: redactHeaders(request.Header), Body: redactJSON(body)},
		Response: RecordedResponse{Status: response.StatusCode, Headers: redactHeaders(response.Header), Body: redactJSON(responseBody)},
	}, nil
}

func readBody(value string, stdin io.Reader) ([]byte, error) {
	if value == "" {
		return nil, nil
	}
	if value == "-" {
		data, err := io.ReadAll(stdin)
		return data, err
	}
	if strings.HasPrefix(value, "@") {
		return os.ReadFile(strings.TrimPrefix(value, "@"))
	}
	return []byte(value), nil
}

func parseHeaders(values []string) (http.Header, error) {
	headers := make(http.Header)
	for _, value := range values {
		name, headerValue, ok := strings.Cut(value, ":")
		if !ok || strings.TrimSpace(name) == "" {
			return nil, fmt.Errorf("invalid header %q", value)
		}
		headers.Add(strings.TrimSpace(name), strings.TrimSpace(headerValue))
	}
	return headers, nil
}

func redactHeaders(headers http.Header) http.Header {
	result := headers.Clone()
	for name := range result {
		if sensitiveKey(name) {
			result.Set(name, redacted)
		}
	}
	return result
}

func redactURL(value *url.URL) string {
	copy := *value
	query := copy.Query()
	for name := range query {
		if sensitiveKey(name) {
			query.Set(name, redacted)
		}
	}
	copy.RawQuery = query.Encode()
	return copy.String()
}

func redactJSON(data []byte) json.RawMessage {
	if len(data) == 0 {
		return nil
	}
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return json.RawMessage(data)
	}
	redactValue(value, "")
	result, err := json.Marshal(value)
	if err != nil {
		return json.RawMessage(data)
	}
	return result
}

func redactValue(value any, key string) {
	switch typed := value.(type) {
	case map[string]any:
		for name, child := range typed {
			if sensitiveKey(name) {
				typed[name] = redacted
				continue
			}
			redactValue(child, name)
		}
	case []any:
		for _, child := range typed {
			redactValue(child, key)
		}
	}
}

func sensitiveKey(key string) bool {
	key = strings.ToLower(strings.ReplaceAll(key, "-", "_"))
	return strings.Contains(key, "token") || strings.Contains(key, "api_key") || strings.Contains(key, "authorization") || strings.Contains(key, "enterprise_id") || key == "enterprise"
}
func fatal(err error) { fmt.Fprintln(os.Stderr, "error:", err); os.Exit(1) }
