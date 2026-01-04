package httpconfig

import (
	"net/http"
	"testing"
	"time"

	"github.com/hasura/goenvconf"
	"github.com/relychan/gohttpc"
	"github.com/relychan/gohttpc/authc"
	"github.com/relychan/gohttpc/authc/authscheme"
	"github.com/relychan/gohttpc/authc/basicauth"
	"github.com/relychan/goutils"
)

func TestHTTPClientConfig_IsZero(t *testing.T) {
	t.Run("returns true when all fields are nil or zero", func(t *testing.T) {
		config := HTTPClientConfig{}

		if !config.IsZero() {
			t.Error("expected IsZero to return true")
		}
	})

	t.Run("returns false when Timeout is set", func(t *testing.T) {
		timeout := goutils.Duration(time.Second * 30)
		config := HTTPClientConfig{
			Timeout: &timeout,
		}

		if config.IsZero() {
			t.Error("expected IsZero to return false")
		}
	})

	t.Run("returns true when Timeout is zero", func(t *testing.T) {
		timeout := goutils.Duration(0)
		config := HTTPClientConfig{
			Timeout: &timeout,
		}

		if !config.IsZero() {
			t.Error("expected IsZero to return true for zero timeout")
		}
	})

	t.Run("returns false when Transport is set", func(t *testing.T) {
		config := HTTPClientConfig{
			Transport: &gohttpc.HTTPTransportConfig{},
		}

		if config.IsZero() {
			t.Error("expected IsZero to return false")
		}
	})

	t.Run("returns false when TLS is set", func(t *testing.T) {
		config := HTTPClientConfig{
			TLS: &TLSConfig{},
		}

		if config.IsZero() {
			t.Error("expected IsZero to return false")
		}
	})

	t.Run("returns true when Retry is empty", func(t *testing.T) {
		config := HTTPClientConfig{
			Retry: &HTTPRetryConfig{},
		}

		if !config.IsZero() {
			t.Error("expected IsZero to return true for empty retry config")
		}
	})

	t.Run("returns false when Retry is set with values", func(t *testing.T) {
		maxAttempts := goenvconf.NewEnvIntValue(3)
		config := HTTPClientConfig{
			Retry: &HTTPRetryConfig{
				MaxAttempts: &maxAttempts,
			},
		}

		if config.IsZero() {
			t.Error("expected IsZero to return false")
		}
	})

	t.Run("returns false when Authentication is set", func(t *testing.T) {
		config := HTTPClientConfig{
			Authentication: &authc.HTTPClientAuthConfig{},
		}

		if config.IsZero() {
			t.Error("expected IsZero to return false")
		}
	})
}

func TestHTTPClientConfig_Equal(t *testing.T) {
	t.Run("returns true for two empty configs", func(t *testing.T) {
		config1 := HTTPClientConfig{}
		config2 := HTTPClientConfig{}

		if !config1.Equal(config2) {
			t.Error("expected Equal to return true for two empty configs")
		}
	})

	t.Run("returns true for identical configs with timeout", func(t *testing.T) {
		timeout := goutils.Duration(time.Second * 30)
		config1 := HTTPClientConfig{
			Timeout: &timeout,
		}
		config2 := HTTPClientConfig{
			Timeout: &timeout,
		}

		if !config1.Equal(config2) {
			t.Error("expected Equal to return true")
		}
	})

	t.Run("returns false for different timeout values", func(t *testing.T) {
		timeout1 := goutils.Duration(time.Second * 30)
		timeout2 := goutils.Duration(time.Second * 60)
		config1 := HTTPClientConfig{
			Timeout: &timeout1,
		}
		config2 := HTTPClientConfig{
			Timeout: &timeout2,
		}

		if config1.Equal(config2) {
			t.Error("expected Equal to return false for different timeouts")
		}
	})

	t.Run("returns false when one has timeout and other doesn't", func(t *testing.T) {
		timeout := goutils.Duration(time.Second * 30)
		config1 := HTTPClientConfig{
			Timeout: &timeout,
		}
		config2 := HTTPClientConfig{}

		if config1.Equal(config2) {
			t.Error("expected Equal to return false")
		}
	})

	t.Run("returns true for identical transport configs", func(t *testing.T) {
		maxIdleConns := 50
		transport := &gohttpc.HTTPTransportConfig{
			MaxIdleConns: &maxIdleConns,
		}
		config1 := HTTPClientConfig{
			Transport: transport,
		}
		config2 := HTTPClientConfig{
			Transport: transport,
		}

		if !config1.Equal(config2) {
			t.Error("expected Equal to return true")
		}
	})

	t.Run("returns true for identical TLS configs", func(t *testing.T) {
		tlsConfig := &TLSConfig{
			MinVersion: "1.2",
		}
		config1 := HTTPClientConfig{
			TLS: tlsConfig,
		}
		config2 := HTTPClientConfig{
			TLS: tlsConfig,
		}

		if !config1.Equal(config2) {
			t.Error("expected Equal to return true")
		}
	})

	t.Run("returns true for identical retry configs", func(t *testing.T) {
		maxAttempts := goenvconf.NewEnvIntValue(3)
		retryConfig := &HTTPRetryConfig{
			MaxAttempts: &maxAttempts,
		}
		config1 := HTTPClientConfig{
			Retry: retryConfig,
		}
		config2 := HTTPClientConfig{
			Retry: retryConfig,
		}

		if !config1.Equal(config2) {
			t.Error("expected Equal to return true")
		}
	})

	t.Run("returns true for identical authentication configs", func(t *testing.T) {
		authConfig := &authc.HTTPClientAuthConfig{
			HTTPClientAuthenticatorConfig: &basicauth.BasicAuthConfig{
				Type:     authscheme.BasicAuthScheme,
				Username: goenvconf.NewEnvStringValue("user"),
				Password: goenvconf.NewEnvStringValue("pass"),
			},
		}
		config1 := HTTPClientConfig{
			Authentication: authConfig,
		}
		config2 := HTTPClientConfig{
			Authentication: authConfig,
		}

		if !config1.Equal(config2) {
			t.Error("expected Equal to return true")
		}
	})

	t.Run("returns true for fully identical configs", func(t *testing.T) {
		timeout := goutils.Duration(time.Second * 30)
		maxIdleConns := 50
		maxAttempts := goenvconf.NewEnvIntValue(3)

		config1 := HTTPClientConfig{
			Timeout: &timeout,
			Transport: &gohttpc.HTTPTransportConfig{
				MaxIdleConns: &maxIdleConns,
			},
			TLS: &TLSConfig{
				MinVersion: "1.2",
			},
			Retry: &HTTPRetryConfig{
				MaxAttempts: &maxAttempts,
			},
		}

		config2 := HTTPClientConfig{
			Timeout: &timeout,
			Transport: &gohttpc.HTTPTransportConfig{
				MaxIdleConns: &maxIdleConns,
			},
			TLS: &TLSConfig{
				MinVersion: "1.2",
			},
			Retry: &HTTPRetryConfig{
				MaxAttempts: &maxAttempts,
			},
		}

		if !config1.Equal(config2) {
			t.Error("expected Equal to return true for identical configs")
		}
	})
}

func TestNewClientFromConfig(t *testing.T) {
	t.Run("creates client with empty config", func(t *testing.T) {
		config := &HTTPClientConfig{}

		client, err := NewClientFromConfig(config)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if client == nil {
			t.Error("expected client to be created")
		}

		defer client.Close()
	})

	t.Run("creates client with timeout", func(t *testing.T) {
		timeout := goutils.Duration(time.Second * 30)
		config := &HTTPClientConfig{
			Timeout: &timeout,
		}

		client, err := NewClientFromConfig(config)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if client == nil {
			t.Error("expected client to be created")
		}

		defer client.Close()
	})

	t.Run("creates client with retry policy", func(t *testing.T) {
		maxAttempts := goenvconf.NewEnvIntValue(3)
		config := &HTTPClientConfig{
			Retry: &HTTPRetryConfig{
				MaxAttempts: &maxAttempts,
			},
		}

		client, err := NewClientFromConfig(config)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if client == nil {
			t.Error("expected client to be created")
		}

		defer client.Close()
	})

	t.Run("returns error when retry policy is invalid", func(t *testing.T) {
		maxAttempts := goenvconf.NewEnvIntValue(-1)
		config := &HTTPClientConfig{
			Retry: &HTTPRetryConfig{
				MaxAttempts: &maxAttempts,
			},
		}

		client, err := NewClientFromConfig(config)

		if err == nil {
			t.Error("expected error for invalid retry policy")
		}

		if client != nil {
			t.Error("expected client to be nil")
		}
	})

	t.Run("creates client with authentication", func(t *testing.T) {
		config := &HTTPClientConfig{
			Authentication: &authc.HTTPClientAuthConfig{
				HTTPClientAuthenticatorConfig: &basicauth.BasicAuthConfig{
					Type:     authscheme.BasicAuthScheme,
					Username: goenvconf.NewEnvStringValue("testuser"),
					Password: goenvconf.NewEnvStringValue("testpass"),
				},
			},
		}

		client, err := NewClientFromConfig(config)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if client == nil {
			t.Error("expected client to be created")
		}

		defer client.Close()
	})

	t.Run("creates client with custom options", func(t *testing.T) {
		config := &HTTPClientConfig{}

		client, err := NewClientFromConfig(
			config,
			gohttpc.WithHTTPClient(http.DefaultClient),
			gohttpc.WithTimeout(time.Second*10),
		)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if client == nil {
			t.Error("expected client to be created")
		}

		defer client.Close()
	})
}

func TestNewHTTPClientFromConfig(t *testing.T) {
	t.Run("returns existing HTTP client when no transport or TLS config", func(t *testing.T) {
		existingClient := &http.Client{}
		options := &gohttpc.ClientOptions{
			HTTPClient: existingClient,
		}

		config := &HTTPClientConfig{}

		client, err := NewHTTPClientFromConfig(config, options)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if client != existingClient {
			t.Error("expected to return existing HTTP client")
		}
	})

	t.Run("creates new HTTP client with transport config", func(t *testing.T) {
		maxIdleConns := 50
		config := &HTTPClientConfig{
			Transport: &gohttpc.HTTPTransportConfig{
				MaxIdleConns: &maxIdleConns,
			},
		}

		options := gohttpc.NewClientOptions()

		client, err := NewHTTPClientFromConfig(config, options)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if client == nil {
			t.Error("expected client to be created")
		}

		transport, ok := client.Transport.(*http.Transport)
		if !ok {
			t.Error("expected transport to be *http.Transport")
		}

		if transport.MaxIdleConns != maxIdleConns {
			t.Errorf("expected MaxIdleConns %d, got %d", maxIdleConns, transport.MaxIdleConns)
		}
	})

	t.Run("creates new HTTP client with TLS config", func(t *testing.T) {
		config := &HTTPClientConfig{
			TLS: &TLSConfig{
				MinVersion: "1.2",
			},
		}

		options := gohttpc.NewClientOptions()

		client, err := NewHTTPClientFromConfig(config, options)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if client == nil {
			t.Error("expected client to be created")
		}

		transport, ok := client.Transport.(*http.Transport)
		if !ok {
			t.Error("expected transport to be *http.Transport")
		}

		if transport.TLSClientConfig == nil {
			t.Error("expected TLS config to be set")
		}
	})

	t.Run("returns error when TLS config is invalid", func(t *testing.T) {
		config := &HTTPClientConfig{
			TLS: &TLSConfig{
				MinVersion: "invalid",
			},
		}

		options := gohttpc.NewClientOptions()

		client, err := NewHTTPClientFromConfig(config, options)

		if err == nil {
			t.Error("expected error for invalid TLS config")
		}

		if client != nil {
			t.Error("expected client to be nil")
		}
	})

	t.Run("preserves CheckRedirect and Jar from existing client", func(t *testing.T) {
		existingClient := &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}

		options := &gohttpc.ClientOptions{
			HTTPClient: existingClient,
		}

		config := &HTTPClientConfig{
			Transport: &gohttpc.HTTPTransportConfig{},
		}

		client, err := NewHTTPClientFromConfig(config, options)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if client.CheckRedirect == nil {
			t.Error("expected CheckRedirect to be preserved")
		}
	})
}
