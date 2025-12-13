package authc

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/relychan/gohttpc/authc/authscheme"
	"github.com/relychan/gohttpc/authc/basicauth"
	"github.com/relychan/gohttpc/authc/httpauth"
	"github.com/relychan/gohttpc/authc/oauth2scheme"
	"go.yaml.in/yaml/v4"
)

var (
	errSecuritySchemeDefinitionRequired = errors.New("security scheme definition is required")
	errUnsupportedSecurityScheme        = errors.New("unsupported security scheme")
)

// HTTPClientAuthConfig contains authentication configurations.
// The schema follows [OpenAPI 3] specification with extensions.
//
// [OpenAPI 3]: https://swagger.io/docs/specification/authentication
type HTTPClientAuthConfig struct {
	authscheme.HTTPClientAuthenticatorConfig `yaml:",inline"`
}

type httpClientAuthConfig struct {
	Type authscheme.HTTPClientAuthType `json:"type" yaml:"type"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *HTTPClientAuthConfig) UnmarshalJSON(b []byte) error {
	var rawScheme httpClientAuthConfig

	err := json.Unmarshal(b, &rawScheme)
	if err != nil {
		return err
	}

	err = rawScheme.Type.Validate()
	if err != nil {
		return err
	}

	switch rawScheme.Type {
	case authscheme.BasicAuthScheme:
		var config basicauth.BasicAuthConfig

		err := json.Unmarshal(b, &config)
		if err != nil {
			return err
		}

		j.HTTPClientAuthenticatorConfig = &config
	case authscheme.HTTPAuthScheme:
		var config httpauth.HTTPAuthConfig

		err := json.Unmarshal(b, &config)
		if err != nil {
			return err
		}

		j.HTTPClientAuthenticatorConfig = &config
	case authscheme.OAuth2Scheme:
		var config oauth2scheme.OAuth2Config

		err := json.Unmarshal(b, &config)
		if err != nil {
			return err
		}

		j.HTTPClientAuthenticatorConfig = &config
	default:
		return fmt.Errorf("%w: %s", errUnsupportedSecurityScheme, rawScheme.Type)
	}

	return nil
}

// MarshalJSON implements json.Marshaler.
func (j HTTPClientAuthConfig) MarshalJSON() ([]byte, error) {
	return json.Marshal(j.HTTPClientAuthenticatorConfig)
}

// UnmarshalYAML implements yaml.Unmarshaler.
func (j *HTTPClientAuthConfig) UnmarshalYAML(value *yaml.Node) error {
	var rawScheme httpClientAuthConfig

	err := value.Decode(&rawScheme)
	if err != nil {
		return err
	}

	err = rawScheme.Type.Validate()
	if err != nil {
		return err
	}

	switch rawScheme.Type {
	case authscheme.BasicAuthScheme:
		var config basicauth.BasicAuthConfig

		err := value.Decode(&config)
		if err != nil {
			return err
		}

		j.HTTPClientAuthenticatorConfig = &config
	case authscheme.HTTPAuthScheme:
		var config httpauth.HTTPAuthConfig

		err := value.Decode(&config)
		if err != nil {
			return err
		}

		j.HTTPClientAuthenticatorConfig = &config
	case authscheme.OAuth2Scheme:
		var config oauth2scheme.OAuth2Config

		err := value.Decode(&config)
		if err != nil {
			return err
		}

		j.HTTPClientAuthenticatorConfig = &config
	default:
		return fmt.Errorf("%w: %s", errUnsupportedSecurityScheme, rawScheme.Type)
	}

	return nil
}

// Validate if the current instance is valid.
func (j *HTTPClientAuthConfig) Validate(strict bool) error {
	if j.HTTPClientAuthenticatorConfig == nil {
		return errSecuritySchemeDefinitionRequired
	}

	return j.HTTPClientAuthenticatorConfig.Validate(strict)
}

// IsZero if the current instance is empty.
func (j *HTTPClientAuthConfig) IsZero() bool {
	return j.HTTPClientAuthenticatorConfig == nil
}

// Equal checks if the target value is equal.
func (j HTTPClientAuthConfig) Equal(target HTTPClientAuthConfig) bool {
	return reflect.DeepEqual(j.HTTPClientAuthenticatorConfig, target.HTTPClientAuthenticatorConfig)
}
