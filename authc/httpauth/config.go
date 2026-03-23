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
	// The location where the auth credential will be injected.
	TokenLocation authscheme.TokenLocation `json:"tokenLocation" yaml:"tokenLocation"`
	// Value of the access token.
	Value goenvconf.EnvString `json:"value" yaml:"value"`
	// Type of the http authenticator.
	Type authscheme.HTTPClientAuthType `json:"type" jsonschema:"enum=http" yaml:"type"`
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

// IsZero if the current instance is empty.
func (bac HTTPAuthConfig) IsZero() bool {
	return bac.Type == 0 &&
		bac.Value.IsZero() &&
		bac.TokenLocation.IsZero()
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

	err := tac.TokenLocation.Validate()
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
