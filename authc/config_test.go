package authc

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/hasura/goenvconf"
	"github.com/relychan/gohttpc/authc/authscheme"
	"github.com/relychan/gohttpc/authc/basicauth"
	"github.com/relychan/gohttpc/authc/httpauth"
	"github.com/relychan/gohttpc/authc/oauth2scheme"
	"go.yaml.in/yaml/v4"
)

func TestHTTPClientAuthConfig_UnmarshalJSON(t *testing.T) {
	t.Run("unmarshals basic auth config from JSON", func(t *testing.T) {
		jsonData := `{
			"type": "basic",
			"username": {"value": "testuser"},
			"password": {"value": "testpass"}
		}`

		var config HTTPClientAuthConfig
		err := json.Unmarshal([]byte(jsonData), &config)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if config.GetType() != authscheme.BasicAuthScheme {
			t.Errorf("expected type %s, got %s", authscheme.BasicAuthScheme, config.GetType())
		}

		basicConfig, ok := config.HTTPClientAuthenticatorConfig.(*basicauth.BasicAuthConfig)
		if !ok {
			t.Errorf("expected BasicAuthConfig, got %T", config.HTTPClientAuthenticatorConfig)
		}

		username, _ := basicConfig.Username.Get()
		if username != "testuser" {
			t.Errorf("expected username 'testuser', got '%s'", username)
		}
	})

	t.Run("unmarshals http auth config from JSON", func(t *testing.T) {
		jsonData := `{
			"type": "http",
			"in": "header",
			"name": "Authorization",
			"scheme": "Bearer",
			"value": {"value": "test-token"}
		}`

		var config HTTPClientAuthConfig
		err := json.Unmarshal([]byte(jsonData), &config)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if config.GetType() != authscheme.HTTPAuthScheme {
			t.Errorf("expected type %s, got %s", authscheme.HTTPAuthScheme, config.GetType())
		}

		httpConfig, ok := config.HTTPClientAuthenticatorConfig.(*httpauth.HTTPAuthConfig)
		if !ok {
			t.Errorf("expected HTTPAuthConfig, got %T", config.HTTPClientAuthenticatorConfig)
		}

		value, _ := httpConfig.Value.Get()
		if value != "test-token" {
			t.Errorf("expected value 'test-token', got '%s'", value)
		}
	})

	t.Run("unmarshals oauth2 config from JSON", func(t *testing.T) {
		jsonData := `{
			"type": "oauth2",
			"flows": {
				"clientCredentials": {
					"tokenUrl": {"value": "https://example.com/token"},
					"clientId": {"value": "client-id"},
					"clientSecret": {"value": "client-secret"}
				}
			}
		}`

		var config HTTPClientAuthConfig
		err := json.Unmarshal([]byte(jsonData), &config)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if config.GetType() != authscheme.OAuth2Scheme {
			t.Errorf("expected type %s, got %s", authscheme.OAuth2Scheme, config.GetType())
		}

		oauth2Config, ok := config.HTTPClientAuthenticatorConfig.(*oauth2scheme.OAuth2Config)
		if !ok {
			t.Errorf("expected OAuth2Config, got %T", config.HTTPClientAuthenticatorConfig)
		}

		tokenURL, _ := oauth2Config.Flows.ClientCredentials.TokenURL.Get()
		if tokenURL != "https://example.com/token" {
			t.Errorf("expected tokenURL 'https://example.com/token', got '%s'", tokenURL)
		}
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		jsonData := `{invalid json}`

		var config HTTPClientAuthConfig
		err := json.Unmarshal([]byte(jsonData), &config)

		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})

	t.Run("returns error for unsupported auth type", func(t *testing.T) {
		jsonData := `{
			"type": "unsupported"
		}`

		var config HTTPClientAuthConfig
		err := json.Unmarshal([]byte(jsonData), &config)

		if err == nil {
			t.Error("expected error for unsupported auth type")
		}
	})

	t.Run("returns error for missing type field", func(t *testing.T) {
		jsonData := `{
			"username": "testuser"
		}`

		var config HTTPClientAuthConfig
		err := json.Unmarshal([]byte(jsonData), &config)

		if err == nil {
			t.Error("expected error for missing type field")
		}
	})
}

func TestHTTPClientAuthConfig_MarshalJSON(t *testing.T) {
	t.Run("marshals basic auth config to JSON", func(t *testing.T) {
		config := HTTPClientAuthConfig{
			HTTPClientAuthenticatorConfig: &basicauth.BasicAuthConfig{
				Type:     authscheme.BasicAuthScheme,
				Username: goenvconf.NewEnvStringValue("testuser"),
				Password: goenvconf.NewEnvStringValue("testpass"),
			},
		}

		data, err := json.Marshal(config)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		var result map[string]interface{}
		err = json.Unmarshal(data, &result)
		if err != nil {
			t.Errorf("failed to unmarshal result: %v", err)
		}

		if result["type"] != "basic" {
			t.Errorf("expected type 'basic', got '%v'", result["type"])
		}
	})

	t.Run("marshals http auth config to JSON", func(t *testing.T) {
		config := HTTPClientAuthConfig{
			HTTPClientAuthenticatorConfig: &httpauth.HTTPAuthConfig{
				Type: authscheme.HTTPAuthScheme,
				TokenLocation: authscheme.TokenLocation{
					In:   authscheme.InHeader,
					Name: "Authorization",
				},
				Value: goenvconf.NewEnvStringValue("test-token"),
			},
		}

		data, err := json.Marshal(config)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		var result map[string]interface{}
		err = json.Unmarshal(data, &result)
		if err != nil {
			t.Errorf("failed to unmarshal result: %v", err)
		}

		if result["type"] != "http" {
			t.Errorf("expected type 'http', got '%v'", result["type"])
		}
	})
}

func TestHTTPClientAuthConfig_UnmarshalYAML(t *testing.T) {
	t.Run("unmarshals basic auth config from YAML", func(t *testing.T) {
		yamlData := `
type: basic
username:
  value: testuser
password:
  value: testpass
`

		var config HTTPClientAuthConfig
		err := yaml.Unmarshal([]byte(yamlData), &config)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if config.GetType() != authscheme.BasicAuthScheme {
			t.Errorf("expected type %s, got %s", authscheme.BasicAuthScheme, config.GetType())
		}

		basicConfig, ok := config.HTTPClientAuthenticatorConfig.(*basicauth.BasicAuthConfig)
		if !ok {
			t.Errorf("expected BasicAuthConfig, got %T", config.HTTPClientAuthenticatorConfig)
		}

		username, _ := basicConfig.Username.Get()
		if username != "testuser" {
			t.Errorf("expected username 'testuser', got '%s'", username)
		}
	})

	t.Run("unmarshals http auth config from YAML", func(t *testing.T) {
		yamlData := `
type: http
in: header
name: Authorization
scheme: Bearer
value:
  value: test-token
`

		var config HTTPClientAuthConfig
		err := yaml.Unmarshal([]byte(yamlData), &config)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if config.GetType() != authscheme.HTTPAuthScheme {
			t.Errorf("expected type %s, got %s", authscheme.HTTPAuthScheme, config.GetType())
		}
	})

	t.Run("unmarshals oauth2 config from YAML", func(t *testing.T) {
		yamlData := `
type: oauth2
flows:
  clientCredentials:
    tokenUrl:
      value: https://example.com/token
    clientId:
      value: client-id
    clientSecret:
      value: client-secret
`

		var config HTTPClientAuthConfig
		err := yaml.Unmarshal([]byte(yamlData), &config)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if config.GetType() != authscheme.OAuth2Scheme {
			t.Errorf("expected type %s, got %s", authscheme.OAuth2Scheme, config.GetType())
		}
	})

	t.Run("returns error for invalid YAML", func(t *testing.T) {
		yamlData := `
type: basic
username: [invalid
`

		var config HTTPClientAuthConfig
		err := yaml.Unmarshal([]byte(yamlData), &config)

		if err == nil {
			t.Error("expected error for invalid YAML")
		}
	})

	t.Run("returns error for unsupported auth type", func(t *testing.T) {
		yamlData := `
type: unsupported
`

		var config HTTPClientAuthConfig
		err := yaml.Unmarshal([]byte(yamlData), &config)

		if err == nil {
			t.Error("expected error for unsupported auth type")
		}
	})
}

func TestHTTPClientAuthConfig_Validate(t *testing.T) {
	t.Run("validates successfully with valid basic auth config", func(t *testing.T) {
		config := &HTTPClientAuthConfig{
			HTTPClientAuthenticatorConfig: &basicauth.BasicAuthConfig{
				Type:     authscheme.BasicAuthScheme,
				Username: goenvconf.NewEnvStringValue("testuser"),
				Password: goenvconf.NewEnvStringValue("testpass"),
			},
		}

		err := config.Validate(false)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("returns error when config is nil", func(t *testing.T) {
		config := &HTTPClientAuthConfig{
			HTTPClientAuthenticatorConfig: nil,
		}

		err := config.Validate(false)

		if err == nil {
			t.Error("expected error for nil config")
		}

		if !errors.Is(err, errSecuritySchemeDefinitionRequired) {
			t.Errorf("expected errSecuritySchemeDefinitionRequired, got %v", err)
		}
	})

	t.Run("returns error when underlying config is invalid", func(t *testing.T) {
		config := &HTTPClientAuthConfig{
			HTTPClientAuthenticatorConfig: &basicauth.BasicAuthConfig{
				Type: authscheme.HTTPAuthScheme, // Wrong type
			},
		}

		err := config.Validate(false)

		if err == nil {
			t.Error("expected error for invalid config")
		}
	})
}

func TestHTTPClientAuthConfig_IsZero(t *testing.T) {
	t.Run("returns true when config is nil", func(t *testing.T) {
		config := &HTTPClientAuthConfig{
			HTTPClientAuthenticatorConfig: nil,
		}

		if !config.IsZero() {
			t.Error("expected IsZero to return true for nil config")
		}
	})

	t.Run("returns false when config is set", func(t *testing.T) {
		config := &HTTPClientAuthConfig{
			HTTPClientAuthenticatorConfig: &basicauth.BasicAuthConfig{
				Type:     authscheme.BasicAuthScheme,
				Username: goenvconf.NewEnvStringValue("testuser"),
				Password: goenvconf.NewEnvStringValue("testpass"),
			},
		}

		if config.IsZero() {
			t.Error("expected IsZero to return false for valid config")
		}
	})
}
