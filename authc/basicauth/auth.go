// Package basicauth implements authentication interfaces for the basic security scheme.
package basicauth

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"

	"github.com/relychan/gohttpc/authc/authscheme"
)

// BasicCredential represents the basic authentication credential.
type BasicCredential struct {
	username string
	password string
	header   string
}

var _ authscheme.HTTPClientAuthenticator = (*BasicCredential)(nil)

// NewBasicCredential creates a new BasicCredential instance.
func NewBasicCredential(config *BasicAuthConfig) (*BasicCredential, error) {
	user, err := config.Username.Get()
	if err != nil {
		return nil, fmt.Errorf("failed to create basic credential. Invalid username: %w", err)
	}

	password, err := config.Password.Get()
	if err != nil {
		return nil, fmt.Errorf("failed to create basic credential. Invalid password: %w", err)
	}

	result := &BasicCredential{
		username: user,
		password: password,
		header:   config.Header,
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

func (bc *BasicCredential) inject(req *http.Request, user, password string) error {
	if bc.username == "" && bc.password == "" {
		return authscheme.ErrAuthCredentialEmpty
	}

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
