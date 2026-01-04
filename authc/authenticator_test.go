package authc

import (
	"testing"

	"github.com/hasura/goenvconf"
	"github.com/relychan/gohttpc/authc/authscheme"
	"github.com/relychan/gohttpc/authc/basicauth"
	"github.com/relychan/gohttpc/authc/httpauth"
	"github.com/relychan/gohttpc/authc/oauth2scheme"
	"github.com/relychan/goutils"
)

// Helper function to create a pointer to EnvString
func ptrEnvString(value string) *goenvconf.EnvString {
	v := goenvconf.NewEnvStringValue(value)
	return &v
}

func TestNewAuthenticatorFromConfig(t *testing.T) {
	t.Run("creates basic auth authenticator from config", func(t *testing.T) {
		config := &HTTPClientAuthConfig{
			HTTPClientAuthenticatorConfig: &basicauth.BasicAuthConfig{
				Type:     authscheme.BasicAuthScheme,
				Username: goutils.ToPtr(goenvconf.NewEnvStringValue("testuser")),
				Password: goutils.ToPtr(goenvconf.NewEnvStringValue("testpass")),
			},
		}

		authenticator, err := NewAuthenticatorFromConfig(config, authscheme.NewHTTPClientAuthenticatorOptions())
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if authenticator == nil {
			t.Error("expected authenticator to be created")
		}

		if _, ok := authenticator.(*basicauth.BasicCredential); !ok {
			t.Errorf("expected BasicCredential, got %T", authenticator)
		}
	})

	t.Run("creates http auth authenticator from config", func(t *testing.T) {
		config := &HTTPClientAuthConfig{
			HTTPClientAuthenticatorConfig: &httpauth.HTTPAuthConfig{
				Type: authscheme.HTTPAuthScheme,
				TokenLocation: authscheme.TokenLocation{
					In:   authscheme.InHeader,
					Name: "Authorization",
				},
				Value: goenvconf.NewEnvStringValue("test-token"),
			},
		}

		authenticator, err := NewAuthenticatorFromConfig(config, authscheme.NewHTTPClientAuthenticatorOptions())
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if authenticator == nil {
			t.Error("expected authenticator to be created")
		}

		if _, ok := authenticator.(*httpauth.HTTPCredential); !ok {
			t.Errorf("expected HTTPCredential, got %T", authenticator)
		}
	})

	t.Run("creates oauth2 authenticator from config", func(t *testing.T) {
		config := &HTTPClientAuthConfig{
			HTTPClientAuthenticatorConfig: &oauth2scheme.OAuth2Config{
				Type: authscheme.OAuth2Scheme,
				Flows: oauth2scheme.OAuth2Flows{
					ClientCredentials: oauth2scheme.ClientCredentialsOAuthFlow{
						TokenURL:     ptrEnvString("https://example.com/token"),
						ClientID:     ptrEnvString("client-id"),
						ClientSecret: ptrEnvString("client-secret"),
					},
				},
			},
		}

		authenticator, err := NewAuthenticatorFromConfig(config, authscheme.NewHTTPClientAuthenticatorOptions())
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if authenticator == nil {
			t.Error("expected authenticator to be created")
		}

		if _, ok := authenticator.(*oauth2scheme.OAuth2Credential); !ok {
			t.Errorf("expected OAuth2Client, got %T", authenticator)
		}
	})

	t.Run("returns error for unsupported auth type", func(t *testing.T) {
		// Create a mock config with unsupported type
		config := &HTTPClientAuthConfig{
			HTTPClientAuthenticatorConfig: &mockUnsupportedConfig{},
		}

		authenticator, err := NewAuthenticatorFromConfig(config, authscheme.NewHTTPClientAuthenticatorOptions())

		if err == nil {
			t.Error("expected error for unsupported auth type")
		}

		if authenticator != nil {
			t.Error("expected authenticator to be nil")
		}
	})
}

// mockUnsupportedConfig is a mock implementation for testing unsupported auth types
type mockUnsupportedConfig struct{}

func (m *mockUnsupportedConfig) GetType() authscheme.HTTPClientAuthType {
	return "unsupported"
}

// IsZero if the current instance is empty.
func (m mockUnsupportedConfig) IsZero() bool {
	return true
}

func (m *mockUnsupportedConfig) Validate(strict bool) error {
	return nil
}
