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

package authscheme

import (
	"testing"
)

func TestHTTPClientAuthType_Validate(t *testing.T) {
	t.Run("validates supported auth types", func(t *testing.T) {
		supportedTypes := []HTTPClientAuthType{
			BasicAuthScheme,
			HTTPAuthScheme,
			OAuth2Scheme,
		}

		for _, authType := range supportedTypes {
			if !authType.IsValid() {
				t.Errorf("expected to be valid, got false")
			}
		}
	})
}

func TestParseHTTPClientAuthType(t *testing.T) {
	t.Run("parses valid auth types", func(t *testing.T) {
		testCases := []struct {
			input    string
			expected HTTPClientAuthType
		}{
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

func TestAuthLocation_Validate(t *testing.T) {
	t.Run("validates supported auth locations", func(t *testing.T) {
		supportedLocations := []AuthLocation{
			InHeader,
			InQuery,
			InCookie,
		}

		for _, location := range supportedLocations {
			if !location.IsValid() {
				t.Errorf("expected %s to be valid, got false", location)
			}
		}
	})

	t.Run("returns error for unsupported auth location", func(t *testing.T) {
		invalidLocation := AuthLocation(255)

		if invalidLocation.IsValid() {
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
