package authscheme

import (
	"errors"
	"testing"
)

func TestErrAuthCredentialEmpty(t *testing.T) {
	t.Run("error message is correct", func(t *testing.T) {
		expected := "the auth credential is empty"
		if ErrAuthCredentialEmpty.Error() != expected {
			t.Errorf("expected error message '%s', got '%s'", expected, ErrAuthCredentialEmpty.Error())
		}
	})
}

func TestNewRequiredSecurityFieldError(t *testing.T) {
	t.Run("creates error with correct message", func(t *testing.T) {
		err := NewRequiredSecurityFieldError(BasicAuthScheme, "username")

		if err == nil {
			t.Fatal("expected error to be created")
		}

		expectedMsg := "required field username for the basic client auth scheme"
		if err.Error() != expectedMsg {
			t.Errorf("expected error message '%s', got '%s'", expectedMsg, err.Error())
		}
	})

	t.Run("wraps errRequiredSecurityField", func(t *testing.T) {
		err := NewRequiredSecurityFieldError(OAuth2Scheme, "clientId")

		if !errors.Is(err, errRequiredSecurityField) {
			t.Error("expected error to wrap errRequiredSecurityField")
		}
	})

	t.Run("creates error for different auth schemes", func(t *testing.T) {
		testCases := []struct {
			scheme HTTPClientAuthType
			field  string
		}{
			{BasicAuthScheme, "password"},
			{HTTPAuthScheme, "value"},
			{OAuth2Scheme, "tokenUrl"},
			{APIKeyScheme, "name"},
		}

		for _, tc := range testCases {
			err := NewRequiredSecurityFieldError(tc.scheme, tc.field)

			if err == nil {
				t.Errorf("expected error for scheme %s and field %s", tc.scheme, tc.field)
			}
		}
	})
}

func TestNewUnmatchedSecuritySchemeError(t *testing.T) {
	t.Run("creates error with correct message", func(t *testing.T) {
		err := NewUnmatchedSecuritySchemeError(BasicAuthScheme, HTTPAuthScheme)

		if err == nil {
			t.Fatal("expected error to be created")
		}

		expectedMsg := "client auth type does not match, expected `basic`, got `http`"
		if err.Error() != expectedMsg {
			t.Errorf("expected error message '%s', got '%s'", expectedMsg, err.Error())
		}
	})

	t.Run("wraps errUnmatchedSecurityScheme", func(t *testing.T) {
		err := NewUnmatchedSecuritySchemeError(OAuth2Scheme, BasicAuthScheme)

		if !errors.Is(err, errUnmatchedSecurityScheme) {
			t.Error("expected error to wrap errUnmatchedSecurityScheme")
		}
	})

	t.Run("creates error for different scheme combinations", func(t *testing.T) {
		testCases := []struct {
			expected HTTPClientAuthType
			got      HTTPClientAuthType
		}{
			{BasicAuthScheme, OAuth2Scheme},
			{HTTPAuthScheme, BasicAuthScheme},
			{OAuth2Scheme, APIKeyScheme},
			{APIKeyScheme, HTTPAuthScheme},
		}

		for _, tc := range testCases {
			err := NewUnmatchedSecuritySchemeError(tc.expected, tc.got)

			if err == nil {
				t.Errorf("expected error for expected=%s and got=%s", tc.expected, tc.got)
			}
		}
	})
}
