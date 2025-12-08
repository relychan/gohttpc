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

// OAuth2Credential represent the client of the OAuth2 client credentials.
type OAuth2Credential struct {
	oauth2Config *clientcredentials.Config
	location     *authscheme.TokenLocation
	config       *OAuth2Config
	options      *authscheme.HTTPClientAuthenticatorOptions
	mu           sync.RWMutex
}

var _ authscheme.HTTPClientAuthenticator = (*OAuth2Credential)(nil)

// NewOAuth2Credential creates an OAuth2 client from the security scheme.
func NewOAuth2Credential(
	ctx context.Context,
	config *OAuth2Config,
	options *authscheme.HTTPClientAuthenticatorOptions,
) (*OAuth2Credential, error) {
	location := config.TokenLocation
	if location == nil {
		location = &authscheme.TokenLocation{
			In:   authscheme.InHeader,
			Name: "Authorization",
		}
	}

	if options == nil || options.CustomEnvGetter == nil {
		options = authscheme.NewHTTPClientAuthenticatorOptions()
	}

	client := &OAuth2Credential{
		config:   config,
		location: location,
		options:  options,
	}

	return client, client.doReload(ctx)
}

// Authenticate the credential into the incoming request.
func (oc *OAuth2Credential) Authenticate(
	req *http.Request,
	options ...authscheme.AuthenticateOption,
) error {
	oauth2Config := oc.getOAuth2Config()
	if oauth2Config == nil {
		return authscheme.ErrAuthCredentialEmpty
	}

	// get the token from client credentials
	token, err := oauth2Config.Token(req.Context())
	if err != nil {
		return err
	}

	_, err = oc.location.InjectRequest(req, token.AccessToken, false)

	return err
}

// Close terminates internal processes before destroyed.
func (*OAuth2Credential) Close() error {
	return nil
}

// Reload reloads the configuration and state.
func (oc *OAuth2Credential) Reload(ctx context.Context) error {
	return oc.doReload(ctx)
}

func (oc *OAuth2Credential) doReload(ctx context.Context) error { //nolint:funlen
	oc.mu.RLock()
	getter := oc.options.CustomEnvGetter(ctx)
	flow := oc.config.Flows.ClientCredentials
	oc.mu.RUnlock()

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

	oauth2Config := &clientcredentials.Config{
		ClientID:       clientID,
		ClientSecret:   clientSecret,
		Scopes:         scopes,
		TokenURL:       tokenURL.String(),
		EndpointParams: endpointParams,
	}

	oc.mu.Lock()
	defer oc.mu.Unlock()

	if oc.location.Scheme == "" {
		token, err := oauth2Config.Token(ctx)
		if err != nil {
			return err
		}

		oc.location.Scheme = strings.ToLower(token.Type())
	}

	oc.oauth2Config = oauth2Config

	return nil
}

func (oc *OAuth2Credential) getOAuth2Config() *clientcredentials.Config {
	oc.mu.RLock()
	defer oc.mu.RUnlock()

	return oc.oauth2Config
}
