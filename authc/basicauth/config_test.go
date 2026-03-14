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

package basicauth

import (
	"testing"

	"github.com/hasura/goenvconf"
	"github.com/relychan/gohttpc/authc/authscheme"
)

func TestNewBasicAuthConfig(t *testing.T) {
	t.Run("creates config with correct type", func(t *testing.T) {
		username := goenvconf.NewEnvStringValue("user")
		password := goenvconf.NewEnvStringValue("pass")

		config := NewBasicAuthConfig(&username, &password)

		if config.Type != authscheme.BasicAuthScheme {
			t.Errorf("expected type %s, got %s", authscheme.BasicAuthScheme, config.Type)
		}

		usernameVal, _ := config.Username.Get()
		if usernameVal != "user" {
			t.Errorf("expected username 'user', got '%s'", usernameVal)
		}

		passwordVal, _ := config.Password.Get()
		if passwordVal != "pass" {
			t.Errorf("expected password 'pass', got '%s'", passwordVal)
		}
	})
}

func TestBasicAuthConfig_GetType(t *testing.T) {
	t.Run("returns basic auth scheme type", func(t *testing.T) {
		config := &BasicAuthConfig{}

		if config.GetType() != authscheme.BasicAuthScheme {
			t.Errorf("expected type %s, got %s", authscheme.BasicAuthScheme, config.GetType())
		}
	})
}

func TestBasicAuthConfig_Validate(t *testing.T) {
	t.Run("validates successfully with valid config in non-strict mode", func(t *testing.T) {
		config := &BasicAuthConfig{
			Type:     authscheme.BasicAuthScheme,
			Username: new(goenvconf.NewEnvStringValue("user")),
			Password: new(goenvconf.NewEnvStringValue("pass")),
		}

		err := config.Validate(false)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("validates successfully with valid config in strict mode", func(t *testing.T) {
		config := &BasicAuthConfig{
			Type:     authscheme.BasicAuthScheme,
			Username: new(goenvconf.NewEnvStringValue("user")),
			Password: new(goenvconf.NewEnvStringValue("pass")),
		}

		err := config.Validate(true)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("returns error when type does not match", func(t *testing.T) {
		config := &BasicAuthConfig{
			Type:     authscheme.HTTPAuthScheme,
			Username: new(goenvconf.NewEnvStringValue("user")),
			Password: new(goenvconf.NewEnvStringValue("pass")),
		}

		err := config.Validate(false)

		if err == nil {
			t.Error("expected error for mismatched type")
		}
	})

	t.Run("returns error when username is empty in strict mode", func(t *testing.T) {
		config := &BasicAuthConfig{
			Type:     authscheme.BasicAuthScheme,
			Username: &goenvconf.EnvString{},
			Password: new(goenvconf.NewEnvStringValue("pass")),
		}

		err := config.Validate(true)
		if err != nil {
			t.Error("expected nil error for empty username in strict mode")
		}
	})

	t.Run("returns error when password is empty in strict mode", func(t *testing.T) {
		config := &BasicAuthConfig{
			Type:     authscheme.BasicAuthScheme,
			Username: new(goenvconf.NewEnvStringValue("user")),
			Password: &goenvconf.EnvString{},
		}

		err := config.Validate(true)
		if err != nil {
			t.Error("expected nil error for empty password in strict mode")
		}
	})

	t.Run("allows empty username and password in non-strict mode", func(t *testing.T) {
		config := &BasicAuthConfig{
			Type:     authscheme.BasicAuthScheme,
			Username: &goenvconf.EnvString{},
			Password: &goenvconf.EnvString{},
		}

		err := config.Validate(false)
		if err != nil {
			t.Errorf("unexpected error in non-strict mode: %v", err)
		}
	})

	t.Run("validates config with custom header", func(t *testing.T) {
		config := &BasicAuthConfig{
			Type:     authscheme.BasicAuthScheme,
			Header:   "X-Custom-Auth",
			Username: new(goenvconf.NewEnvStringValue("user")),
			Password: new(goenvconf.NewEnvStringValue("pass")),
		}

		err := config.Validate(true)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("validates config with description", func(t *testing.T) {
		config := &BasicAuthConfig{
			Type:        authscheme.BasicAuthScheme,
			Username:    new(goenvconf.NewEnvStringValue("user")),
			Password:    new(goenvconf.NewEnvStringValue("pass")),
			Description: "Basic authentication for API",
		}

		err := config.Validate(true)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}
