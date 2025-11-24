// Package authscheme defines types and interfaces for security schemes.
package authscheme

import (
	"fmt"
	"net/http"
	"slices"
)

// HTTPClientAuthenticator abstracts an interface for injecting authentication value into HTTP requests.
type HTTPClientAuthenticator interface {
	// Authenticate the credential into the incoming request.
	Authenticate(req *http.Request) error
}

// HTTPClientAuthenticatorConfig abstracts an interface of the HTTP client authentication config.
type HTTPClientAuthenticatorConfig interface {
	// GetType gets the type of security scheme.
	GetType() HTTPClientAuthType
	// Validate checks if the instance is valid.
	Validate(strict bool) error
}

// HTTPClientAuthType represents the authentication scheme enum.
type HTTPClientAuthType string

const (
	APIKeyScheme    HTTPClientAuthType = "apiKey"
	BasicAuthScheme HTTPClientAuthType = "basic"
	HTTPAuthScheme  HTTPClientAuthType = "http"
	OAuth2Scheme    HTTPClientAuthType = "oauth2"
)

var enumValueHTTPClientAuthTypes = []HTTPClientAuthType{
	APIKeyScheme,
	HTTPAuthScheme,
	BasicAuthScheme,
	OAuth2Scheme,
}

var errInvalidHTTPClientAuthType = fmt.Errorf(
	"invalid HTTPClientAuthType. Expected %v",
	enumValueHTTPClientAuthTypes,
)

// Validate checks if the security scheme type is valid.
func (j HTTPClientAuthType) Validate() error {
	if !slices.Contains(GetSupportedHTTPClientAuthTypes(), j) {
		return fmt.Errorf(
			"%w, got <%s>",
			errInvalidHTTPClientAuthType,
			j,
		)
	}

	return nil
}

// ParseHTTPClientAuthType parses SecurityScheme from string.
func ParseHTTPClientAuthType(value string) (HTTPClientAuthType, error) {
	result := HTTPClientAuthType(value)

	return result, result.Validate()
}

// GetSupportedHTTPClientAuthTypes get the list of supported security scheme types.
func GetSupportedHTTPClientAuthTypes() []HTTPClientAuthType {
	return enumValueHTTPClientAuthTypes
}

// AuthLocation represents the location enum for setting authentication value.
type AuthLocation string

const (
	InHeader AuthLocation = "header"
	InQuery  AuthLocation = "query"
	InCookie AuthLocation = "cookie"
)

var (
	enumValuesAuthLocations = []AuthLocation{InHeader, InQuery, InCookie}
	errInvalidAuthLocation  = fmt.Errorf(
		"invalid AuthLocation. Expected %v",
		enumValuesAuthLocations,
	)
)

// Validate checks if the security scheme type is valid.
func (j AuthLocation) Validate() error {
	if !slices.Contains(GetSupportedAuthLocations(), j) {
		return fmt.Errorf(
			"%w, got <%s>",
			errInvalidAuthLocation,
			j,
		)
	}

	return nil
}

// ParseAuthLocation parses the auth location from string.
func ParseAuthLocation(value string) (AuthLocation, error) {
	result := AuthLocation(value)

	return result, result.Validate()
}

// GetSupportedAuthLocations get the list of supported auth locations.
func GetSupportedAuthLocations() []AuthLocation {
	return enumValuesAuthLocations
}
