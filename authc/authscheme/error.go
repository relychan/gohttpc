package authscheme

import (
	"errors"
	"fmt"
)

// ErrAuthCredentialEmpty occurs when the auth credential is empty.
var ErrAuthCredentialEmpty = errors.New("the auth credential is empty")

var (
	errUnmatchedSecurityScheme = errors.New("client auth type does not match")
	errRequiredSecurityField   = errors.New("required field")
)

// NewRequiredSecurityFieldError creates an error for required field in the security scheme config.
func NewRequiredSecurityFieldError(scheme HTTPClientAuthType, name string) error {
	return fmt.Errorf("%w %s for the %s client auth scheme", errRequiredSecurityField, name, scheme)
}

// NewUnmatchedSecuritySchemeError creates an error for unexpected security scheme type.
func NewUnmatchedSecuritySchemeError(expected HTTPClientAuthType, got HTTPClientAuthType) error {
	return fmt.Errorf("%w, expected `%s`, got `%s`", errUnmatchedSecurityScheme, expected, got)
}
