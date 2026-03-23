// Copyright 2026 RelyChan Pte. Ltd
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
	"github.com/relychan/goutils"
	"go.yaml.in/yaml/v4"
)

var (
	errSecuritySchemeDefinitionRequired = errors.New("security scheme definition is required")
	errUnsupportedSecurityScheme        = errors.New("unsupported security scheme")
	errAuthConfigTypeRequired           = errors.New(
		"invalid http client auth config: type is required",
	)
)

// HTTPClientAuthConfig contains authentication configurations.
// The schema follows [OpenAPI 3] specification with extensions.
//
// [OpenAPI 3]: https://swagger.io/docs/specification/authentication
type HTTPClientAuthConfig struct {
	authscheme.HTTPClientAuthenticatorConfig `yaml:",inline"`
}

type httpClientAuthConfig struct {
	Type string `json:"type" yaml:"type"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *HTTPClientAuthConfig) UnmarshalJSON(b []byte) error {
	var rawScheme httpClientAuthConfig

	err := json.Unmarshal(b, &rawScheme)
	if err != nil {
		return err
	}

	authType, err := authscheme.ParseHTTPClientAuthType(rawScheme.Type)
	if err != nil {
		return err
	}

	switch authType {
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
	rawAuthType, err := goutils.GetStringValueFromYAMLMap(value, "type")
	if err != nil {
		return err
	}

	if rawAuthType == nil {
		return errAuthConfigTypeRequired
	}

	authType, err := authscheme.ParseHTTPClientAuthType(*rawAuthType)
	if err != nil {
		return err
	}

	switch authType {
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
		return fmt.Errorf("%w: %s", errUnsupportedSecurityScheme, *rawAuthType)
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
func (j HTTPClientAuthConfig) IsZero() bool {
	return j.HTTPClientAuthenticatorConfig == nil
}

// Equal checks if the target value is equal.
func (j HTTPClientAuthConfig) Equal(target HTTPClientAuthConfig) bool {
	return reflect.DeepEqual(j.HTTPClientAuthenticatorConfig, target.HTTPClientAuthenticatorConfig)
}
