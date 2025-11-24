package gohttpc

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/relychan/goutils"
)

// Response is the wrapper of the http.Response.
type Response struct {
	RawResponse *http.Response

	body       io.ReadCloser
	isBodyRead bool
	closed     bool
}

// StatusCode returns the HTTP status code of the response.
func (resp *Response) StatusCode() int {
	return resp.RawResponse.StatusCode
}

// Header returns the response header map.
func (resp *Response) Header() http.Header {
	return resp.RawResponse.Header
}

// IsBodyRead returns the status whether the response body was read.
func (resp *Response) IsBodyRead() bool {
	return resp.isBodyRead
}

// Body returns the response body.
func (resp *Response) Body() io.ReadCloser {
	return resp.body
}

// ReadBytes reads the entire body to bytes.
func (resp *Response) ReadBytes() ([]byte, error) {
	if resp.closed {
		return nil, http.ErrBodyReadAfterClose
	}

	if resp.isBodyRead {
		return nil, ErrResponseBodyAlreadyRead
	}

	if resp.body == nil {
		return nil, ErrResponseBodyNoContent
	}

	defer goutils.CatchWarnErrorFunc(resp.body.Close)

	result, err := io.ReadAll(resp.body)
	resp.isBodyRead = true

	return result, err
}

// ReadJSON reads the body and decodes to JSON.
func (resp *Response) ReadJSON(target any) error {
	if resp.closed {
		return http.ErrBodyReadAfterClose
	}

	if resp.isBodyRead {
		return ErrResponseBodyAlreadyRead
	}

	if resp.body == nil {
		return ErrResponseBodyNoContent
	}

	defer goutils.CatchWarnErrorFunc(resp.body.Close)

	err := json.NewDecoder(resp.body).Decode(&target)
	resp.isBodyRead = true

	return err
}

// Close the response body.
func (resp *Response) Close() error {
	if resp.closed {
		return nil
	}

	var err error

	if !resp.isBodyRead && resp.body != nil {
		err = resp.body.Close()
	}

	resp.closed = true

	return err
}
