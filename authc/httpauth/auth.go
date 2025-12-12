// Package httpauth implements authentication interfaces for the http security scheme.
package httpauth

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/relychan/gohttpc/authc/authscheme"
)

// HTTPCredential presents a header authentication credential.
type HTTPCredential struct {
	location authscheme.TokenLocation
	value    string
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

	if options == nil || options.CustomEnvGetter == nil {
		options = authscheme.NewHTTPClientAuthenticatorOptions()
	}

	getter := options.CustomEnvGetter(ctx)

	value, err := config.Value.GetCustom(getter)
	if err != nil {
		return nil, fmt.Errorf("failed to get HTTP credential: %w", err)
	}

	result := &HTTPCredential{
		location: authscheme.TokenLocation{
			In:     authscheme.InHeader,
			Name:   header,
			Scheme: strings.ToLower(scheme),
		},
		value: value,
	}

	return result, nil
}

// Authenticate the credential into the incoming request.
func (hc *HTTPCredential) Authenticate(
	req *http.Request,
	options ...authscheme.AuthenticateOption,
) error {
	_, err := hc.location.InjectRequest(req, hc.value, false)

	return err
}

// Equal checks if the target value is equal.
func (hc *HTTPCredential) Equal(target *HTTPCredential) bool {
	if target == nil {
		return false
	}

	return hc.value == target.value &&
		hc.location.Equal(target.location)
}

// Close terminates internal processes before destroyed.
func (*HTTPCredential) Close() error {
	return nil
}
