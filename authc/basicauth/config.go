package basicauth

import (
	"github.com/hasura/goenvconf"
	"github.com/relychan/gohttpc/authc/authscheme"
)

// BasicAuthConfig contains configurations for the [basic] authentication.
//
// [basic]: https://swagger.io/docs/specification/authentication/basic-authentication
type BasicAuthConfig struct {
	Type authscheme.HTTPClientAuthType `json:"type" jsonschema:"enum=basic" yaml:"type"`
	// Header where the credential will be set.
	Header string `json:"header,omitempty" yaml:"header,omitempty"`
	// Username to authenticate.
	Username goenvconf.EnvString `json:"username" yaml:"username"`
	// Password to authenticate.
	Password goenvconf.EnvString `json:"password" yaml:"password"`
	// A description for security scheme.
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
}

var _ authscheme.HTTPClientAuthenticatorConfig = (*BasicAuthConfig)(nil)

// NewBasicAuthConfig creates a new BasicAuthConfig instance.
func NewBasicAuthConfig(username, password goenvconf.EnvString) *BasicAuthConfig {
	return &BasicAuthConfig{
		Type:     authscheme.BasicAuthScheme,
		Username: username,
		Password: password,
	}
}

// Validate if the current instance is valid.
func (ss BasicAuthConfig) Validate(strict bool) error {
	authType := ss.GetType()

	if ss.Type != authType {
		return authscheme.NewUnmatchedSecuritySchemeError(authType, ss.Type)
	}

	if !strict {
		return nil
	}

	if ss.Username.IsZero() {
		return authscheme.NewRequiredSecurityFieldError(authType, "username")
	}

	if ss.Password.IsZero() {
		return authscheme.NewRequiredSecurityFieldError(authType, "password")
	}

	return nil
}

// GetType get the type of security scheme.
func (ss BasicAuthConfig) GetType() authscheme.HTTPClientAuthType {
	return authscheme.BasicAuthScheme
}
