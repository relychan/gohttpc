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
	"errors"
	"fmt"
)

// ErrAuthCredentialEmpty occurs when the auth credential is empty.
var ErrAuthCredentialEmpty = errors.New("the auth credential is empty")

var (
	errUnmatchedSecurityScheme = errors.New("client auth type does not match")
	errRequiredSecurityField   = errors.New("required field")
	errInvalidAuthLocation     = fmt.Errorf(
		"invalid AuthLocation. Expected %v",
		enumValuesAuthLocations,
	)
)

// NewRequiredSecurityFieldError creates an error for required field in the security scheme config.
func NewRequiredSecurityFieldError(scheme HTTPClientAuthType, name string) error {
	return fmt.Errorf("%w %s for the %s client auth scheme", errRequiredSecurityField, name, scheme)
}

// NewUnmatchedSecuritySchemeError creates an error for unexpected security scheme type.
func NewUnmatchedSecuritySchemeError(expected HTTPClientAuthType, got HTTPClientAuthType) error {
	return fmt.Errorf("%w, expected `%s`, got `%s`", errUnmatchedSecurityScheme, expected, got)
}
