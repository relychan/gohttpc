package httpconfig

import (
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
	Transport *HTTPTransportConfig `json:"transport,omitempty" yaml:"transport,omitempty"`
	// The transport layer security (LTS) configuration for the mutualTLS authentication.
	TLS *TLSConfig `json:"tls,omitempty" yaml:"tls,omitempty"`
	// Retry policy of client requests.
	Retry *HTTPRetryConfig `json:"retry,omitempty" yaml:"retry,omitempty"`
	// Authentication configuration.
	Authentication *authc.HTTPClientAuthConfig `json:"authentication,omitempty" yaml:"authentication,omitempty"`
}

func (c *HTTPClientConfig) IsZero() bool {
	return (c.Timeout == nil || *c.Timeout <= 0) &&
		c.Transport == nil && c.TLS == nil && c.Retry == nil && c.Authentication == nil
}

// NewClientFromConfig creates a HTTP client with configuration.
func NewClientFromConfig(
	config HTTPClientConfig,
	options ...gohttpc.Option,
) (*gohttpc.Client, error) {
	if config.Timeout != nil && *config.Timeout > 0 {
		options = append(options, gohttpc.WithTimeout(time.Duration(*config.Timeout)))
	}

	if config.Retry != nil {
		retry, err := config.Retry.Validate()
		if err != nil {
			return nil, err
		}

		options = append(options, gohttpc.WithRetry(retry))
	}

	opts := gohttpc.NewClientOptions(options...)

	if config.Authentication != nil {
		authenticator, err := authc.NewAuthenticatorFromConfig(config.Authentication)
		if err != nil {
			return nil, err
		}

		opts.Authenticator = authenticator
	}

	if config.Transport == nil && config.TLS == nil && opts.HTTPClient != nil {
		return gohttpc.NewClientWithOptions(opts), nil
	}

	newTransport := TransportFromConfig(config.Transport, opts)
	httpClient := &http.Client{
		Transport: newTransport,
	}

	if opts.HTTPClient != nil {
		httpClient.CheckRedirect = opts.HTTPClient.CheckRedirect
		httpClient.Jar = opts.HTTPClient.Jar
		httpClient.Timeout = opts.HTTPClient.Timeout
	}

	opts.HTTPClient = httpClient

	if config.TLS != nil {
		tlsConfig, err := loadTLSConfig(config.TLS)
		if err != nil {
			return nil, err
		}

		newTransport.TLSClientConfig = tlsConfig
	}

	return gohttpc.NewClientWithOptions(opts), nil
}
