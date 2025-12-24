package authscheme

import (
	"fmt"
	"net/http"
)

// TokenLocation contains the configuration for the location of the access token.
type TokenLocation struct {
	// Location where the api key is in.
	In AuthLocation `json:"in" jsonschema:"enum=header,enum=query,enum=cookie,default=header" yaml:"in"`
	// Name of the field to validate, for example, Authorization header.
	Name string `json:"name" yaml:"name" jsonschema:"default=Authorization"`
	// The name of the HTTP Authentication scheme to be used in the Authorization header as defined in RFC7235.
	// The values used SHOULD be registered in the IANA Authentication Scheme registry. https://www.iana.org/assignments/http-authschemes/http-authschemes.xhtml
	// The value is case-insensitive, as defined in RFC7235.
	Scheme string `json:"scheme,omitempty" yaml:"scheme,omitempty"`
}

// IsZero if the current instance is empty.
func (tl TokenLocation) IsZero() bool {
	return tl.In == "" && tl.Name == "" && tl.Scheme == ""
}

// Equal checks if the target value is equal.
func (tl TokenLocation) Equal(target TokenLocation) bool {
	return tl.In == target.In &&
		tl.Name == target.Name &&
		tl.Scheme == target.Scheme
}

// Validate if the current instance is valid.
func (tl TokenLocation) Validate() error {
	err := tl.In.Validate()
	if err != nil {
		return err
	}

	if tl.Name == "" {
		return fmt.Errorf("%w name for the token location", errRequiredSecurityField)
	}

	return nil
}

// InjectRequest injects the authentication token value into the request.
func (tl TokenLocation) InjectRequest(
	req *http.Request,
	value string,
	replace bool,
) (bool, error) {
	value = tl.addTokenSchemeToValue(value)

	switch tl.In {
	case InHeader:
		if !replace && req.Header.Get(tl.Name) != "" {
			return true, nil
		}

		if value != "" {
			req.Header.Set(tl.Name, value)

			return true, nil
		}

		return false, nil
	case InQuery:
		if value == "" {
			return false, nil
		}

		q := req.URL.Query()
		q.Add(tl.Name, value)

		req.URL.RawQuery = q.Encode()

		return true, nil
	case InCookie:
		// Cookies should be forwarded from the frontend client side.
		if !replace {
			for _, cookie := range req.Cookies() {
				if cookie.Name == tl.Name && value != "" {
					return true, nil
				}
			}
		}
	}

	return false, nil
}

func (tl TokenLocation) addTokenSchemeToValue(value string) string {
	switch tl.Scheme {
	case "bearer":
		return "Bearer " + value
	case "basic":
		return "Basic " + value
	case "":
		return value
	default:
		return tl.Scheme + " " + value
	}
}
