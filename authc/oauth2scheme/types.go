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
