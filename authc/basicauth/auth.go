// Package basicauth implements authentication interfaces for the basic security scheme.
package basicauth

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"

	"github.com/relychan/gohttpc/authc/authscheme"
)

// BasicCredential represents the basic authentication credential.
type BasicCredential struct {
	header   string
	username string
	password string
}

var _ authscheme.HTTPClientAuthenticator = (*BasicCredential)(nil)

// NewBasicCredential creates a new BasicCredential instance.
func NewBasicCredential(
	ctx context.Context,
	config *BasicAuthConfig,
	options *authscheme.HTTPClientAuthenticatorOptions,
) (*BasicCredential, error) {
	if options == nil || options.CustomEnvGetter == nil {
		options = authscheme.NewHTTPClientAuthenticatorOptions()
	}

	getter := options.CustomEnvGetter(ctx)

	user, err := config.Username.GetCustom(getter)
	if err != nil {
		return nil, fmt.Errorf("failed to load basic credential. Invalid username: %w", err)
	}

	password, err := config.Password.GetCustom(getter)
	if err != nil {
		return nil, fmt.Errorf("failed to load basic credential. Invalid password: %w", err)
	}

	if user == "" && password == "" {
		return nil, authscheme.ErrAuthCredentialEmpty
	}

	result := &BasicCredential{
		header:   config.Header,
		username: user,
		password: password,
	}

	return result, nil
}

// Authenticate the credential into the incoming request.
func (bc *BasicCredential) Authenticate(
	req *http.Request,
	options ...authscheme.AuthenticateOption,
) error {
	return bc.inject(req, bc.username, bc.password)
}

// Equal checks if the target value is equal.
func (bc *BasicCredential) Equal(target *BasicCredential) bool {
	if target == nil {
		return false
	}

	return bc.header == target.header &&
		bc.username == target.username &&
		bc.password == target.password
}

// Close terminates internal processes before destroyed.
func (*BasicCredential) Close() error {
	return nil
}

func (bc *BasicCredential) inject(req *http.Request, user, password string) error {
	if bc.header != "" {
		var userInfo *url.Userinfo

		if password != "" {
			userInfo = url.UserPassword(user, password)
		} else {
			userInfo = url.User(user)
		}

		b64Value := base64.StdEncoding.EncodeToString([]byte(userInfo.String()))
		req.Header.Set(bc.header, "Basic "+b64Value)
	} else {
		req.SetBasicAuth(user, password)
	}

	return nil
}
