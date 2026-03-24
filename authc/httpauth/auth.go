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

// Package httpauth implements authentication interfaces for the http security scheme.
package httpauth

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/relychan/gohttpc/authc/authscheme"
)

// HTTPCredential presents a header authentication credential.
type HTTPCredential struct {
	location authscheme.TokenLocation
	value    string
}

var _ authscheme.HTTPClientAuthenticator = (*HTTPCredential)(nil)

// NewHTTPCredential creates a new HTTPCredential instance.
func NewHTTPCredential(
	config *HTTPAuthConfig,
	options *authscheme.HTTPClientAuthenticatorOptions,
) (*HTTPCredential, error) {
	header := strings.TrimSpace(config.TokenLocation.Name)
	if header == "" {
		header = "Authorization"
	}

	scheme := strings.TrimSpace(config.TokenLocation.Scheme)

	if options == nil {
		options = authscheme.NewHTTPClientAuthenticatorOptions()
	}

	value, err := config.Value.GetCustom(options.GetEnvFunc())
	if err != nil {
		return nil, fmt.Errorf("failed to get HTTP credential: %w", err)
	}

	result := &HTTPCredential{
		location: authscheme.TokenLocation{
			In:     config.TokenLocation.In,
			Name:   header,
			Scheme: strings.ToLower(scheme),
		},
		value: value,
	}

	return result, nil
}

// Authenticate the credential into the incoming request.
func (hc *HTTPCredential) Authenticate(
	req *http.Request,
	options ...authscheme.AuthenticateOption,
) error {
	_, err := hc.location.InjectRequest(req, hc.value, false)

	return err
}

// Equal checks if the target value is equal.
func (hc HTTPCredential) Equal(target HTTPCredential) bool {
	return hc.value == target.value &&
		hc.location.Equal(target.location)
}

// Close terminates internal processes before destroyed.
func (*HTTPCredential) Close() error {
	return nil
}
