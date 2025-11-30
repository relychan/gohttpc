package gohttpc

import (
	"context"
	"io"
	"net/http"
)

// HTTPClientGetter abstracts an interface to get an HTTP client.
type HTTPClientGetter interface {
	// HTTPClient returns the current or inner HTTP client for load balancing.
	HTTPClient() (HTTPClient, error)
}

// HTTPClient abstracts an HTTP client with methods.
type HTTPClient interface {
	// NewRequest returns a new http.Request given a method, URL, and optional body.
	NewRequest(
		ctx context.Context,
		method string,
		url string,
		body io.Reader,
	) (*http.Request, error)
	// Do sends an HTTP request and returns an HTTP response, following policy
	// (such as redirects, cookies, auth) as configured on the client.
	Do(req *http.Request) (*http.Response, error)
}

// Client represents an HTTP client wrapper with extended functionality.
type Client struct {
	options *ClientOptions
}

// NewClient creates a new HTTP client wrapper.
func NewClient(options ...ClientOption) *Client {
	return NewClientWithOptions(NewClientOptions(options...))
}

// NewClientWithOptions creates a new HTTP client wrapper with client options.
func NewClientWithOptions(options *ClientOptions) *Client {
	if options.HTTPClient == nil {
		options.HTTPClient = &http.Client{
			Transport: TransportFromConfig(nil, options),
		}
	}

	return &Client{
		options: options,
	}
}

// R is the shortcut to create a Request given a method, URL with default request options.
func (c *Client) R(method string, url string) *RequestWithClient {
	return &RequestWithClient{
		Request: NewRequest(method, url, &c.options.RequestOptions),
		client:  c,
	}
}

// HTTPClient returns the current or inner HTTP client for load balancing.
func (c *Client) HTTPClient() (HTTPClient, error) {
	return c, nil
}

// NewRequest returns a new http.Request given a method, URL, and optional body.
func (c *Client) NewRequest(
	ctx context.Context,
	method string,
	url string,
	body io.Reader,
) (*http.Request, error) {
	return http.NewRequestWithContext(ctx, method, url, body)
}

// Do sends an HTTP request and returns an HTTP response, following policy
// (such as redirects, cookies, auth) as configured on the client.
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	return c.options.HTTPClient.Do(req)
}

// Clone creates a new client with properties copied.
func (c *Client) Clone(options ...ClientOption) *Client {
	return &Client{
		options: c.options.Clone(options...),
	}
}

// Close terminates internal processes.
func (c *Client) Close() error {
	if c.options.HTTPClient != nil {
		c.options.HTTPClient.CloseIdleConnections()
	}

	return nil
}
