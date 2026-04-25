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
)

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
