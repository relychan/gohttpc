// Copyright 2026 RelyChan Pte. Ltd
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package gohttpc_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/relychan/gohttpc"
	"github.com/relychan/goutils"
)

// execute makes a GET request and returns both the response and any error.
// For 4xx/5xx responses, the client returns both a non-nil *http.Response and a non-nil error.
func executeGet(t *testing.T, url string) (*http.Response, error) {
	t.Helper()

	client := gohttpc.NewClient()

	return client.R(http.MethodGet, url).Execute(t.Context())
}

func assertHTTPError(t *testing.T, err error, wantStatus int) *goutils.RFC9457ErrorWithExtensions {
	t.Helper()

	if err == nil {
		t.Fatal("expected an error, got nil")
	}

	httpErr, ok := err.(*goutils.RFC9457ErrorWithExtensions)
	if !ok {
		t.Fatalf("expected *RFC9457ErrorWithExtensions, got %T: %v", err, err)
	}

	if httpErr.Status != wantStatus {
		t.Errorf("expected status %d, got %d", wantStatus, httpErr.Status)
	}

	return httpErr
}

func TestHTTPErrorFromResponse_EmptyBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	resp, err := executeGet(t, server.URL+"/")

	if resp == nil {
		t.Fatal("expected response, got nil")
	}

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}

	httpErr := assertHTTPError(t, err, http.StatusNotFound)

	if httpErr.Title == "" {
		t.Error("expected non-empty Title")
	}

	if httpErr.Extensions["headers"] == nil {
		t.Error("expected headers in extensions")
	}
}

func TestHTTPErrorFromResponse_JSONFullBody(t *testing.T) {
	body := `{"status":404,"title":"Not Found","detail":"resource not found"}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(body))
	}))
	defer server.Close()

	_, err := executeGet(t, server.URL+"/")

	httpErr := assertHTTPError(t, err, http.StatusNotFound)

	if httpErr.Title != "Not Found" {
		t.Errorf("expected title 'Not Found', got %q", httpErr.Title)
	}

	if httpErr.Detail != "resource not found" {
		t.Errorf("expected detail 'resource not found', got %q", httpErr.Detail)
	}
}

func TestHTTPErrorFromResponse_JSONNoStatus_FallsBackToHTTPStatus(t *testing.T) {
	// Status omitted in body — should be filled from HTTP response status code.
	body := `{"title":"Custom Error","detail":"something went wrong"}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(body))
	}))
	defer server.Close()

	_, err := executeGet(t, server.URL+"/")

	httpErr := assertHTTPError(t, err, http.StatusUnprocessableEntity)

	if httpErr.Title != "Custom Error" {
		t.Errorf("expected title 'Custom Error', got %q", httpErr.Title)
	}
}

func TestHTTPErrorFromResponse_JSONNoTitle_FallsBackToHTTPStatusText(t *testing.T) {
	// Title omitted in body — should be filled from resp.Status string.
	body := `{"status":422,"detail":"something went wrong"}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(body))
	}))
	defer server.Close()

	_, err := executeGet(t, server.URL+"/")

	httpErr := assertHTTPError(t, err, http.StatusUnprocessableEntity)

	// resp.Status is "422 Unprocessable Entity"
	if !strings.Contains(httpErr.Title, "422") {
		t.Errorf("expected title to contain HTTP status, got %q", httpErr.Title)
	}
}

func TestHTTPErrorFromResponse_InvalidJSON_FallsBackToNoContent(t *testing.T) {
	// Invalid JSON with application/json content-type falls back to no-content error.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`not-valid-json`))
	}))
	defer server.Close()

	_, err := executeGet(t, server.URL+"/")

	httpErr := assertHTTPError(t, err, http.StatusInternalServerError)

	if httpErr.Extensions["headers"] == nil {
		t.Error("expected headers in extensions")
	}
}

func TestHTTPErrorFromResponse_NonJSONBody_IncludesBodyAsDetail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("bad request details"))
	}))
	defer server.Close()

	_, err := executeGet(t, server.URL+"/")

	httpErr := assertHTTPError(t, err, http.StatusBadRequest)

	if httpErr.Detail != "bad request details" {
		t.Errorf("expected detail to contain body text, got %q", httpErr.Detail)
	}
}

func TestHTTPErrorFromResponse_JSONWithExtensionFields(t *testing.T) {
	body := map[string]any{
		"status":       403,
		"title":        "Forbidden",
		"detail":       "access denied",
		"custom_field": "custom_value",
	}

	bodyBytes, _ := json.Marshal(body)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write(bodyBytes)
	}))
	defer server.Close()

	_, err := executeGet(t, server.URL+"/")

	httpErr := assertHTTPError(t, err, http.StatusForbidden)

	if httpErr.Title != "Forbidden" {
		t.Errorf("expected title 'Forbidden', got %q", httpErr.Title)
	}
}

func TestHTTPErrorFromResponse_ResponseHeadersInExtensions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Request-ID", "abc-123")
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	_, err := executeGet(t, server.URL+"/")

	httpErr := assertHTTPError(t, err, http.StatusInternalServerError)

	if httpErr.Extensions["headers"] == nil {
		t.Error("expected headers extension to be set")
	}
}

func TestHTTPErrorFromResponse_ErrorString(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"status":401,"title":"Unauthorized","detail":"token expired"}`))
	}))
	defer server.Close()

	_, err := executeGet(t, server.URL+"/")

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	errStr := err.Error()

	if !strings.Contains(errStr, "Unauthorized") {
		t.Errorf("expected error string to contain 'Unauthorized', got: %s", errStr)
	}

	if !strings.Contains(errStr, "token expired") {
		t.Errorf("expected error string to contain 'token expired', got: %s", errStr)
	}
}
