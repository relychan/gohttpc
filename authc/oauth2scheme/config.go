package oauth2scheme

import (
	"errors"

	"github.com/hasura/goenvconf"
	"github.com/relychan/gohttpc/authc/authscheme"
	"github.com/relychan/goutils"
)

var (
	// ErrClientIDRequired represents the client ID required error.
	ErrClientIDRequired = errors.New(
		"clientId is required for the OAuth2 client_credentials flow",
	)
	// ErrClientSecretRequired represents the client secret required error.
	ErrClientSecretRequired = errors.New(
		"clientSecret is required for the OAuth2 client_credentials flow",
	)
	// ErrTokenURLRequired represents the token URL required error.
	ErrTokenURLRequired = errors.New("tokenUrl: value and env are empty")
)

// OAuth2Config contains configurations for OAuth 2.0 with client_credentials type.
type OAuth2Config struct {
	Type authscheme.HTTPClientAuthType `json:"type" jsonschema:"enum=oauth2" yaml:"type"`
	// An object containing configuration information for the flow types supported.
	Flows OAuth2Flows `json:"flows" yaml:"flows"`
	// A description for security scheme.
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	// The location where the auth credential will be injected.
	TokenLocation *authscheme.TokenLocation `json:"tokenLocation,omitempty" yaml:"tokenLocation,omitempty"`
}

var _ authscheme.HTTPClientAuthenticatorConfig = (*OAuth2Config)(nil)

// NewOAuth2Config creates a new OAuth2Config instance.
func NewOAuth2Config(flows OAuth2Flows) *OAuth2Config {
	return &OAuth2Config{
		Type:  authscheme.OAuth2Scheme,
		Flows: flows,
	}
}

// GetType get the type of security scheme.
func (ss OAuth2Config) GetType() authscheme.HTTPClientAuthType {
	return authscheme.OAuth2Scheme
}

// IsZero if the current instance is empty.
func (ss OAuth2Config) IsZero() bool {
	return ss.Type == "" &&
		ss.Flows.IsZero() &&
		ss.Description == "" &&
		ss.TokenLocation == nil
}

// Equal checks if this instance equals the target value.
func (ss OAuth2Config) Equal(target OAuth2Config) bool {
	return ss.Type == target.Type &&
		ss.Flows.Equal(target.Flows) &&
		goutils.EqualPtr(ss.TokenLocation, target.TokenLocation)
}

// Validate if the current instance is valid.
func (ss OAuth2Config) Validate(_ bool) error {
	authType := ss.GetType()

	if ss.Type != authType {
		return authscheme.NewUnmatchedSecuritySchemeError(authType, ss.Type)
	}

	return ss.Flows.ClientCredentials.Validate()
}

// OAuth2Flows contain configuration information for the flow types supported.
type OAuth2Flows struct {
	// OAuth2 flow for client_credentials
	ClientCredentials ClientCredentialsOAuthFlow `json:"clientCredentials" yaml:"clientCredentials"`
}

// IsZero if the current instance is empty.
func (ss OAuth2Flows) IsZero() bool {
	return ss.ClientCredentials.IsZero()
}

// Equal checks if this instance equals the target value.
func (ss OAuth2Flows) Equal(target OAuth2Flows) bool {
	return ss.ClientCredentials.Equal(target.ClientCredentials)
}

// ClientCredentialsOAuthFlow contains flow configurations for OAuth 2.0 client credential flow.
type ClientCredentialsOAuthFlow struct {
	// The token URL to be used for this flow. This MUST be in the form of a URL. The OAuth2 standard requires the use of TLS.
	TokenURL *goenvconf.EnvString `json:"tokenUrl,omitempty" yaml:"tokenUrl,omitempty"`
	// The URL to be used for obtaining refresh tokens. This MUST be in the form of a URL. The OAuth2 standard requires the use of TLS.
	RefreshURL *goenvconf.EnvString `json:"refreshUrl,omitempty" yaml:"refreshUrl,omitempty"`
	// The available scopes for the OAuth2 security scheme. A map between the scope name and a short description for it. The map MAY be empty.
	Scopes         map[string]string              `json:"scopes,omitempty"         yaml:"scopes,omitempty"`
	ClientID       *goenvconf.EnvString           `json:"clientId,omitempty"       yaml:"clientId,omitempty"`
	ClientSecret   *goenvconf.EnvString           `json:"clientSecret,omitempty"   yaml:"clientSecret,omitempty"`
	EndpointParams map[string]goenvconf.EnvString `json:"endpointParams,omitempty" yaml:"endpointParams,omitempty"`
}

// IsZero if the current instance is empty.
func (ss ClientCredentialsOAuthFlow) IsZero() bool {
	return (ss.TokenURL == nil || ss.TokenURL.IsZero()) &&
		(ss.RefreshURL == nil || ss.RefreshURL.IsZero()) &&
		(ss.ClientID == nil || ss.ClientID.IsZero()) &&
		(ss.ClientSecret == nil || ss.ClientSecret.IsZero()) &&
		len(ss.Scopes) == 0 && len(ss.EndpointParams) == 0
}

// Equal checks if this instance equals the target value.
func (ss ClientCredentialsOAuthFlow) Equal(target ClientCredentialsOAuthFlow) bool {
	return goutils.EqualPtr(ss.TokenURL, target.TokenURL) &&
		goutils.EqualPtr(ss.RefreshURL, target.RefreshURL) &&
		goutils.EqualMap(ss.Scopes, target.Scopes, true) &&
		goutils.EqualPtr(ss.ClientID, target.ClientID) &&
		goutils.EqualPtr(ss.ClientSecret, target.ClientSecret) &&
		goutils.EqualMap(ss.EndpointParams, target.EndpointParams, true)
}

// Validate if the current instance is valid.
func (ss ClientCredentialsOAuthFlow) Validate() error {
	if ss.TokenURL == nil || ss.TokenURL.IsZero() {
		return ErrTokenURLRequired
	}

	if ss.ClientID == nil || ss.ClientID.IsZero() {
		return ErrClientIDRequired
	}

	if ss.ClientSecret == nil || ss.ClientSecret.IsZero() {
		return ErrClientSecretRequired
	}

	return nil
}
