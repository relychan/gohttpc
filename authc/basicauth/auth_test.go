package basicauth

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hasura/goenvconf"
	"github.com/relychan/gohttpc/authc/authscheme"
)

func TestNewBasicCredential(t *testing.T) {
	t.Run("creates credential with valid config", func(t *testing.T) {
		config := &BasicAuthConfig{
			Type:     authscheme.BasicAuthScheme,
			Username: goenvconf.NewEnvStringValue("testuser"),
			Password: goenvconf.NewEnvStringValue("testpass"),
		}

		cred, err := NewBasicCredential(config)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if cred == nil {
			t.Fatal("expected credential to be created")
		}

		if cred.username != "testuser" {
			t.Errorf("expected username 'testuser', got '%s'", cred.username)
		}

		if cred.password != "testpass" {
			t.Errorf("expected password 'testpass', got '%s'", cred.password)
		}
	})

	t.Run("creates credential with custom header", func(t *testing.T) {
		config := &BasicAuthConfig{
			Type:     authscheme.BasicAuthScheme,
			Header:   "X-Custom-Auth",
			Username: goenvconf.NewEnvStringValue("testuser"),
			Password: goenvconf.NewEnvStringValue("testpass"),
		}

		cred, err := NewBasicCredential(config)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if cred.header != "X-Custom-Auth" {
			t.Errorf("expected header 'X-Custom-Auth', got '%s'", cred.header)
		}
	})

	t.Run("returns error when username resolution fails", func(t *testing.T) {
		config := &BasicAuthConfig{
			Type:     authscheme.BasicAuthScheme,
			Username: goenvconf.NewEnvStringVariable("NONEXISTENT_VAR"),
			Password: goenvconf.NewEnvStringValue("testpass"),
		}

		_, err := NewBasicCredential(config)

		if err == nil {
			t.Error("expected error when username resolution fails")
		}
	})

	t.Run("returns error when password resolution fails", func(t *testing.T) {
		config := &BasicAuthConfig{
			Type:     authscheme.BasicAuthScheme,
			Username: goenvconf.NewEnvStringValue("testuser"),
			Password: goenvconf.NewEnvStringVariable("NONEXISTENT_VAR"),
		}

		_, err := NewBasicCredential(config)

		if err == nil {
			t.Error("expected error when password resolution fails")
		}
	})
}

func TestBasicCredential_Authenticate(t *testing.T) {
	t.Run("authenticates with standard Authorization header", func(t *testing.T) {
		config := &BasicAuthConfig{
			Type:     authscheme.BasicAuthScheme,
			Username: goenvconf.NewEnvStringValue("testuser"),
			Password: goenvconf.NewEnvStringValue("testpass"),
		}

		cred, err := NewBasicCredential(config)
		if err != nil {
			t.Fatalf("failed to create credential: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "https://example.com", nil)
		err = cred.Authenticate(req)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		authHeader := req.Header.Get("Authorization")
		if authHeader == "" {
			t.Fatal("expected Authorization header to be set")
		}

		// Verify the header format
		expectedAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("testuser:testpass"))
		if authHeader != expectedAuth {
			t.Errorf("expected Authorization header '%s', got '%s'", expectedAuth, authHeader)
		}
	})

	t.Run("authenticates with custom header", func(t *testing.T) {
		config := &BasicAuthConfig{
			Type:     authscheme.BasicAuthScheme,
			Header:   "X-Custom-Auth",
			Username: goenvconf.NewEnvStringValue("testuser"),
			Password: goenvconf.NewEnvStringValue("testpass"),
		}

		cred, err := NewBasicCredential(config)
		if err != nil {
			t.Fatalf("failed to create credential: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "https://example.com", nil)
		err = cred.Authenticate(req, authscheme.WithAuthenticationName("test"))

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		authHeader := req.Header.Get("X-Custom-Auth")
		if authHeader == "" {
			t.Fatal("expected X-Custom-Auth header to be set")
		}

		// Verify the header format
		expectedAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("testuser:testpass"))
		if authHeader != expectedAuth {
			t.Errorf("expected X-Custom-Auth header '%s', got '%s'", expectedAuth, authHeader)
		}
	})

	t.Run("returns error when both username and password are empty", func(t *testing.T) {
		config := &BasicAuthConfig{
			Type:     authscheme.BasicAuthScheme,
			Username: goenvconf.NewEnvStringValue(""),
			Password: goenvconf.NewEnvStringValue(""),
		}

		cred, err := NewBasicCredential(config)
		if err != nil {
			t.Fatalf("failed to create credential: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "https://example.com", nil)
		err = cred.Authenticate(req)

		if err != authscheme.ErrAuthCredentialEmpty {
			t.Errorf("expected ErrAuthCredentialEmpty, got %v", err)
		}
	})

	t.Run("authenticates with username only (no password)", func(t *testing.T) {
		config := &BasicAuthConfig{
			Type:     authscheme.BasicAuthScheme,
			Username: goenvconf.NewEnvStringValue("testuser"),
			Password: goenvconf.NewEnvStringValue(""),
		}

		cred, err := NewBasicCredential(config)
		if err != nil {
			t.Fatalf("failed to create credential: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "https://example.com", nil)
		err = cred.Authenticate(req)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		authHeader := req.Header.Get("Authorization")
		if authHeader == "" {
			t.Fatal("expected Authorization header to be set")
		}
	})

	t.Run("authenticates with username only using custom header", func(t *testing.T) {
		config := &BasicAuthConfig{
			Type:     authscheme.BasicAuthScheme,
			Header:   "X-Custom-Auth",
			Username: goenvconf.NewEnvStringValue("testuser"),
			Password: goenvconf.NewEnvStringValue(""),
		}

		cred, err := NewBasicCredential(config)
		if err != nil {
			t.Fatalf("failed to create credential: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "https://example.com", nil)
		err = cred.Authenticate(req)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		authHeader := req.Header.Get("X-Custom-Auth")
		if authHeader == "" {
			t.Fatal("expected X-Custom-Auth header to be set")
		}

		// Verify the header format for username only
		expectedAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("testuser"))
		if authHeader != expectedAuth {
			t.Errorf("expected X-Custom-Auth header '%s', got '%s'", expectedAuth, authHeader)
		}
	})

	t.Run("authenticates with special characters in credentials", func(t *testing.T) {
		config := &BasicAuthConfig{
			Type:     authscheme.BasicAuthScheme,
			Username: goenvconf.NewEnvStringValue("user@example.com"),
			Password: goenvconf.NewEnvStringValue("p@ss:w0rd!"),
		}

		cred, err := NewBasicCredential(config)
		if err != nil {
			t.Fatalf("failed to create credential: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "https://example.com", nil)
		err = cred.Authenticate(req)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		authHeader := req.Header.Get("Authorization")
		if authHeader == "" {
			t.Fatal("expected Authorization header to be set")
		}

		// Verify the header format with special characters
		expectedAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("user@example.com:p@ss:w0rd!"))
		if authHeader != expectedAuth {
			t.Errorf("expected Authorization header '%s', got '%s'", expectedAuth, authHeader)
		}
	})
}
