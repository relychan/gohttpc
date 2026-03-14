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

package oauth2scheme

import (
	"fmt"
	"slices"
)

// OAuthFlowType represents the OAuth flow type enum.
type OAuthFlowType string

const (
	// AuthorizationCodeFlow represents the OAuth2 Authorization Code flow type.
	AuthorizationCodeFlow OAuthFlowType = "authorizationCode"
	// ImplicitFlow represents the Implicit OAuth2 flow type.
	ImplicitFlow OAuthFlowType = "implicit"
	// PasswordFlow represents the Password OAuth2 flow type.
	PasswordFlow OAuthFlowType = "password"
	// ClientCredentialsFlow represents the client credentials OAuth2 flow type.
	ClientCredentialsFlow OAuthFlowType = "clientCredentials"
)

var enumValueOAuthFlowTypes = []OAuthFlowType{
	AuthorizationCodeFlow,
	ImplicitFlow,
	PasswordFlow,
	ClientCredentialsFlow,
}

var errInvalidOAuthFlowType = fmt.Errorf(
	"invalid OAuthFlowType. Expected %+v",
	enumValueOAuthFlowTypes,
)

// Validate checks if the current value is valid.
func (j OAuthFlowType) Validate() error {
	if !slices.Contains(enumValueOAuthFlowTypes, j) {
		return fmt.Errorf(
			"%w, got <%s>",
			errInvalidOAuthFlowType,
			j,
		)
	}

	return nil
}

// ParseOAuthFlowType parses OAuthFlowType from string.
func ParseOAuthFlowType(value string) (OAuthFlowType, error) {
	result := OAuthFlowType(value)

	return result, result.Validate()
}
