package basicauth

import (
	"github.com/hasura/goenvconf"
	"github.com/relychan/gohttpc/authc/authscheme"
)

// BasicAuthConfig contains configurations for the [basic] authentication.
//
// [basic]: https://swagger.io/docs/specification/authentication/basic-authentication
type BasicAuthConfig struct {
	// Type of the basic authenticator.
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

// IsZero if the current instance is empty.
func (bac BasicAuthConfig) IsZero() bool {
	return bac.Type == "" &&
		bac.Header == "" &&
		bac.Username.IsZero() &&
		bac.Password.IsZero() &&
		bac.Description == ""
}

// Equal checks if the target value is equal.
func (bac BasicAuthConfig) Equal(target BasicAuthConfig) bool {
	return bac.Type == target.Type &&
		bac.Header == target.Header &&
		bac.Username.Equal(target.Username) &&
		bac.Password.Equal(target.Password)
}

// Validate if the current instance is valid.
func (bac BasicAuthConfig) Validate(strict bool) error {
	authType := bac.GetType()

	if bac.Type != authType {
		return authscheme.NewUnmatchedSecuritySchemeError(authType, bac.Type)
	}

	if !strict {
		return nil
	}

	if bac.Username.IsZero() {
		return authscheme.NewRequiredSecurityFieldError(authType, "username")
	}

	if bac.Password.IsZero() {
		return authscheme.NewRequiredSecurityFieldError(authType, "password")
	}

	return nil
}

// GetType get the type of security scheme.
func (ss BasicAuthConfig) GetType() authscheme.HTTPClientAuthType {
	return authscheme.BasicAuthScheme
}
