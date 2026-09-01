package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRecordRedactsFixture(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Authorization") != "Bearer live-token" {
			t.Fatalf("authorization = %q", request.Header.Get("Authorization"))
		}
		writer.Header().Set("X-Token", "response-token")
		_, _ = writer.Write([]byte(`{"enterprise_id":"enterprise-live","nested":{"api_key":"key-live"},"value":"kept"}`))
	}))
	defer server.Close()
	recording, err := Record(server.Client(), Options{
		Method:  http.MethodPost,
		URL:     server.URL + "/devices?enterprise_id=enterprise-live",
		Body:    `{"enterprise":"enterprise-live","token":"request-token","name":"kept"}`,
		Headers: http.Header{"Authorization": {"Bearer live-token"}},
	}, strings.NewReader(""))
	if err != nil {
		t.Fatal(err)
	}
	if recording.Request.Headers.Get("Authorization") != redacted {
		t.Fatalf("request headers = %#v", recording.Request.Headers)
	}
	if !strings.Contains(recording.Request.URL, "enterprise_id=REDACTED") {
		t.Fatalf("request URL = %s", recording.Request.URL)
	}
	if string(recording.Request.Body) != `{"enterprise":"REDACTED","name":"kept","token":"REDACTED"}` {
		t.Fatalf("request body = %s", recording.Request.Body)
	}
	if recording.Response.Headers.Get("X-Token") != redacted {
		t.Fatalf("response headers = %#v", recording.Response.Headers)
	}
	if string(recording.Response.Body) != `{"enterprise_id":"REDACTED","nested":{"api_key":"REDACTED"},"value":"kept"}` {
		t.Fatalf("response body = %s", recording.Response.Body)
	}
}

func TestParseHeaders(t *testing.T) {
	headers, err := parseHeaders([]string{"X-Test: value", "Authorization: Bearer token"})
	if err != nil {
		t.Fatal(err)
	}
	if headers.Get("X-Test") != "value" || headers.Get("Authorization") != "Bearer token" {
		t.Fatalf("headers = %#v", headers)
	}
}
