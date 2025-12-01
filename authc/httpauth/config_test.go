package httpauth

import (
	"testing"

	"github.com/hasura/goenvconf"
	"github.com/relychan/gohttpc/authc/authscheme"
)

func TestNewHTTPAuthConfig(t *testing.T) {
	t.Run("creates config with correct type", func(t *testing.T) {
		location := authscheme.TokenLocation{
			In:   authscheme.InHeader,
			Name: "Authorization",
		}
		value := goenvconf.NewEnvStringValue("test-token")

		config := NewHTTPAuthConfig(location, value)

		if config.Type != authscheme.HTTPAuthScheme {
			t.Errorf("expected type %s, got %s", authscheme.HTTPAuthScheme, config.Type)
		}

		if config.In != authscheme.InHeader {
			t.Errorf("expected location In to be %s, got %s", authscheme.InHeader, config.In)
		}

		if config.Name != "Authorization" {
			t.Errorf("expected location Name to be 'Authorization', got '%s'", config.Name)
		}
	})
}

func TestHTTPAuthConfig_GetType(t *testing.T) {
	t.Run("returns http auth scheme type", func(t *testing.T) {
		config := &HTTPAuthConfig{}

		if config.GetType() != authscheme.HTTPAuthScheme {
			t.Errorf("expected type %s, got %s", authscheme.HTTPAuthScheme, config.GetType())
		}
	})
}

func TestHTTPAuthConfig_Validate(t *testing.T) {
	t.Run("validates successfully with valid config in non-strict mode", func(t *testing.T) {
		config := &HTTPAuthConfig{
			Type: authscheme.HTTPAuthScheme,
			TokenLocation: authscheme.TokenLocation{
				In:   authscheme.InHeader,
				Name: "Authorization",
			},
			Value: goenvconf.NewEnvStringValue("test-token"),
		}

		err := config.Validate(false)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("validates successfully with valid config in strict mode", func(t *testing.T) {
		config := &HTTPAuthConfig{
			Type: authscheme.HTTPAuthScheme,
			TokenLocation: authscheme.TokenLocation{
				In:   authscheme.InHeader,
				Name: "Authorization",
			},
			Value: goenvconf.NewEnvStringValue("test-token"),
		}

		err := config.Validate(true)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("returns error when type does not match", func(t *testing.T) {
		config := &HTTPAuthConfig{
			Type: authscheme.BasicAuthScheme,
			TokenLocation: authscheme.TokenLocation{
				In:   authscheme.InHeader,
				Name: "Authorization",
			},
			Value: goenvconf.NewEnvStringValue("test-token"),
		}

		err := config.Validate(false)

		if err == nil {
			t.Error("expected error for mismatched type")
		}
	})

	t.Run("returns error when name is empty", func(t *testing.T) {
		config := &HTTPAuthConfig{
			Type: authscheme.HTTPAuthScheme,
			TokenLocation: authscheme.TokenLocation{
				In:   authscheme.InHeader,
				Name: "",
			},
			Value: goenvconf.NewEnvStringValue("test-token"),
		}

		err := config.Validate(false)

		if err == nil {
			t.Error("expected error for empty name")
		}
	})

	t.Run("returns error when location is invalid", func(t *testing.T) {
		config := &HTTPAuthConfig{
			Type: authscheme.HTTPAuthScheme,
			TokenLocation: authscheme.TokenLocation{
				In:   authscheme.AuthLocation("invalid"),
				Name: "Authorization",
			},
			Value: goenvconf.NewEnvStringValue("test-token"),
		}

		err := config.Validate(false)

		if err == nil {
			t.Error("expected error for invalid location")
		}
	})

	t.Run("returns error when value is empty in strict mode", func(t *testing.T) {
		config := &HTTPAuthConfig{
			Type: authscheme.HTTPAuthScheme,
			TokenLocation: authscheme.TokenLocation{
				In:   authscheme.InHeader,
				Name: "Authorization",
			},
			Value: goenvconf.EnvString{},
		}

		err := config.Validate(true)

		if err == nil {
			t.Error("expected error for empty value in strict mode")
		}
	})

	t.Run("allows empty value in non-strict mode", func(t *testing.T) {
		config := &HTTPAuthConfig{
			Type: authscheme.HTTPAuthScheme,
			TokenLocation: authscheme.TokenLocation{
				In:   authscheme.InHeader,
				Name: "Authorization",
			},
			Value: goenvconf.EnvString{},
		}

		err := config.Validate(false)

		if err != nil {
			t.Errorf("unexpected error in non-strict mode: %v", err)
		}
	})
}

