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
	"context"
	"io"
	"log/slog"
	"maps"
	"net/http"
	"time"

	"github.com/failsafe-go/failsafe-go/retrypolicy"
	"github.com/google/uuid"
	"github.com/hasura/gotel/otelutils"
	"github.com/relychan/gohttpc/authc/authscheme"
	"go.opentelemetry.io/otel/trace"
)

// Requester abstracts an interface of a request instance.
type Requester interface {
	URL() string
	Method() string
	Header() http.Header
}

// Request represents the details and configuration for an individual HTTP(S)
// request. It encompasses URL, headers, method, body, proxy settings,
// timeouts, and other configurations necessary for customizing the request
// and its execution.
type Request struct {
	// Method specifies the HTTP method (GET, POST, PUT, etc.).
	// For client requests, an empty string means GET.
	method string

	// URL specifies either the URI being requested (for server
	// requests) or the URL to access (for client requests).
	//
	// For server requests, the URL is parsed from the URI
	// supplied on the Request-Line as stored in RequestURI.  For
	// most requests, fields other than Path and RawQuery will be
	// empty. (See RFC 7230, Section 5.3)
	//
	// For client requests, the URL's Host specifies the server to
	// connect to, while the Request's Host field optionally
	// specifies the Host header value to send in the HTTP
	// request.
	url string

	// Body is the request's body.
	//
	// For client requests, a nil body means the request has no
	// body, such as a GET request. The HTTP Client's Transport
	// is responsible for calling the Close method.
	//
	// Body must allow Read to be called concurrently with Close.
	// In particular, calling Close should unblock a Read waiting
	// for input.
	body io.Reader

	// Timeout is the maximum timeout for the request.
	timeout time.Duration

	// RetryPolicy is the retry policy for the request.
	retry         retrypolicy.RetryPolicy[*http.Response]
	authenticator authscheme.HTTPClientAuthenticator
	header        http.Header
	retryAttempts int
	options       *RequestOptions
}

// NewRequest creates a raw request without client options.
func NewRequest(method string, url string, options *RequestOptions) *Request {
	return &Request{
		method:  method,
		url:     url,
		options: options,
	}
}

// Header returns the request header fields to be sent by the client.
//
// HTTP defines that header names are case-insensitive. The
// request parser implements this by using CanonicalHeaderKey,
// making the first character and any characters following a
// hyphen uppercase and the rest lowercase.
//
// For client requests, certain headers such as Content-Length
// and Connection are automatically written when needed and
// values in Header may be ignored. See the documentation
// for the Request.Write method.
func (r *Request) Header() http.Header {
	if r.header == nil {
		r.header = make(http.Header)
	}

	return r.header
}

// Clone creates a new request. The body can be nil if it was already read.
func (r *Request) Clone() *Request {
	newRequest := *r

	if newRequest.header != nil {
		newRequest.header = maps.Clone(r.header)
	}

	return &newRequest
}

// URL returns the request URL.
func (r *Request) URL() string {
	return r.url
}

// SetURL sets the request URL.
func (r *Request) SetURL(value string) *Request {
	r.url = value

	return r
}

// Method returns the request method.
func (r *Request) Method() string {
	return r.method
}

// SetMethod sets the request method.
func (r *Request) SetMethod(method string) *Request {
	r.method = method

	return r
}

// Timeout returns the request timeout.
func (r *Request) Timeout() time.Duration {
	return r.timeout
}

// SetTimeout sets the request timeout.
func (r *Request) SetTimeout(timeout time.Duration) *Request {
	r.timeout = timeout

	return r
}

// Body returns the request body.
func (r *Request) Body() io.Reader {
	return r.body
}

// SetBody sets the request body.
func (r *Request) SetBody(body io.Reader) *Request {
	r.body = body

	return r
}

// Retry returns the retry policy.
func (r *Request) Retry() retrypolicy.RetryPolicy[*http.Response] {
	return r.retry
}

// SetRetry sets the retry policy.
func (r *Request) SetRetry(retry retrypolicy.RetryPolicy[*http.Response]) *Request {
	r.retry = retry

	return r
}

// Authenticator returns the HTTP client authenticator.
func (r *Request) Authenticator() authscheme.HTTPClientAuthenticator {
	return r.authenticator
}

// SetAuthenticator sets the HTTP authenticator.
func (r *Request) SetAuthenticator(authenticator authscheme.HTTPClientAuthenticator) *Request {
	r.authenticator = authenticator

	return r
}

func (r *Request) applyAuth(req *http.Request) error {
	authenticator := r.authenticator

	if authenticator == nil {
		authenticator = r.options.Authenticator
	}

	if authenticator == nil {
		return nil
	}

	return authenticator.Authenticate(req)
}

func (r *Request) getRetryPolicy() retrypolicy.RetryPolicy[*http.Response] {
	if r.retry != nil {
		return r.retry
	}

	return r.options.Retry
}

func (r *Request) getTimeout() time.Duration {
	if r.timeout > 0 {
		return r.timeout
	}

	return r.options.Timeout
}

func (r *Request) getLogger(ctx context.Context) *slog.Logger {
	typeAttr := slog.String("type", "http-client")

	value := ctx.Value(otelutils.LoggerContextKey)
	if value != nil {
		if logger, ok := value.(*slog.Logger); ok {
			return logger.With(typeAttr)
		}
	}

	var requestID string

	spanContext := trace.SpanContextFromContext(ctx)
	if spanContext.HasTraceID() {
		requestID = spanContext.TraceID().String()
	} else {
		requestID = uuid.NewString()
	}

	return slog.Default().With(typeAttr, slog.String("request_id", requestID))
}

// RequestWithClient embeds the [Request] with an [HTTPClient] to make the Execute method shorter.
type RequestWithClient struct {
	*Request

	client HTTPClientGetter
}

// NewRequestWithClient creates a new [RequestWithClient] instance.
func NewRequestWithClient(req *Request, client HTTPClientGetter) *RequestWithClient {
	return &RequestWithClient{
		Request: req,
		client:  client,
	}
}

// Execute handles the HTTP request to the remote server.
func (rwc *RequestWithClient) Execute(ctx context.Context) (*http.Response, error) {
	return rwc.Request.Execute(ctx, rwc.client)
}
