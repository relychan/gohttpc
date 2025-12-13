package httpauth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hasura/goenvconf"
	"github.com/relychan/gohttpc/authc/authscheme"
	"github.com/relychan/goutils"
)

func TestNewHTTPCredential(t *testing.T) {
	t.Run("creates credential with valid config", func(t *testing.T) {
		config := &HTTPAuthConfig{
			Type: authscheme.HTTPAuthScheme,
			TokenLocation: authscheme.TokenLocation{
				In:     authscheme.InHeader,
				Name:   "Authorization",
				Scheme: "Bearer",
			},
			Value: goenvconf.NewEnvStringValue("test-token"),
		}

		cred, err := NewHTTPCredential(context.TODO(), config, authscheme.NewHTTPClientAuthenticatorOptions())

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if cred == nil {
			t.Fatal("expected credential to be created")
		}

		if cred.value != "test-token" {
			t.Errorf("expected value 'test-token', got '%s'", cred.value)
		}

		if !goutils.EqualPtr(cred, cred) {
			t.Errorf("expected self equality, got 'false'")
		}

		if goutils.EqualPtr(cred, nil) {
			t.Errorf("expected not equal, got 'true'")
		}
	})

	t.Run("creates credential with custom header name", func(t *testing.T) {
		config := &HTTPAuthConfig{
			Type: authscheme.HTTPAuthScheme,
			TokenLocation: authscheme.TokenLocation{
				In:   authscheme.InHeader,
				Name: "X-API-Key",
			},
			Value: goenvconf.NewEnvStringValue("test-key"),
		}

		cred, err := NewHTTPCredential(context.TODO(), config, authscheme.NewHTTPClientAuthenticatorOptions())

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if cred.location.In != authscheme.InHeader {
			t.Errorf("expected location In to be %s, got %s", authscheme.InHeader, cred.location.In)
		}

		if cred.location.Name != "X-API-Key" {
			t.Errorf("expected location Name to be 'X-API-Key', got '%s'", cred.location.Name)
		}
	})

	t.Run("returns error when value resolution fails", func(t *testing.T) {
		config := &HTTPAuthConfig{
			Type: authscheme.HTTPAuthScheme,
			TokenLocation: authscheme.TokenLocation{
				In:   authscheme.InHeader,
				Name: "Authorization",
			},
			Value: goenvconf.NewEnvStringVariable("NONEXISTENT_VAR"),
		}

		_, err := NewHTTPCredential(context.TODO(), config, authscheme.NewHTTPClientAuthenticatorOptions())

		if err == nil {
			t.Error("expected error when value resolution fails")
		}
	})

	t.Run("creates credential with empty value", func(t *testing.T) {
		config := &HTTPAuthConfig{
			Type: authscheme.HTTPAuthScheme,
			TokenLocation: authscheme.TokenLocation{
				In:     authscheme.InHeader,
				Name:   "Authorization",
				Scheme: "Bearer",
			},
			Value: goenvconf.NewEnvStringVariable("NONEXISTENT_VAR"),
		}

		_, err := NewHTTPCredential(context.TODO(), config, &authscheme.HTTPClientAuthenticatorOptions{
			CustomEnvGetter: func(ctx context.Context) goenvconf.GetEnvFunc {
				return func(s string) (string, error) {
					return "", nil
				}
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestHTTPCredential_Authenticate(t *testing.T) {
	t.Run("authenticates with bearer token in header", func(t *testing.T) {
		config := &HTTPAuthConfig{
			Type: authscheme.HTTPAuthScheme,
			TokenLocation: authscheme.TokenLocation{
				In:     authscheme.InHeader,
				Name:   "Authorization",
				Scheme: "Bearer",
			},
			Value: goenvconf.NewEnvStringValue("test-token"),
		}

		cred, err := NewHTTPCredential(context.TODO(), config, authscheme.NewHTTPClientAuthenticatorOptions())
		if err != nil {
			t.Fatalf("failed to create credential: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "https://example.com", nil)
		err = cred.Authenticate(req)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		authHeader := req.Header.Get("Authorization")
		if authHeader != "Bearer test-token" {
			t.Errorf("expected Authorization header 'Bearer test-token', got '%s'", authHeader)
		}
	})

	t.Run("authenticates with token without scheme", func(t *testing.T) {
		config := &HTTPAuthConfig{
			Type: authscheme.HTTPAuthScheme,
			TokenLocation: authscheme.TokenLocation{
				In:   authscheme.InHeader,
				Name: "X-API-Token",
			},
			Value: goenvconf.NewEnvStringValue("test-key"),
		}

		cred, err := NewHTTPCredential(context.TODO(), config, authscheme.NewHTTPClientAuthenticatorOptions())
		if err != nil {
			t.Fatalf("failed to create credential: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "https://example.com", nil)
		err = cred.Authenticate(req)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		apiToken := req.Header.Get("X-API-Token")
		if apiToken != "test-key" {
			t.Errorf("expected X-API-Token header 'test-key', got '%s'", apiToken)
		}
	})

	t.Run("authenticates with custom header", func(t *testing.T) {
		config := &HTTPAuthConfig{
			Type: authscheme.HTTPAuthScheme,
			TokenLocation: authscheme.TokenLocation{
				In:   authscheme.InHeader,
				Name: "X-API-Key",
			},
			Value: goenvconf.NewEnvStringValue("custom-key"),
		}

		cred, err := NewHTTPCredential(context.TODO(), config, authscheme.NewHTTPClientAuthenticatorOptions())
		if err != nil {
			t.Fatalf("failed to create credential: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "https://example.com", nil)
		err = cred.Authenticate(req)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		apiKey := req.Header.Get("X-API-Key")
		if apiKey != "custom-key" {
			t.Errorf("expected X-API-Key header 'custom-key', got '%s'", apiKey)
		}
	})
}
