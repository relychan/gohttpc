package httpconfig

import (
	"context"
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

	t.Run("returns false when Retry is set", func(t *testing.T) {
		config := HTTPClientConfig{
			Retry: &HTTPRetryConfig{},
		}

		if !config.IsZero() {
			t.Error("expected IsZero to return true")
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

func TestNewClientFromConfig(t *testing.T) {
	t.Run("creates client with empty config", func(t *testing.T) {
		config := &HTTPClientConfig{}

		client, err := NewClientFromConfig(context.TODO(), config)

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

		client, err := NewClientFromConfig(context.TODO(), config)

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

		client, err := NewClientFromConfig(context.TODO(), config)

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

		client, err := NewClientFromConfig(context.TODO(), config)

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

		client, err := NewClientFromConfig(context.TODO(), config)

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
			context.TODO(),
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
