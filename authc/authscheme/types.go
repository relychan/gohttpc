// Package authscheme defines types and interfaces for security schemes.
package authscheme

import (
	"fmt"
	"net/http"
	"slices"

	"github.com/hasura/goenvconf"
	"github.com/relychan/goutils"
)

// HTTPClientAuthenticator abstracts an interface for injecting authentication value into HTTP requests.
type HTTPClientAuthenticator interface {
	// Authenticate the credential into the incoming request.
	Authenticate(req *http.Request, options ...AuthenticateOption) error
	// Close terminates internal processes before destroyed.
	Close() error
}

// HTTPClientAuthenticatorConfig abstracts an interface of the HTTP client authentication config.
type HTTPClientAuthenticatorConfig interface {
	goutils.IsZeroer
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
			"%w; got: %s",
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

var enumValuesAuthLocations = []AuthLocation{InHeader, InQuery, InCookie}

// Validate checks if the security scheme type is valid.
func (j AuthLocation) Validate() error {
	if !slices.Contains(GetSupportedAuthLocations(), j) {
		return fmt.Errorf(
			"%w; got: %s",
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

// AuthenticateOptions represents custom options for the authentication.
type AuthenticateOptions struct {
	Name string
}

// AuthenticateOption adds custom options to the authenticate request.
type AuthenticateOption func(*AuthenticateOptions)

// WithAuthenticationName creates an option to set the authentication name.
func WithAuthenticationName(name string) AuthenticateOption {
	return func(ao *AuthenticateOptions) {
		ao.Name = name
	}
}

// HTTPClientAuthenticatorOptions define common options for the authenticator client.
type HTTPClientAuthenticatorOptions struct {
	GetEnv goenvconf.GetEnvFunc
}

// NewHTTPClientAuthenticatorOptions creates a new [HTTPClientAuthenticatorOptions] instance.
func NewHTTPClientAuthenticatorOptions(
	options ...HTTPClientAuthenticatorOption,
) *HTTPClientAuthenticatorOptions {
	result := &HTTPClientAuthenticatorOptions{}

	for _, opt := range options {
		opt(result)
	}

	return result
}

// GetEnvFunc returns the getEnv function. Fallback to GetOSEnv.
func (hcao HTTPClientAuthenticatorOptions) GetEnvFunc() goenvconf.GetEnvFunc {
	if hcao.GetEnv == nil {
		return goenvconf.GetOSEnv
	}

	return hcao.GetEnv
}

// HTTPClientAuthenticatorOption defines a function to modify [HTTPClientAuthenticatorOptions].
type HTTPClientAuthenticatorOption func(*HTTPClientAuthenticatorOptions)

// WithGetEnvFunc returns a function to set the GetEnvFunc getter to [HTTPClientAuthenticatorOptions].
func WithGetEnvFunc(
	getFunc goenvconf.GetEnvFunc,
) HTTPClientAuthenticatorOption {
	return func(hao *HTTPClientAuthenticatorOptions) {
		if getFunc == nil {
			return
		}

		hao.GetEnv = getFunc
	}
}
