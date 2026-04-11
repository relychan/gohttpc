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

package gohttpc_test

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/relychan/gohttpc"
)

// closeSpy wraps a ReadCloser and records whether Close was called.
type closeSpy struct {
	io.ReadCloser
	closed bool
}

func (s *closeSpy) Close() error {
	s.closed = true
	return s.ReadCloser.Close()
}

func TestCloseResponse_NilResponse(t *testing.T) {
	// Must not panic.
	gohttpc.CloseResponse(nil)
}

func TestCloseResponse_NilBody(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       nil,
		Header:     make(http.Header),
	}
	// Must not panic.
	gohttpc.CloseResponse(resp)
}

func TestCloseResponse_NoBody(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       http.NoBody,
		Header:     make(http.Header),
	}
	// Must not panic.
	gohttpc.CloseResponse(resp)
}

func TestCloseResponse_SmallBodyIsDrained(t *testing.T) {
	body := "small response body"
	spy := &closeSpy{ReadCloser: io.NopCloser(strings.NewReader(body))}

	resp := &http.Response{
		StatusCode:    http.StatusOK,
		Body:          spy,
		ContentLength: int64(len(body)),
		Header:        make(http.Header),
	}

	gohttpc.CloseResponse(resp)

	if !spy.closed {
		t.Error("expected body to be closed")
	}
}

func TestCloseResponse_UnknownContentLength_HeaderFallback(t *testing.T) {
	body := "some content"
	spy := &closeSpy{ReadCloser: io.NopCloser(strings.NewReader(body))}

	resp := &http.Response{
		StatusCode:    http.StatusOK,
		Body:          spy,
		ContentLength: -1, // unknown from Transport
		Header: http.Header{
			"Content-Length": []string{fmt.Sprintf("%d", len(body))},
		},
	}

	gohttpc.CloseResponse(resp)

	if !spy.closed {
		t.Error("expected body to be closed after draining via Content-Length header fallback")
	}
}

func TestCloseResponse_UnknownContentLength_NoHeader(t *testing.T) {
	// ContentLength -1 with no Content-Length header — treated as unknown/small, still closes.
	body := "body without content-length"
	spy := &closeSpy{ReadCloser: io.NopCloser(strings.NewReader(body))}

	resp := &http.Response{
		StatusCode:    http.StatusOK,
		Body:          spy,
		ContentLength: -1,
		Header:        make(http.Header),
	}

	gohttpc.CloseResponse(resp)

	if !spy.closed {
		t.Error("expected body to be closed")
	}
}

func TestCloseResponse_LargeBodySkipsDrain(t *testing.T) {
	// Body larger than maxPostCloseReadBytes (256 KiB) — drain is skipped, Close still called.
	const maxBytes = 256 << 10
	largeBody := bytes.Repeat([]byte("x"), maxBytes+1)
	spy := &closeSpy{ReadCloser: io.NopCloser(bytes.NewReader(largeBody))}

	resp := &http.Response{
		StatusCode:    http.StatusOK,
		Body:          spy,
		ContentLength: int64(len(largeBody)),
		Header:        make(http.Header),
	}

	gohttpc.CloseResponse(resp)

	if !spy.closed {
		t.Error("expected body to be closed even when too large to drain")
	}
}

func TestCloseResponse_InvalidContentLengthHeader_FallsBackToNoDrain(t *testing.T) {
	// Malformed Content-Length header — parse error is ignored, contentLength stays -1.
	// ContentLength -1 with no parseable header: treated as 0-ish, drain proceeds.
	body := "data"
	spy := &closeSpy{ReadCloser: io.NopCloser(strings.NewReader(body))}

	resp := &http.Response{
		StatusCode:    http.StatusOK,
		Body:          spy,
		ContentLength: -1,
		Header: http.Header{
			"Content-Length": []string{"not-a-number"},
		},
	}

	gohttpc.CloseResponse(resp)

	// Body should still be closed (drain attempted for unknown/small size).
	if !spy.closed {
		t.Error("expected body to be closed even with invalid Content-Length header")
	}
}
