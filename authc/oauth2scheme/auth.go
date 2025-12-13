// Package oauth2scheme implements authentication interfaces for OAuth2 security scheme.
package oauth2scheme

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/hasura/goenvconf"
	"github.com/relychan/gohttpc/authc/authscheme"
	"github.com/relychan/goutils"
	"golang.org/x/oauth2/clientcredentials"
)

// OAuth2Credential represent the client of the OAuth2 client credentials.
type OAuth2Credential struct {
	oauth2Config *clientcredentials.Config
	location     authscheme.TokenLocation
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

	oauth2Config, err := newClientCredentialsConfig(ctx, config, options)
	if err != nil {
		return nil, err
	}

	client := &OAuth2Credential{
		location:     *location,
		oauth2Config: oauth2Config,
	}

	return client, nil
}

// Authenticate the credential into the incoming request.
func (oc *OAuth2Credential) Authenticate(
	req *http.Request,
	options ...authscheme.AuthenticateOption,
) error {
	oauth2Config := oc.oauth2Config
	if oauth2Config == nil {
		return authscheme.ErrAuthCredentialEmpty
	}

	// get the token from client credentials
	token, err := oauth2Config.Token(req.Context())
	if err != nil {
		return err
	}

	location := oc.location

	if location.Scheme == "" {
		location.Scheme = strings.ToLower(token.Type())
	}

	_, err = location.InjectRequest(req, token.AccessToken, false)

	return err
}

// Equal checks if the target value is equal.
func (oc OAuth2Credential) Equal(target OAuth2Credential) bool {
	return oc.location.Equal(target.location) &&
		EqualClientCredentialsConfig(oc.oauth2Config, target.oauth2Config)
}

// Close terminates internal processes before destroyed.
func (*OAuth2Credential) Close() error {
	return nil
}

// EqualClientCredentialsConfig checks if both client credentials configs are equal.
func EqualClientCredentialsConfig(a, b *clientcredentials.Config) bool {
	if a == nil && b == nil {
		return true
	}

	if a == nil || b == nil {
		return false
	}

	isEqual := a.AuthStyle == b.AuthStyle &&
		a.ClientID == b.ClientID &&
		a.ClientSecret == b.ClientSecret &&
		a.TokenURL == b.TokenURL &&
		len(a.EndpointParams) == len(b.EndpointParams) &&
		goutils.EqualSliceSorted(a.Scopes, b.Scopes)

	if !isEqual {
		return false
	}

	if len(a.EndpointParams) == 0 {
		return true
	}

	for key, values := range a.EndpointParams {
		targetValues := b.EndpointParams[key]
		if !goutils.EqualSliceSorted(values, targetValues) {
			return false
		}
	}

	return true
}

func newClientCredentialsConfig(
	ctx context.Context,
	config *OAuth2Config,
	options *authscheme.HTTPClientAuthenticatorOptions,
) (*clientcredentials.Config, error) {
	getter := options.CustomEnvGetter(ctx)
	flow := config.Flows.ClientCredentials

	rawTokenURL, err := flow.TokenURL.GetCustom(getter)
	if err != nil {
		return nil, fmt.Errorf("tokenUrl: %w", err)
	}

	tokenURL, err := goutils.ParseRelativeOrHTTPURL(rawTokenURL)
	if err != nil {
		return nil, fmt.Errorf("tokenUrl: %w", err)
	}

	scopes := make([]string, 0, len(flow.Scopes))
	for scope := range flow.Scopes {
		scopes = append(scopes, scope)
	}

	clientID, err := flow.ClientID.GetCustom(getter)
	if err != nil {
		return nil, fmt.Errorf("clientId: %w", err)
	}

	clientSecret, err := flow.ClientSecret.GetCustom(getter)
	if err != nil {
		return nil, fmt.Errorf("clientSecret: %w", err)
	}

	var endpointParams url.Values

	for key, envValue := range flow.EndpointParams {
		value, err := envValue.GetCustom(getter)
		if err != nil && !errors.Is(err, goenvconf.ErrEnvironmentVariableValueRequired) {
			return nil, fmt.Errorf("endpointParams[%s]: %w", key, err)
		}

		if value != "" {
			endpointParams.Set(key, value)
		}
	}

	return &clientcredentials.Config{
		ClientID:       clientID,
		ClientSecret:   clientSecret,
		Scopes:         scopes,
		TokenURL:       tokenURL.String(),
		EndpointParams: endpointParams,
	}, nil
}
