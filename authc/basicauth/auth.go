// Package basicauth implements authentication interfaces for the basic security scheme.
package basicauth

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"sync"

	"github.com/relychan/gohttpc/authc/authscheme"
)

// BasicCredential represents the basic authentication credential.
type BasicCredential struct {
	config   *BasicAuthConfig
	options  *authscheme.HTTPClientAuthenticatorOptions
	username string
	password string
	mu       sync.RWMutex
}

var _ authscheme.HTTPClientAuthenticator = (*BasicCredential)(nil)

// NewBasicCredential creates a new BasicCredential instance.
func NewBasicCredential(
	ctx context.Context,
	config *BasicAuthConfig,
	options *authscheme.HTTPClientAuthenticatorOptions,
) (*BasicCredential, error) {
	result := &BasicCredential{
		config:  config,
		options: options,
	}

	return result, result.doReload(ctx)
}

// Authenticate the credential into the incoming request.
func (bc *BasicCredential) Authenticate(
	req *http.Request,
	options ...authscheme.AuthenticateOption,
) error {
	return bc.inject(req, bc.username, bc.password)
}

// Reload reloads the configuration and state.
func (bc *BasicCredential) Reload(ctx context.Context) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	return bc.doReload(ctx)
}

func (bc *BasicCredential) doReload(ctx context.Context) error {
	getter := bc.options.CustomEnvGetter(ctx)

	user, err := bc.config.Username.GetCustom(getter)
	if err != nil {
		return fmt.Errorf("failed to create basic credential. Invalid username: %w", err)
	}

	password, err := bc.config.Password.GetCustom(getter)
	if err != nil {
		return fmt.Errorf("failed to create basic credential. Invalid password: %w", err)
	}

	if user == "" && password == "" {
		return authscheme.ErrAuthCredentialEmpty
	}

	bc.username = user
	bc.password = password

	return nil
}

func (bc *BasicCredential) inject(req *http.Request, user, password string) error {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	if bc.config.Header != "" {
		var userInfo *url.Userinfo

		if password != "" {
			userInfo = url.UserPassword(user, password)
		} else {
			userInfo = url.User(user)
		}

		b64Value := base64.StdEncoding.EncodeToString([]byte(userInfo.String()))
		req.Header.Set(bc.config.Header, "Basic "+b64Value)
	} else {
		req.SetBasicAuth(user, password)
	}

	return nil
}
