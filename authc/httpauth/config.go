package httpauth

import (
	"github.com/hasura/goenvconf"
	"github.com/relychan/gohttpc/authc/authscheme"
)

// HTTPAuthConfig contains configurations for http authentication
// If the scheme is [bearer], the authenticator follows OpenAPI 3 specification.
//
// [bearer]: https://swagger.io/docs/specification/authentication/bearer-authentication
type HTTPAuthConfig struct {
	authscheme.TokenLocation `yaml:",inline"`

	Type authscheme.HTTPClientAuthType `json:"type" jsonschema:"enum=http" yaml:"type"`
	// Value of the access token.
	Value goenvconf.EnvString `json:"value" yaml:"value"`
	// A description for security scheme.
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
}

var _ authscheme.HTTPClientAuthenticatorConfig = (*HTTPAuthConfig)(nil)

// NewHTTPAuthConfig creates a new HTTPAuthConfig instance.
func NewHTTPAuthConfig(
	location authscheme.TokenLocation,
	value goenvconf.EnvString,
) *HTTPAuthConfig {
	return &HTTPAuthConfig{
		Type:          authscheme.HTTPAuthScheme,
		Value:         value,
		TokenLocation: location,
	}
}

// Equal checks if the target value is equal.
func (tac HTTPAuthConfig) Equal(target HTTPAuthConfig) bool {
	return tac.Type == target.Type &&
		tac.Value.Equal(target.Value) &&
		tac.TokenLocation.Equal(target.TokenLocation)
}

// Validate if the current instance is valid.
func (tac HTTPAuthConfig) Validate(strict bool) error {
	authType := tac.GetType()

	if tac.Type != authType {
		return authscheme.NewUnmatchedSecuritySchemeError(authType, tac.Type)
	}

	if tac.Name == "" {
		return authscheme.NewRequiredSecurityFieldError(authType, "name")
	}

	err := tac.In.Validate()
	if err != nil {
		return err
	}

	if !strict {
		return nil
	}

	if tac.Value.IsZero() {
		return authscheme.NewRequiredSecurityFieldError(authType, "value")
	}

	return nil
}

// GetType get the type of security scheme.
func (ss HTTPAuthConfig) GetType() authscheme.HTTPClientAuthType {
	return authscheme.HTTPAuthScheme
}
