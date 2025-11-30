package loadbalancer

import (
	"os"
	"strings"
	"testing"

	"github.com/hasura/goenvconf"
)

func TestHTTPHealthCheckConfig_ToPolicyBuilder_Headers(t *testing.T) {
	t.Run("headers correctly resolved from environment variables", func(t *testing.T) {
		// Set up environment variable
		const envVarName = "TEST_HEALTH_CHECK_HEADER_VALUE"
		const expectedValue = "test-header-value"
		t.Setenv(envVarName, expectedValue)

		config := HTTPHealthCheckConfig{
			Path: "/healthz",
			Headers: map[string]goenvconf.EnvString{
				"Authorization": goenvconf.NewEnvStringVariable(envVarName),
			},
		}

		builder, err := config.ToPolicyBuilder()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if builder.headers == nil {
			t.Fatal("expected headers to be initialized")
		}

		if got := builder.headers["Authorization"]; got != expectedValue {
			t.Errorf("expected header value %q, got %q", expectedValue, got)
		}
	})

	t.Run("headers resolved from literal values", func(t *testing.T) {
		const expectedValue = "literal-header-value"

		config := HTTPHealthCheckConfig{
			Path: "/healthz",
			Headers: map[string]goenvconf.EnvString{
				"X-Custom-Header": goenvconf.NewEnvStringValue(expectedValue),
			},
		}

		builder, err := config.ToPolicyBuilder()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if builder.headers == nil {
			t.Fatal("expected headers to be initialized")
		}

		if got := builder.headers["X-Custom-Header"]; got != expectedValue {
			t.Errorf("expected header value %q, got %q", expectedValue, got)
		}
	})

	t.Run("error handling when environment variable resolution fails", func(t *testing.T) {
		// Use a zero-valued EnvString which will return an error
		config := HTTPHealthCheckConfig{
			Path: "/healthz",
			Headers: map[string]goenvconf.EnvString{
				"Authorization": {}, // zero value - neither value nor env set
			},
		}

		_, err := config.ToPolicyBuilder()
		if err == nil {
			t.Fatal("expected error when EnvString is zero-valued")
		}

		// Verify the error message contains the header key
		expectedSubstring := "failed to get header Authorization"
		if !strings.Contains(err.Error(), expectedSubstring) {
			t.Errorf("expected error to contain %q, got %q", expectedSubstring, err.Error())
		}
	})

	t.Run("missing environment variable returns empty header (excluded)", func(t *testing.T) {
		// Ensure the environment variable does not exist
		const nonExistentEnvVar = "NON_EXISTENT_HEALTH_CHECK_ENV_VAR"
		os.Unsetenv(nonExistentEnvVar)

		config := HTTPHealthCheckConfig{
			Path: "/healthz",
			Headers: map[string]goenvconf.EnvString{
				"Authorization": goenvconf.NewEnvStringVariable(nonExistentEnvVar),
			},
		}

		builder, err := config.ToPolicyBuilder()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// When env var doesn't exist, GetOrDefault returns empty string,
		// and empty headers are not included
		if _, exists := builder.headers["Authorization"]; exists {
			t.Error("expected missing env var header to not be included")
		}
	})

	t.Run("empty header values are not included in headers", func(t *testing.T) {
		config := HTTPHealthCheckConfig{
			Path: "/healthz",
			Headers: map[string]goenvconf.EnvString{
				"Empty-Header":    goenvconf.NewEnvStringValue(""),
				"NonEmpty-Header": goenvconf.NewEnvStringValue("some-value"),
			},
		}

		builder, err := config.ToPolicyBuilder()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if builder.headers == nil {
			t.Fatal("expected headers to be initialized")
		}

		// Empty header should not be included
		if _, exists := builder.headers["Empty-Header"]; exists {
			t.Error("expected empty header to not be included")
		}

		// Non-empty header should be included
		if got, exists := builder.headers["NonEmpty-Header"]; !exists || got != "some-value" {
			t.Errorf("expected NonEmpty-Header to be %q, got %q (exists: %v)", "some-value", got, exists)
		}
	})

	t.Run("multiple headers resolved correctly", func(t *testing.T) {
		const envVarName = "TEST_MULTI_HEADER_ENV"
		const envVarValue = "env-value"
		t.Setenv(envVarName, envVarValue)

		config := HTTPHealthCheckConfig{
			Path: "/healthz",
			Headers: map[string]goenvconf.EnvString{
				"Authorization":   goenvconf.NewEnvStringVariable(envVarName),
				"X-Custom-Header": goenvconf.NewEnvStringValue("literal-value"),
			},
		}

		builder, err := config.ToPolicyBuilder()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if builder.headers == nil {
			t.Fatal("expected headers to be initialized")
		}

		if len(builder.headers) != 2 {
			t.Errorf("expected 2 headers, got %d", len(builder.headers))
		}

		if got := builder.headers["Authorization"]; got != envVarValue {
			t.Errorf("expected Authorization header %q, got %q", envVarValue, got)
		}

		if got := builder.headers["X-Custom-Header"]; got != "literal-value" {
			t.Errorf("expected X-Custom-Header %q, got %q", "literal-value", got)
		}
	})

	t.Run("no headers when config has empty headers map", func(t *testing.T) {
		config := HTTPHealthCheckConfig{
			Path:    "/healthz",
			Headers: map[string]goenvconf.EnvString{},
		}

		builder, err := config.ToPolicyBuilder()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Headers should be nil or empty when no headers are configured
		if len(builder.headers) > 0 {
			t.Errorf("expected no headers, got %v", builder.headers)
		}
	})

	t.Run("no headers when config has nil headers map", func(t *testing.T) {
		config := HTTPHealthCheckConfig{
			Path:    "/healthz",
			Headers: nil,
		}

		builder, err := config.ToPolicyBuilder()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Headers should be nil when no headers are configured
		if builder.headers != nil {
			t.Errorf("expected nil headers, got %v", builder.headers)
		}
	})
}
