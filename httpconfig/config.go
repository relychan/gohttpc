package httpconfig

import (
	"context"
	"net/http"
	"time"

	"github.com/relychan/gohttpc"
	"github.com/relychan/gohttpc/authc"
	"github.com/relychan/goutils"
)

// HTTPClientConfig contains configurations to create client.
type HTTPClientConfig struct {
	// Default maximum timeout duration that is applied for all requests.
	Timeout *goutils.Duration `json:"timeout,omitempty" jsonschema:"oneof_ref=#/$defs/Duration,oneof_type=null" yaml:"timeout,omitempty"`
	// Transport stores the http.Transport configuration for the http client.
	Transport *gohttpc.HTTPTransportConfig `json:"transport,omitempty" yaml:"transport,omitempty"`
	// The transport layer security (LTS) configuration for the mutualTLS authentication.
	TLS *TLSConfig `json:"tls,omitempty" yaml:"tls,omitempty"`
	// Retry policy of client requests.
	Retry *HTTPRetryConfig `json:"retry,omitempty" yaml:"retry,omitempty"`
	// Authentication configuration.
	Authentication *authc.HTTPClientAuthConfig `json:"authentication,omitempty" yaml:"authentication,omitempty"`
}

// IsZero if the current instance is empty.
func (c *HTTPClientConfig) IsZero() bool {
	return (c.Timeout == nil || *c.Timeout <= 0) &&
		goutils.IsZeroPtr(c.Transport) &&
		goutils.IsZeroPtr(c.TLS) &&
		goutils.IsZeroPtr(c.Retry) &&
		c.Authentication == nil
}

// Equal checks if the target value is equal.
func (j HTTPClientConfig) Equal(target HTTPClientConfig) bool {
	return goutils.EqualComparablePtr(j.Timeout, target.Timeout) &&
		goutils.EqualPtr(j.Transport, target.Transport) &&
		goutils.EqualPtr(j.TLS, target.TLS) &&
		goutils.EqualPtr(j.Retry, target.Retry) &&
		goutils.EqualPtr(j.Authentication, target.Authentication)
}

// NewClientFromConfig creates a HTTP client wrapper with configuration.
func NewClientFromConfig(
	ctx context.Context,
	config *HTTPClientConfig,
	options ...gohttpc.ClientOption,
) (*gohttpc.Client, error) {
	if config == nil {
		config = &HTTPClientConfig{}
	}

	if config.Timeout != nil && *config.Timeout > 0 {
		options = append(options, gohttpc.WithTimeout(time.Duration(*config.Timeout)))
	}

	opts := gohttpc.NewClientOptions(options...)

	if config.Retry != nil {
		retry, err := config.Retry.ToRetryPolicy() //nolint:bodyclose
		if err != nil {
			return nil, err
		}

		opts.Retry = retry
	}

	if config.Authentication != nil {
		authenticator, err := authc.NewAuthenticatorFromConfig(
			config.Authentication,
			&opts.HTTPClientAuthenticatorOptions,
		)
		if err != nil {
			return nil, err
		}

		opts.Authenticator = authenticator
	}

	httpClient, err := NewHTTPClientFromConfig(config, opts)
	if err != nil {
		return nil, err
	}

	opts.HTTPClient = httpClient

	return gohttpc.NewClientWithOptions(opts), nil
}

// NewHTTPClientFromConfig creates a HTTP client with configuration.
func NewHTTPClientFromConfig(
	config *HTTPClientConfig,
	options *gohttpc.ClientOptions,
) (*http.Client, error) {
	if config.Transport == nil && config.TLS == nil && options.HTTPClient != nil {
		return options.HTTPClient, nil
	}

	newTransport := gohttpc.TransportFromConfig(config.Transport, options)

	if config.TLS != nil {
		tlsConfig, err := loadTLSConfig(config.TLS)
		if err != nil {
			return nil, err
		}

		newTransport.TLSClientConfig = tlsConfig
	}

	httpClient := &http.Client{
		Transport: newTransport,
	}

	if options.HTTPClient != nil {
		httpClient.CheckRedirect = options.HTTPClient.CheckRedirect
		httpClient.Jar = options.HTTPClient.Jar
	}

	return httpClient, nil
}
