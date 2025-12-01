package authscheme

import (
	"testing"
)

func TestHTTPClientAuthType_Validate(t *testing.T) {
	t.Run("validates supported auth types", func(t *testing.T) {
		supportedTypes := []HTTPClientAuthType{
			APIKeyScheme,
			BasicAuthScheme,
			HTTPAuthScheme,
			OAuth2Scheme,
		}

		for _, authType := range supportedTypes {
			err := authType.Validate()
			if err != nil {
				t.Errorf("expected %s to be valid, got error: %v", authType, err)
			}
		}
	})

	t.Run("returns error for unsupported auth type", func(t *testing.T) {
		invalidType := HTTPClientAuthType("invalid")
		err := invalidType.Validate()

		if err == nil {
			t.Error("expected error for invalid auth type")
		}
	})

	t.Run("returns error for empty auth type", func(t *testing.T) {
		emptyType := HTTPClientAuthType("")
		err := emptyType.Validate()

		if err == nil {
			t.Error("expected error for empty auth type")
		}
	})
}

func TestParseHTTPClientAuthType(t *testing.T) {
	t.Run("parses valid auth types", func(t *testing.T) {
		testCases := []struct {
			input    string
			expected HTTPClientAuthType
		}{
			{"apiKey", APIKeyScheme},
			{"basic", BasicAuthScheme},
			{"http", HTTPAuthScheme},
			{"oauth2", OAuth2Scheme},
		}

		for _, tc := range testCases {
			result, err := ParseHTTPClientAuthType(tc.input)
			if err != nil {
				t.Errorf("unexpected error for %s: %v", tc.input, err)
			}

			if result != tc.expected {
				t.Errorf("expected %s, got %s", tc.expected, result)
			}
		}
	})

	t.Run("returns error for invalid auth type", func(t *testing.T) {
		_, err := ParseHTTPClientAuthType("invalid")

		if err == nil {
			t.Error("expected error for invalid auth type")
		}
	})
}

func TestGetSupportedHTTPClientAuthTypes(t *testing.T) {
	t.Run("returns all supported auth types", func(t *testing.T) {
		types := GetSupportedHTTPClientAuthTypes()

		if len(types) != 4 {
			t.Errorf("expected 4 supported types, got %d", len(types))
		}

		expectedTypes := []HTTPClientAuthType{
			APIKeyScheme,
			HTTPAuthScheme,
			BasicAuthScheme,
			OAuth2Scheme,
		}

		for _, expected := range expectedTypes {
			found := false
			for _, actual := range types {
				if actual == expected {
					found = true
					break
				}
			}

			if !found {
				t.Errorf("expected type %s not found in supported types", expected)
			}
		}
	})
}

func TestAuthLocation_Validate(t *testing.T) {
	t.Run("validates supported auth locations", func(t *testing.T) {
		supportedLocations := []AuthLocation{
			InHeader,
			InQuery,
			InCookie,
		}

		for _, location := range supportedLocations {
			err := location.Validate()
			if err != nil {
				t.Errorf("expected %s to be valid, got error: %v", location, err)
			}
		}
	})

	t.Run("returns error for unsupported auth location", func(t *testing.T) {
		invalidLocation := AuthLocation("invalid")
		err := invalidLocation.Validate()

		if err == nil {
			t.Error("expected error for invalid auth location")
		}
	})
}

func TestParseAuthLocation(t *testing.T) {
	t.Run("parses valid auth locations", func(t *testing.T) {
		testCases := []struct {
			input    string
			expected AuthLocation
		}{
			{"header", InHeader},
			{"query", InQuery},
			{"cookie", InCookie},
		}

		for _, tc := range testCases {
			result, err := ParseAuthLocation(tc.input)
			if err != nil {
				t.Errorf("unexpected error for %s: %v", tc.input, err)
			}

			if result != tc.expected {
				t.Errorf("expected %s, got %s", tc.expected, result)
			}
		}
	})
}
