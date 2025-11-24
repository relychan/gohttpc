package authc

import (
	"fmt"

	"github.com/relychan/gohttpc/authc/authscheme"
	"github.com/relychan/gohttpc/authc/basicauth"
	"github.com/relychan/gohttpc/authc/httpauth"
	"github.com/relychan/gohttpc/authc/oauth2scheme"
)

// NewAuthenticatorFromConfig creates an authenticator from the configuration.
func NewAuthenticatorFromConfig(
	config *HTTPClientAuthConfig,
) (authscheme.HTTPClientAuthenticator, error) {
	switch conf := config.HTTPClientAuthenticatorConfig.(type) {
	case *basicauth.BasicAuthConfig:
		return basicauth.NewBasicCredential(conf)
	case *httpauth.HTTPAuthConfig:
		return httpauth.NewHTTPCredential(conf)
	case *oauth2scheme.OAuth2Config:
		return oauth2scheme.NewOAuth2Client(conf)
	default:
		return nil, fmt.Errorf("%w: %s", errUnsupportedSecurityScheme, config.GetType())
	}
}
