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
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/relychan/goutils"
	"github.com/relychan/goutils/httpheader"
)

// CloseResponse gracefully closes the HTTP response and tries to drain the body if it exists.
// It makes a best effort to reuse the HTTP connection.
func CloseResponse(resp *http.Response) {
	if resp == nil || resp.Body == nil || resp.Body == http.NoBody {
		return
	}

	contentLength := resp.ContentLength
	if contentLength == -1 {
		rawContentLength := resp.Header[httpheader.ContentLength]
		if len(rawContentLength) > 0 {
			intContentLength, err := strconv.ParseInt(rawContentLength[0], 10, 64)
			if err == nil {
				contentLength = intContentLength
			}
		}
	}

	if contentLength <= maxPostCloseReadBytes {
		maybeDrainBody(resp.Body)
	}

	goutils.CatchWarnErrorFunc(resp.Body.Close)
}

// responseBodyWithCancel wraps the original body of the HTTP response with cancel if timeout is configured.
type responseBodyWithCancel struct {
	io.ReadCloser

	cancel func()
}

// Close closes the body reader and cancels the context.
func (rb *responseBodyWithCancel) Close() error {
	err := rb.ReadCloser.Close()

	rb.cancel()

	return err
}

// maxPostCloseReadBytes is the max number of bytes that a client is willing to
// read when draining the response body of any unread bytes after it has been
// closed. This number is chosen for consistency with maxPostHandlerReadBytes.
const maxPostCloseReadBytes = 256 << 10

// maxPostCloseReadTime defines the maximum amount of time that a client is
// willing to spend on draining a response body of any unread bytes after it
// has been closed.
const maxPostCloseReadTime = 50 * time.Millisecond

// Try to drain the response body to reuse the HTTP connection.
// TODO: deprecate this function if this PR was merged https://go-review.googlesource.com/c/go/+/737720
//
//nolint:godox
func maybeDrainBody(body io.Reader) bool {
	drainedCh := make(chan bool, 1)

	go func() {
		_, err := io.CopyN(io.Discard, body, maxPostCloseReadBytes+1)
		drainedCh <- err == io.EOF //nolint:errorlint
	}()

	select {
	case drained := <-drainedCh:
		return drained
	case <-time.After(maxPostCloseReadTime):
		return false
	}
}
