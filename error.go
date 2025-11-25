package gohttpc

import (
	"errors"
	"net/http"

	"github.com/relychan/goutils"
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
func httpErrorFromResponse(resp *Response) goutils.RFC9457ErrorWithExtensions {
	if resp.body == nil {
		return httpErrorFromNoContentResponse(resp.RawResponse)
	}

	if resp.RawResponse.Header.Get(httpheader.ContentType) == httpheader.ContentTypeJSON {
		var httpError goutils.RFC9457ErrorWithExtensions

		err := resp.ReadJSON(&httpError)
		if err != nil {
			return httpErrorFromNoContentResponse(resp.RawResponse)
		}

		if httpError.Status == 0 {
			httpError.Status = resp.RawResponse.StatusCode
		}

		if httpError.Title == "" {
			httpError.Title = resp.RawResponse.Status
		}

		httpError.Extensions["headers"] = goutils.ExtractHeaders(resp.RawResponse.Header)

		return httpError
	}

	result := httpErrorFromNoContentResponse(resp.RawResponse)

	rawBody, readErr := resp.ReadBytes()
	if readErr == nil {
		result.Detail = string(rawBody)
	} else {
		result.Extensions["read_error"] = readErr
	}

	return result
}

func httpErrorFromNoContentResponse(resp *http.Response) goutils.RFC9457ErrorWithExtensions {
	return goutils.RFC9457ErrorWithExtensions{
		RFC9457Error: goutils.RFC9457Error{
			Status: resp.StatusCode,
			Title:  resp.Status,
		},
		Extensions: map[string]any{
			"headers": goutils.ExtractHeaders(resp.Header),
		},
	}
}
