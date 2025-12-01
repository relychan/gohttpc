package authscheme

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTokenLocation_InjectRequest_Header(t *testing.T) {
	t.Run("injects token into header", func(t *testing.T) {
		location := TokenLocation{
			In:   InHeader,
			Name: "Authorization",
		}

		req := httptest.NewRequest(http.MethodGet, "https://example.com", nil)
		injected, err := location.InjectRequest(req, "test-token", false)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if !injected {
			t.Error("expected token to be injected")
		}

		if req.Header.Get("Authorization") != "test-token" {
			t.Errorf("expected Authorization header to be 'test-token', got '%s'", req.Header.Get("Authorization"))
		}
	})

	t.Run("does not replace existing header when replace=false", func(t *testing.T) {
		location := TokenLocation{
			In:   InHeader,
			Name: "Authorization",
		}

		req := httptest.NewRequest(http.MethodGet, "https://example.com", nil)
		req.Header.Set("Authorization", "existing-token")

		injected, err := location.InjectRequest(req, "new-token", false)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if !injected {
			t.Error("expected injected to be true")
		}

		if req.Header.Get("Authorization") != "existing-token" {
			t.Errorf("expected Authorization header to remain 'existing-token', got '%s'", req.Header.Get("Authorization"))
		}
	})

	t.Run("replaces existing header when replace=true", func(t *testing.T) {
		location := TokenLocation{
			In:   InHeader,
			Name: "Authorization",
		}

		req := httptest.NewRequest(http.MethodGet, "https://example.com", nil)
		req.Header.Set("Authorization", "existing-token")

		injected, err := location.InjectRequest(req, "new-token", true)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if !injected {
			t.Error("expected token to be injected")
		}

		if req.Header.Get("Authorization") != "new-token" {
			t.Errorf("expected Authorization header to be 'new-token', got '%s'", req.Header.Get("Authorization"))
		}
	})

	t.Run("returns false when value is empty", func(t *testing.T) {
		location := TokenLocation{
			In:   InHeader,
			Name: "Authorization",
		}

		req := httptest.NewRequest(http.MethodGet, "https://example.com", nil)
		injected, err := location.InjectRequest(req, "", false)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if injected {
			t.Error("expected injected to be false for empty value")
		}

		if req.Header.Get("Authorization") != "" {
			t.Error("expected Authorization header to be empty")
		}
	})
}

func TestTokenLocation_InjectRequest_Query(t *testing.T) {
	t.Run("injects token into query parameter", func(t *testing.T) {
		location := TokenLocation{
			In:   InQuery,
			Name: "access_token",
		}

		req := httptest.NewRequest(http.MethodGet, "https://example.com/api", nil)
		injected, err := location.InjectRequest(req, "test-token", false)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if !injected {
			t.Error("expected token to be injected")
		}

		if req.URL.Query().Get("access_token") != "test-token" {
			t.Errorf("expected access_token query param to be 'test-token', got '%s'", req.URL.Query().Get("access_token"))
		}
	})

	t.Run("returns false when value is empty", func(t *testing.T) {
		location := TokenLocation{
			In:   InQuery,
			Name: "access_token",
		}

		req := httptest.NewRequest(http.MethodGet, "https://example.com/api", nil)
		injected, err := location.InjectRequest(req, "", false)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if injected {
			t.Error("expected injected to be false for empty value")
		}

		if req.URL.Query().Get("access_token") != "" {
			t.Error("expected access_token query param to be empty")
		}
	})

	t.Run("preserves existing query parameters", func(t *testing.T) {
		location := TokenLocation{
			In:   InQuery,
			Name: "access_token",
		}

		req := httptest.NewRequest(http.MethodGet, "https://example.com/api?foo=bar", nil)
		injected, err := location.InjectRequest(req, "test-token", false)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if !injected {
			t.Error("expected token to be injected")
		}

		if req.URL.Query().Get("foo") != "bar" {
			t.Error("expected existing query parameter to be preserved")
		}

		if req.URL.Query().Get("access_token") != "test-token" {
			t.Error("expected access_token query param to be set")
		}
	})
}

func TestTokenLocation_InjectRequest_Cookie(t *testing.T) {
	t.Run("does not inject cookie when replace=false and cookie exists", func(t *testing.T) {
		location := TokenLocation{
			In:   InCookie,
			Name: "session",
		}

		req := httptest.NewRequest(http.MethodGet, "https://example.com", nil)
		req.AddCookie(&http.Cookie{Name: "session", Value: "existing-value"})

		injected, err := location.InjectRequest(req, "new-value", false)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if !injected {
			t.Error("expected injected to be true when cookie exists")
		}
	})

	t.Run("returns false when cookie does not exist and replace=false", func(t *testing.T) {
		location := TokenLocation{
			In:   InCookie,
			Name: "session",
		}

		req := httptest.NewRequest(http.MethodGet, "https://example.com", nil)

		injected, err := location.InjectRequest(req, "new-value", false)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if injected {
			t.Error("expected injected to be false when cookie does not exist")
		}
	})
}

func TestTokenLocation_addTokenSchemeToValue(t *testing.T) {
	t.Run("adds Bearer scheme", func(t *testing.T) {
		location := TokenLocation{
			Scheme: "bearer",
		}

		result := location.addTokenSchemeToValue("test-token")

		if result != "Bearer test-token" {
			t.Errorf("expected 'Bearer test-token', got '%s'", result)
		}
	})

	t.Run("adds Basic scheme", func(t *testing.T) {
		location := TokenLocation{
			Scheme: "basic",
		}

		result := location.addTokenSchemeToValue("test-token")

		if result != "Basic test-token" {
			t.Errorf("expected 'Basic test-token', got '%s'", result)
		}
	})

	t.Run("returns value as-is when scheme is empty", func(t *testing.T) {
		location := TokenLocation{
			Scheme: "",
		}

		result := location.addTokenSchemeToValue("test-token")

		if result != "test-token" {
			t.Errorf("expected 'test-token', got '%s'", result)
		}
	})

	t.Run("adds custom scheme", func(t *testing.T) {
		location := TokenLocation{
			Scheme: "custom",
		}

		result := location.addTokenSchemeToValue("test-token")

		if result != "custom test-token" {
			t.Errorf("expected 'custom test-token', got '%s'", result)
		}
	})

	t.Run("injects bearer token with scheme into header", func(t *testing.T) {
		location := TokenLocation{
			In:     InHeader,
			Name:   "Authorization",
			Scheme: "bearer",
		}

		req := httptest.NewRequest(http.MethodGet, "https://example.com", nil)
		injected, err := location.InjectRequest(req, "test-token", false)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if !injected {
			t.Error("expected token to be injected")
		}

		if req.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("expected Authorization header to be 'Bearer test-token', got '%s'", req.Header.Get("Authorization"))
		}
	})
}
