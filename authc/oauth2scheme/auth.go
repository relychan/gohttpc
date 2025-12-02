// Package oauth2scheme implements authentication interfaces for OAuth2 security scheme.
package oauth2scheme

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/relychan/gohttpc/authc/authscheme"
	"github.com/relychan/goutils"
	"golang.org/x/oauth2/clientcredentials"
)

// OAuth2Client represent the client of the OAuth2 client credentials.
type OAuth2Client struct {
	oauth2Config *clientcredentials.Config
	location     authscheme.TokenLocation
}

var _ authscheme.HTTPClientAuthenticator = (*OAuth2Client)(nil)

// NewOAuth2Client creates an OAuth2 client from the security scheme.
func NewOAuth2Client(config *OAuth2Config) (*OAuth2Client, error) {
	location := config.TokenLocation
	if location == nil {
		location = &authscheme.TokenLocation{
			In:   authscheme.InHeader,
			Name: "Authorization",
		}
	}

	flow := config.Flows.ClientCredentials

	rawTokenURL, err := flow.TokenURL.Get()
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

	clientID, err := flow.ClientID.Get()
	if err != nil {
		return nil, fmt.Errorf("clientId: %w", err)
	}

	clientSecret, err := flow.ClientSecret.Get()
	if err != nil {
		return nil, fmt.Errorf("clientSecret: %w", err)
	}

	var endpointParams url.Values

	for key, envValue := range flow.EndpointParams {
		value, err := envValue.GetOrDefault("")
		if err != nil {
			return nil, fmt.Errorf("endpointParams[%s]: %w", key, err)
		}

		if value != "" {
			endpointParams.Set(key, value)
		}
	}

	conf := &clientcredentials.Config{
		ClientID:       clientID,
		ClientSecret:   clientSecret,
		Scopes:         scopes,
		TokenURL:       tokenURL.String(),
		EndpointParams: endpointParams,
	}

	return &OAuth2Client{
		oauth2Config: conf,
		location:     *location,
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
