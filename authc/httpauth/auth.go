// Package httpauth implements authentication interfaces for the http security scheme.
package httpauth

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/hasura/goenvconf"
	"github.com/relychan/gohttpc/authc/authscheme"
)

// HTTPCredential presents a header authentication credential.
type HTTPCredential struct {
	location authscheme.TokenLocation
	options  *authscheme.HTTPClientAuthenticatorOptions
	value    string
	valueRef goenvconf.EnvString
	mu       sync.RWMutex
}

var _ authscheme.HTTPClientAuthenticator = (*HTTPCredential)(nil)

// NewHTTPCredential creates a new HTTPCredential instance.
func NewHTTPCredential(
	ctx context.Context,
	config *HTTPAuthConfig,
	options *authscheme.HTTPClientAuthenticatorOptions,
) (*HTTPCredential, error) {
	if config.In == "" {
		config.In = authscheme.InHeader
	}

	header := config.Name

	if header == "" {
		header = "Authorization"
	}

	scheme := strings.TrimSpace(config.Scheme)

	result := &HTTPCredential{
		location: authscheme.TokenLocation{
			In:     authscheme.InHeader,
			Name:   header,
			Scheme: strings.ToLower(scheme),
		},
		options:  options,
		valueRef: config.Value,
	}

	return result, result.doReload(ctx)
}

// Authenticate the credential into the incoming request.
func (hc *HTTPCredential) Authenticate(
	req *http.Request,
	options ...authscheme.AuthenticateOption,
) error {
	_, err := hc.location.InjectRequest(req, hc.getValue(), false)

	return err
}

// Close terminates internal processes before destroyed.
func (*HTTPCredential) Close() error {
	return nil
}

// Reload reloads the configuration and state.
func (hc *HTTPCredential) Reload(ctx context.Context) error {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	return hc.doReload(ctx)
}

func (hc *HTTPCredential) doReload(ctx context.Context) error {
	getter := hc.options.CustomEnvGetter(ctx)

	value, err := hc.valueRef.GetCustom(getter)
	if err != nil {
		return fmt.Errorf("failed to get HTTP credential: %w", err)
	}

	hc.value = value

	return nil
}

func (hc *HTTPCredential) getValue() string {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	return hc.value
}
