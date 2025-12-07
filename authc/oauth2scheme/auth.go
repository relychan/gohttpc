// Package oauth2scheme implements authentication interfaces for OAuth2 security scheme.
package oauth2scheme

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/hasura/goenvconf"
	"github.com/relychan/gohttpc/authc/authscheme"
	"github.com/relychan/goutils"
	"golang.org/x/oauth2/clientcredentials"
)

// OAuth2Client represent the client of the OAuth2 client credentials.
type OAuth2Client struct {
	oauth2Config *clientcredentials.Config
	location     *authscheme.TokenLocation
	config       *OAuth2Config
	options      *authscheme.HTTPClientAuthenticatorOptions
	mu           sync.RWMutex
}

var _ authscheme.HTTPClientAuthenticator = (*OAuth2Client)(nil)

// NewOAuth2Client creates an OAuth2 client from the security scheme.
func NewOAuth2Client(
	ctx context.Context,
	config *OAuth2Config,
	options *authscheme.HTTPClientAuthenticatorOptions,
) (*OAuth2Client, error) {
	location := config.TokenLocation
	if location == nil {
		location = &authscheme.TokenLocation{
			In:   authscheme.InHeader,
			Name: "Authorization",
		}
	}

	return &OAuth2Client{
		config:   config,
		location: location,
		options:  options,
	}, nil
}

// Authenticate the credential into the incoming request.
func (oc *OAuth2Client) Authenticate(
	req *http.Request,
	options ...authscheme.AuthenticateOption,
) error {
	if oc.oauth2Config == nil {
		return authscheme.ErrAuthCredentialEmpty
	}

	// get the token from client credentials
	token, err := oc.oauth2Config.Token(req.Context())
	if err != nil {
		return err
	}

	if oc.location.Scheme == "" {
		oc.location.Scheme = strings.ToLower(token.Type())
	}

	_, err = oc.location.InjectRequest(req, token.AccessToken, false)

	return err
}

// Reload reloads the configuration and state.
func (oc *OAuth2Client) Reload(ctx context.Context) error {
	oc.mu.Lock()
	defer oc.mu.Unlock()

	return oc.doReload(ctx)
}

func (oc *OAuth2Client) doReload(ctx context.Context) error {
	getter := oc.options.CustomEnvGetter(ctx)

	flow := oc.config.Flows.ClientCredentials

	rawTokenURL, err := flow.TokenURL.GetCustom(getter)
	if err != nil {
		return fmt.Errorf("tokenUrl: %w", err)
	}

	tokenURL, err := goutils.ParseRelativeOrHTTPURL(rawTokenURL)
	if err != nil {
		return fmt.Errorf("tokenUrl: %w", err)
	}

	scopes := make([]string, 0, len(flow.Scopes))
	for scope := range flow.Scopes {
		scopes = append(scopes, scope)
	}

	clientID, err := flow.ClientID.GetCustom(getter)
	if err != nil {
		return fmt.Errorf("clientId: %w", err)
	}

	clientSecret, err := flow.ClientSecret.GetCustom(getter)
	if err != nil {
		return fmt.Errorf("clientSecret: %w", err)
	}

	var endpointParams url.Values

	for key, envValue := range flow.EndpointParams {
		value, err := envValue.GetCustom(getter)
		if err != nil && !errors.Is(err, goenvconf.ErrEnvironmentVariableValueRequired) {
			return fmt.Errorf("endpointParams[%s]: %w", key, err)
		}

		if value != "" {
			endpointParams.Set(key, value)
		}
	}

	oc.oauth2Config = &clientcredentials.Config{
		ClientID:       clientID,
		ClientSecret:   clientSecret,
		Scopes:         scopes,
		TokenURL:       tokenURL.String(),
		EndpointParams: endpointParams,
	}

	return nil
}
