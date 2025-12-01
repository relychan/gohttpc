package oauth2scheme

import (
	"errors"
	"testing"
)

func TestOAuthFlowType_Validate(t *testing.T) {
	t.Run("validates supported OAuth flow types", func(t *testing.T) {
		testCases := []struct {
			name     string
			flowType OAuthFlowType
		}{
			{"authorization code flow", AuthorizationCodeFlow},
			{"implicit flow", ImplicitFlow},
			{"password flow", PasswordFlow},
			{"client credentials flow", ClientCredentialsFlow},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				err := tc.flowType.Validate()
				if err != nil {
					t.Errorf("expected no error for %s, got %v", tc.flowType, err)
				}
			})
		}
	})

	t.Run("returns error for unsupported OAuth flow type", func(t *testing.T) {
		flowType := OAuthFlowType("unsupported")

		err := flowType.Validate()

		if err == nil {
			t.Error("expected error for unsupported OAuth flow type")
		}

		if !errors.Is(err, errInvalidOAuthFlowType) {
			t.Errorf("expected error to wrap errInvalidOAuthFlowType, got %v", err)
		}
	})

	t.Run("returns error for empty OAuth flow type", func(t *testing.T) {
		flowType := OAuthFlowType("")

		err := flowType.Validate()

		if err == nil {
			t.Error("expected error for empty OAuth flow type")
		}
	})
}

func TestParseOAuthFlowType(t *testing.T) {
	t.Run("parses valid OAuth flow types", func(t *testing.T) {
		testCases := []struct {
			name     string
			input    string
			expected OAuthFlowType
		}{
			{"authorization code flow", "authorizationCode", AuthorizationCodeFlow},
			{"implicit flow", "implicit", ImplicitFlow},
			{"password flow", "password", PasswordFlow},
			{"client credentials flow", "clientCredentials", ClientCredentialsFlow},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result, err := ParseOAuthFlowType(tc.input)

				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}

				if result != tc.expected {
					t.Errorf("expected %s, got %s", tc.expected, result)
				}
			})
		}
	})

	t.Run("returns error for invalid OAuth flow type", func(t *testing.T) {
		_, err := ParseOAuthFlowType("invalid")

		if err == nil {
			t.Error("expected error for invalid OAuth flow type")
		}

		if !errors.Is(err, errInvalidOAuthFlowType) {
			t.Errorf("expected error to wrap errInvalidOAuthFlowType, got %v", err)
		}
	})

	t.Run("returns error for empty string", func(t *testing.T) {
		_, err := ParseOAuthFlowType("")

		if err == nil {
			t.Error("expected error for empty string")
		}
	})
}
