package authc

import (
	"context"
	"fmt"

	"github.com/relychan/gohttpc/authc/authscheme"
	"github.com/relychan/gohttpc/authc/basicauth"
	"github.com/relychan/gohttpc/authc/httpauth"
	"github.com/relychan/gohttpc/authc/oauth2scheme"
)

// NewAuthenticatorFromConfig creates an authenticator from the configuration.
func NewAuthenticatorFromConfig(
	ctx context.Context,
	config *HTTPClientAuthConfig,
	options *authscheme.HTTPClientAuthenticatorOptions,
) (authscheme.HTTPClientAuthenticator, error) {
	switch conf := config.HTTPClientAuthenticatorConfig.(type) {
	case *basicauth.BasicAuthConfig:
		return basicauth.NewBasicCredential(ctx, conf, options)
	case *httpauth.HTTPAuthConfig:
		return httpauth.NewHTTPCredential(ctx, conf, options)
	case *oauth2scheme.OAuth2Config:
		return oauth2scheme.NewOAuth2Credential(ctx, conf, options)
	default:
		return nil, fmt.Errorf("%w: %s", errUnsupportedSecurityScheme, config.GetType())
	}
}
