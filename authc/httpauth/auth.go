// Package httpauth implements authentication interfaces for the http security scheme.
package httpauth

import (
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
func NewHTTPCredential(config *HTTPAuthConfig) (*HTTPCredential, error) {
	value, err := config.Value.Get()
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP credential: %w", err)
	}

	if config.In == "" {
		config.In = authscheme.InHeader
	}

	header := config.Name

	if header == "" {
		header = "Authorization"
	}

	scheme := strings.TrimSpace(config.Scheme)

	return &HTTPCredential{
		location: authscheme.TokenLocation{
			In:     authscheme.InHeader,
			Name:   header,
			Scheme: strings.ToLower(scheme),
		},
		value: value,
	}, nil
}

// Authenticate the credential into the incoming request.
func (hc *HTTPCredential) Authenticate(
	req *http.Request,
	options ...authscheme.AuthenticateOption,
) error {
	_, err := hc.location.InjectRequest(req, hc.value, false)

	return err
}
