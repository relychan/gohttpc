package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/invopop/jsonschema"
	"github.com/relychan/gohttpc/authc/authscheme"
	"github.com/relychan/gohttpc/authc/basicauth"
	"github.com/relychan/gohttpc/authc/httpauth"
	"github.com/relychan/gohttpc/authc/oauth2scheme"
	"github.com/relychan/gohttpc/httpconfig"
	"github.com/relychan/goutils"
)

func main() {
	err := jsonSchemaConfiguration()
	if err != nil {
		panic(fmt.Errorf("failed to write jsonschema for HTTPClientConfig: %w", err))
	}
}

func jsonSchemaConfiguration() error {
	r := new(jsonschema.Reflector)

	err := r.AddGoComments("github.com/relychan/gohttpc/httpconfig", "httpconfig", jsonschema.WithFullComment())
	if err != nil {
		return err
	}

	reflectSchema := r.Reflect(httpconfig.HTTPClientConfig{})

	for _, externalType := range []any{
		basicauth.BasicAuthConfig{},
		httpauth.HTTPAuthConfig{},
		authscheme.TokenLocation{},
		oauth2scheme.OAuth2Config{},
	} {
		externalSchema := r.Reflect(externalType)

		for key, def := range externalSchema.Definitions {
			if _, ok := reflectSchema.Definitions[key]; !ok {
				reflectSchema.Definitions[key] = def
			}
		}
	}

	// custom schema types
	reflectSchema.Definitions["Duration"] = &jsonschema.Schema{
		Type:        "string",
		Description: "Duration string",
		Pattern:     "^((([0-9]+h)?([0-9]+m)?([0-9]+s))|(([0-9]+h)?([0-9]+m))|([0-9]+h))$",
	}
	reflectSchema.Definitions["HTTPClientAuthConfig"] = &jsonschema.Schema{
		Description: "Define authentication configurations",
		OneOf: []*jsonschema.Schema{
			{
				Description: "Configuration for the basic authentication",
				Ref:         "#/$defs/BasicAuthConfig",
			},
			{
				Description: "Configuration for the http and API Key authentication",
				Ref:         "#/$defs/HTTPAuthConfig",
			},
			{
				Description: "Configuration for the OAuth2 authentication",
				Ref:         "#/$defs/OAuth2Config",
			},
		},
	}

	reflectSchema.Definitions["AuthLocation"] = &jsonschema.Schema{
		Type:        "string",
		Description: "Defines the location enum for setting authentication value",
		Enum:        goutils.ToAnySlice(authscheme.GetSupportedAuthLocations()),
	}
	inSchema := &jsonschema.Schema{
		Description: "The location enum for setting authentication value",
		Ref:         "#/$defs/AuthLocation",
	}
	reflectSchema.Definitions["TokenLocation"].Properties.Set("in", inSchema)
	reflectSchema.Definitions["HTTPAuthConfig"].Properties.Set("in", inSchema)

	schemaBytes, err := json.MarshalIndent(reflectSchema, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join("jsonschema", "gohttpc.schema.json"), schemaBytes, 0o644) //nolint:gosec
}
