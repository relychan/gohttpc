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

package authc

import (
	"fmt"

	"github.com/relychan/gohttpc/authc/authscheme"
	"github.com/relychan/gohttpc/authc/basicauth"
	"github.com/relychan/gohttpc/authc/httpauth"
	"github.com/relychan/gohttpc/authc/oauth2scheme"
)

// NewAuthenticatorFromConfig creates an authenticator from the configuration.
func NewAuthenticatorFromConfig(
	config *HTTPClientAuthConfig,
	options *authscheme.HTTPClientAuthenticatorOptions,
) (authscheme.HTTPClientAuthenticator, error) {
	switch conf := config.HTTPClientAuthenticatorConfig.(type) {
	case *basicauth.BasicAuthConfig:
		return basicauth.NewBasicCredential(conf, options)
	case *httpauth.HTTPAuthConfig:
		return httpauth.NewHTTPCredential(conf, options)
	case *oauth2scheme.OAuth2Config:
		return oauth2scheme.NewOAuth2Credential(conf, options)
	default:
		return nil, fmt.Errorf("%w: %s", errUnsupportedSecurityScheme, config.GetType())
	}
}
