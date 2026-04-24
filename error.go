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

package gohttpc

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/relychan/goutils"
	"github.com/relychan/goutils/httperror"
	"github.com/relychan/goutils/httpheader"
)

var (
	// ErrResponseBodyNoContent occurs when the response body has no content.
	ErrResponseBodyNoContent = errors.New("response body has no content")
	// ErrResponseBodyAlreadyRead occurs when the response body was already read.
	ErrResponseBodyAlreadyRead = errors.New("response body was already read")
	// ErrRequestMethodRequired occurs when the request method is null.
	ErrRequestMethodRequired = errors.New("request method is required")
	// ErrRequestAlreadyExecuted occurs when the request was already executed.
	ErrRequestAlreadyExecuted = errors.New("request was already executed")
)

// httpErrorFromResponse creates an error from the HTTP response.
func httpErrorFromResponse(resp *http.Response) *goutils.HTTPErrorWithExtensions {
	if resp.Body == nil {
		return httpErrorFromNoContentResponse(resp)
	}

	contentTypes := resp.Header[httpheader.ContentType]

	if len(contentTypes) > 0 && httpheader.IsContentTypeJSON(contentTypes[0]) {
		var httpError goutils.HTTPErrorWithExtensions

		err := json.NewDecoder(resp.Body).Decode(&httpError)

		goutils.CloseResponse(resp)

		if err != nil {
			return httpErrorFromNoContentResponse(resp)
		}

		if httpError.Status == 0 {
			httpError.Status = resp.StatusCode
		}

		if httpError.Title == "" {
			httpError.Title = resp.Status
		}

		httpError.Extensions["headers"] = goutils.ExtractHeaders(resp.Header)

		return &httpError
	}

	result := httpErrorFromNoContentResponse(resp)
	rawBody, readErr := io.ReadAll(resp.Body)

	goutils.CatchWarnErrorFunc(resp.Body.Close)

	if readErr == nil {
		result.Detail = string(rawBody)
	} else {
		result.Extensions["read_error"] = readErr
	}

	return result
}

func httpErrorFromNoContentResponse(resp *http.Response) *goutils.HTTPErrorWithExtensions {
	return &goutils.HTTPErrorWithExtensions{
		HTTPError: httperror.HTTPError{
			Status: resp.StatusCode,
			Title:  resp.Status,
		},
		Extensions: map[string]any{
			"headers": goutils.ExtractHeaders(resp.Header),
		},
	}
}
