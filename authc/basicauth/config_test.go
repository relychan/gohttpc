package basicauth

import (
	"testing"

	"github.com/hasura/goenvconf"
	"github.com/relychan/gohttpc/authc/authscheme"
)

func TestNewBasicAuthConfig(t *testing.T) {
	t.Run("creates config with correct type", func(t *testing.T) {
		username := goenvconf.NewEnvStringValue("user")
		password := goenvconf.NewEnvStringValue("pass")

		config := NewBasicAuthConfig(username, password)

		if config.Type != authscheme.BasicAuthScheme {
			t.Errorf("expected type %s, got %s", authscheme.BasicAuthScheme, config.Type)
		}

		usernameVal, _ := config.Username.Get()
		if usernameVal != "user" {
			t.Errorf("expected username 'user', got '%s'", usernameVal)
		}

		passwordVal, _ := config.Password.Get()
		if passwordVal != "pass" {
			t.Errorf("expected password 'pass', got '%s'", passwordVal)
		}
	})
}

func TestBasicAuthConfig_GetType(t *testing.T) {
	t.Run("returns basic auth scheme type", func(t *testing.T) {
		config := &BasicAuthConfig{}

		if config.GetType() != authscheme.BasicAuthScheme {
			t.Errorf("expected type %s, got %s", authscheme.BasicAuthScheme, config.GetType())
		}
	})
}

func TestBasicAuthConfig_Validate(t *testing.T) {
	t.Run("validates successfully with valid config in non-strict mode", func(t *testing.T) {
		config := &BasicAuthConfig{
			Type:     authscheme.BasicAuthScheme,
			Username: goenvconf.NewEnvStringValue("user"),
			Password: goenvconf.NewEnvStringValue("pass"),
		}

		err := config.Validate(false)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("validates successfully with valid config in strict mode", func(t *testing.T) {
		config := &BasicAuthConfig{
			Type:     authscheme.BasicAuthScheme,
			Username: goenvconf.NewEnvStringValue("user"),
			Password: goenvconf.NewEnvStringValue("pass"),
		}

		err := config.Validate(true)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("returns error when type does not match", func(t *testing.T) {
		config := &BasicAuthConfig{
			Type:     authscheme.HTTPAuthScheme,
			Username: goenvconf.NewEnvStringValue("user"),
			Password: goenvconf.NewEnvStringValue("pass"),
		}

		err := config.Validate(false)

		if err == nil {
			t.Error("expected error for mismatched type")
		}
	})

	t.Run("returns error when username is empty in strict mode", func(t *testing.T) {
		config := &BasicAuthConfig{
			Type:     authscheme.BasicAuthScheme,
			Username: goenvconf.EnvString{},
			Password: goenvconf.NewEnvStringValue("pass"),
		}

		err := config.Validate(true)

		if err == nil {
			t.Error("expected error for empty username in strict mode")
		}
	})

	t.Run("returns error when password is empty in strict mode", func(t *testing.T) {
		config := &BasicAuthConfig{
			Type:     authscheme.BasicAuthScheme,
			Username: goenvconf.NewEnvStringValue("user"),
			Password: goenvconf.EnvString{},
		}

		err := config.Validate(true)

		if err == nil {
			t.Error("expected error for empty password in strict mode")
		}
	})

	t.Run("allows empty username and password in non-strict mode", func(t *testing.T) {
		config := &BasicAuthConfig{
			Type:     authscheme.BasicAuthScheme,
			Username: goenvconf.EnvString{},
			Password: goenvconf.EnvString{},
		}

		err := config.Validate(false)

		if err != nil {
			t.Errorf("unexpected error in non-strict mode: %v", err)
		}
	})

	t.Run("validates config with custom header", func(t *testing.T) {
		config := &BasicAuthConfig{
			Type:     authscheme.BasicAuthScheme,
			Header:   "X-Custom-Auth",
			Username: goenvconf.NewEnvStringValue("user"),
			Password: goenvconf.NewEnvStringValue("pass"),
		}

		err := config.Validate(true)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("validates config with description", func(t *testing.T) {
		config := &BasicAuthConfig{
			Type:        authscheme.BasicAuthScheme,
			Username:    goenvconf.NewEnvStringValue("user"),
			Password:    goenvconf.NewEnvStringValue("pass"),
			Description: "Basic authentication for API",
		}

		err := config.Validate(true)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}
