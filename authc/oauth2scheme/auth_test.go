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
	"testing"

	"github.com/relychan/gohttpc/authc/authscheme"
	"github.com/relychan/goutils"
)

func TestNewOAuth2Credential(t *testing.T) {
	t.Run("creates config with correct type", func(t *testing.T) {
		flows := OAuth2Flows{
			ClientCredentials: ClientCredentialsOAuthFlow{
				TokenURL:     ptrEnvString("https://example.com/token"),
				ClientID:     ptrEnvString("client-id"),
				ClientSecret: ptrEnvString("client-secret"),
			},
		}

		config := NewOAuth2Config(flows)

		if config.Type != authscheme.OAuth2Scheme {
			t.Errorf("expected type %s, got %s", authscheme.OAuth2Scheme, config.Type)
		}

		if config.Flows.ClientCredentials.TokenURL == nil {
			t.Error("expected TokenURL to be set")
		}

		cred, err := NewOAuth2Credential(config, nil)
		if err != nil {
			t.Errorf("expected nil error, got: %s", err)
		}

		if !cred.Equal(*cred) {
			t.Errorf("expected self equality, got 'false'")
		}

		if goutils.EqualPtr(cred, nil) {
			t.Errorf("expected not equal, got 'true'")
		}
	})
}
