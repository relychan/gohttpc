package oauth2scheme

import (
	"errors"
	"testing"

	"github.com/hasura/goenvconf"
	"github.com/relychan/gohttpc/authc/authscheme"
)

// Helper function to create a pointer to EnvString
func ptrEnvString(value string) *goenvconf.EnvString {
	v := goenvconf.NewEnvStringValue(value)
	return &v
}

// Helper function to create a pointer to EnvString from variable
func ptrEnvStringVar(varName string) *goenvconf.EnvString {
	v := goenvconf.NewEnvStringVariable(varName)
	return &v
}

func TestNewOAuth2Config(t *testing.T) {
	t.Run("creates config with correct type", func(t *testing.T) {
		flows := OAuth2Flows{
			ClientCredentials: ClientCredentialsOAuthFlow{
				TokenURL:     ptrEnvString("https://example.com/token"),
				ClientID:     ptrEnvString("client-id"),
				ClientSecret: ptrEnvString("client-secret"),
			},
		}

		config := NewOAuth2Config(flows)

		if config.Type != authscheme.OAuth2Scheme {
			t.Errorf("expected type %s, got %s", authscheme.OAuth2Scheme, config.Type)
		}

		if config.Flows.ClientCredentials.TokenURL == nil {
			t.Error("expected TokenURL to be set")
		}
	})
}

func TestOAuth2Config_GetType(t *testing.T) {
	t.Run("returns oauth2 scheme type", func(t *testing.T) {
		config := &OAuth2Config{}

		if config.GetType() != authscheme.OAuth2Scheme {
			t.Errorf("expected type %s, got %s", authscheme.OAuth2Scheme, config.GetType())
		}
	})
}

func TestOAuth2Config_Validate(t *testing.T) {
	t.Run("validates successfully with valid config", func(t *testing.T) {
		config := &OAuth2Config{
			Type: authscheme.OAuth2Scheme,
			Flows: OAuth2Flows{
				ClientCredentials: ClientCredentialsOAuthFlow{
					TokenURL:     ptrEnvString("https://example.com/token"),
					ClientID:     ptrEnvString("client-id"),
					ClientSecret: ptrEnvString("client-secret"),
				},
			},
		}

		err := config.Validate(false)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("returns error when type does not match", func(t *testing.T) {
		config := &OAuth2Config{
			Type: authscheme.BasicAuthScheme,
			Flows: OAuth2Flows{
				ClientCredentials: ClientCredentialsOAuthFlow{
					TokenURL:     ptrEnvString("https://example.com/token"),
					ClientID:     ptrEnvString("client-id"),
					ClientSecret: ptrEnvString("client-secret"),
				},
			},
		}

		err := config.Validate(false)

		if err == nil {
			t.Error("expected error for mismatched type")
		}
	})

	t.Run("returns error when client credentials flow is invalid", func(t *testing.T) {
		config := &OAuth2Config{
			Type: authscheme.OAuth2Scheme,
			Flows: OAuth2Flows{
				ClientCredentials: ClientCredentialsOAuthFlow{
					// Missing required fields
				},
			},
		}

		err := config.Validate(false)

		if err == nil {
			t.Error("expected error for invalid client credentials flow")
		}
	})
}

func TestClientCredentialsOAuthFlow_Validate(t *testing.T) {
	t.Run("validates successfully with all required fields", func(t *testing.T) {
		flow := ClientCredentialsOAuthFlow{
			TokenURL:     ptrEnvString("https://example.com/token"),
			ClientID:     ptrEnvString("client-id"),
			ClientSecret: ptrEnvString("client-secret"),
		}

		err := flow.Validate()

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("validates successfully with optional fields", func(t *testing.T) {
		flow := ClientCredentialsOAuthFlow{
			TokenURL:     ptrEnvString("https://example.com/token"),
			RefreshURL:   ptrEnvString("https://example.com/refresh"),
			ClientID:     ptrEnvString("client-id"),
			ClientSecret: ptrEnvString("client-secret"),
			Scopes: map[string]string{
				"read":  "Read access",
				"write": "Write access",
			},
			EndpointParams: map[string]goenvconf.EnvString{
				"audience": goenvconf.NewEnvStringValue("https://api.example.com"),
			},
		}

		err := flow.Validate()

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("returns error when token URL is nil", func(t *testing.T) {
		flow := ClientCredentialsOAuthFlow{
			TokenURL:     nil,
			ClientID:     ptrEnvString("client-id"),
			ClientSecret: ptrEnvString("client-secret"),
		}

		err := flow.Validate()

		if err == nil {
			t.Error("expected error for nil token URL")
		}

		if !errors.Is(err, ErrTokenURLRequired) {
			t.Errorf("expected ErrTokenURLRequired, got %v", err)
		}
	})

	t.Run("returns error when token URL is empty", func(t *testing.T) {
		flow := ClientCredentialsOAuthFlow{
			TokenURL:     &goenvconf.EnvString{},
			ClientID:     ptrEnvString("client-id"),
			ClientSecret: ptrEnvString("client-secret"),
		}

		err := flow.Validate()

		if err == nil {
			t.Error("expected error for empty token URL")
		}

		if !errors.Is(err, ErrTokenURLRequired) {
			t.Errorf("expected ErrTokenURLRequired, got %v", err)
		}
	})

	t.Run("returns error when client ID is nil", func(t *testing.T) {
		flow := ClientCredentialsOAuthFlow{
			TokenURL:     ptrEnvString("https://example.com/token"),
			ClientID:     nil,
			ClientSecret: ptrEnvString("client-secret"),
		}

		err := flow.Validate()

		if err == nil {
			t.Error("expected error for nil client ID")
		}

		if !errors.Is(err, ErrClientIDRequired) {
			t.Errorf("expected ErrClientIDRequired, got %v", err)
		}
	})

	t.Run("returns error when client ID is empty", func(t *testing.T) {
		flow := ClientCredentialsOAuthFlow{
			TokenURL:     ptrEnvString("https://example.com/token"),
			ClientID:     &goenvconf.EnvString{},
			ClientSecret: ptrEnvString("client-secret"),
		}

		err := flow.Validate()

		if err == nil {
			t.Error("expected error for empty client ID")
		}

		if !errors.Is(err, ErrClientIDRequired) {
			t.Errorf("expected ErrClientIDRequired, got %v", err)
		}
	})

	t.Run("returns error when client secret is nil", func(t *testing.T) {
		flow := ClientCredentialsOAuthFlow{
			TokenURL:     ptrEnvString("https://example.com/token"),
			ClientID:     ptrEnvString("client-id"),
			ClientSecret: nil,
		}

		err := flow.Validate()

		if err == nil {
			t.Error("expected error for nil client secret")
		}

		if !errors.Is(err, ErrClientSecretRequired) {
			t.Errorf("expected ErrClientSecretRequired, got %v", err)
		}
	})

	t.Run("returns error when client secret is empty", func(t *testing.T) {
		flow := ClientCredentialsOAuthFlow{
			TokenURL:     ptrEnvString("https://example.com/token"),
			ClientID:     ptrEnvString("client-id"),
			ClientSecret: &goenvconf.EnvString{},
		}

		err := flow.Validate()

		if err == nil {
			t.Error("expected error for empty client secret")
		}

		if !errors.Is(err, ErrClientSecretRequired) {
			t.Errorf("expected ErrClientSecretRequired, got %v", err)
		}
	})
}
