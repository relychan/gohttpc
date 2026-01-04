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
