package gohttpc

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/relychan/gocompress"
	"github.com/relychan/gohttpc/authc/authscheme"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// Client represents an HTTP client wrapper with extended functionality.
type Client struct {
	options     *ClientOptions
	compressors *gocompress.Compressors
}

// NewClient creates a new HTTP client wrapper.
func NewClient(options ...Option) *Client {
	return NewClientWithOptions(NewClientOptions(options...))
}

// NewClientWithOptions creates a new HTTP client wrapper with client options.
func NewClientWithOptions(options *ClientOptions) *Client {
	if options.HTTPClient == nil {
		options.HTTPClient = http.DefaultClient
	}

	return &Client{
		options:     options,
		compressors: gocompress.DefaultCompressor,
	}
}

// NewRequest creates a new request with default parameters.
func (c *Client) NewRequest(method string, requestURI string) *Request {
	return &Request{
		Method:  method,
		URL:     requestURI,
		Retry:   c.options.Retry,
		Timeout: c.options.Timeout,
		client:  c,
	}
}

// Clone a new client with properties copied.
func (c *Client) Clone() *Client {
	options := *c.options

	return &Client{
		options:     &options,
		compressors: c.compressors,
	}
}

// Close terminates internal processes.
func (c *Client) Close() error {
	if c.options.HTTPClient != nil {
		c.options.HTTPClient.CloseIdleConnections()
	}

	return nil
}

// ClientOptions defines options for the client.
type ClientOptions struct {
	HTTPClient                  *http.Client
	Logger                      *slog.Logger
	Tracer                      trace.Tracer
	Metrics                     *HTTPClientMetrics
	TraceHighCardinalityPath    bool
	MetricHighCardinalityPath   bool
	CustomAttributesFunc        CustomAttributesFunc
	Retry                       *RetryPolicy
	Timeout                     time.Duration
	Authenticator               authscheme.HTTPClientAuthenticator
	ClientTraceEnabled          bool
	UserAgent                   string
	CreateRequest               CreateRequestFunc
	AllowedTraceRequestHeaders  []string
	AllowedTraceResponseHeaders []string
}

// NewClientOptions create a new ClientOptions instance.
func NewClientOptions(options ...Option) *ClientOptions {
	opts := ClientOptions{
		Logger:             slog.Default(),
		Tracer:             tracer,
		Metrics:            &noopHTTPClientMetrics,
		UserAgent:          "gohttpc/" + getBuildVersion(),
		CreateRequest:      createRequest,
		ClientTraceEnabled: os.Getenv("HTTP_CLIENT_TRACE_ENABLED") == "true",
	}

	for _, opt := range options {
		opt(&opts)
	}

	return &opts
}

// IsTraceRequestHeadersEnabled checks if the trace request headers are enabled.
func (co ClientOptions) IsTraceRequestHeadersEnabled() bool {
	return co.AllowedTraceRequestHeaders == nil || len(co.AllowedTraceRequestHeaders) > 0
}

// IsTraceResponseHeadersEnabled checks if the trace request headers are enabled.
func (co ClientOptions) IsTraceResponseHeadersEnabled() bool {
	return co.AllowedTraceResponseHeaders == nil || len(co.AllowedTraceResponseHeaders) > 0
}

// CustomAttributesFunc abstracts a function to add custom attributes to spans and metrics.
type CustomAttributesFunc func(*http.Request) []attribute.KeyValue

// Option abstracts a function to modify client options.
type Option func(*ClientOptions)

// WithHTTPClient create an option to set the HTTP client.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(co *ClientOptions) {
		co.HTTPClient = httpClient
	}
}

// WithLogger create an option to set the logger.
func WithLogger(logger *slog.Logger) Option {
	return func(co *ClientOptions) {
		if logger != nil {
			co.Logger = logger
		}
	}
}

// WithTracer create an option to set the tracer.
func WithTracer(tracer trace.Tracer) Option {
	return func(co *ClientOptions) {
		co.Tracer = tracer
	}
}

// WithMetrics creates an option to set metrics for metrics.
func WithMetrics(metrics *HTTPClientMetrics) Option {
	return func(co *ClientOptions) {
		co.Metrics = metrics
	}
}

// WithTraceHighCardinalityPath enables high cardinality path on traces.
func WithTraceHighCardinalityPath(enabled bool) Option {
	return func(co *ClientOptions) {
		co.TraceHighCardinalityPath = enabled
	}
}

// WithMetricHighCardinalityPath enables high cardinality path on metrics.
func WithMetricHighCardinalityPath(enabled bool) Option {
	return func(co *ClientOptions) {
		co.MetricHighCardinalityPath = enabled
	}
}

// WithCustomAttributesFunc sets the function to add custom attributes to spans and metrics.
func WithCustomAttributesFunc(fn CustomAttributesFunc) Option {
	return func(co *ClientOptions) {
		co.CustomAttributesFunc = fn
	}
}

// WithRetry creates an option to set the default retry policy.
func WithRetry(retry *RetryPolicy) Option {
	return func(co *ClientOptions) {
		co.Retry = retry
	}
}

// WithTimeout creates an option to set the default timeout.
func WithTimeout(timeout time.Duration) Option {
	return func(co *ClientOptions) {
		co.Timeout = timeout
	}
}

// WithAuthenticator creates an option to set the default authenticator.
func WithAuthenticator(authenticator authscheme.HTTPClientAuthenticator) Option {
	return func(co *ClientOptions) {
		co.Authenticator = authenticator
	}
}

// EnableClientTrace creates an option to enable the HTTP client trace.
func EnableClientTrace(enabled bool) Option {
	return func(co *ClientOptions) {
		co.ClientTraceEnabled = enabled
	}
}

// AllowTraceRequestHeaders creates an option to set allowed headers for tracing.
func AllowTraceRequestHeaders(keys []string) Option {
	return func(co *ClientOptions) {
		co.AllowedTraceRequestHeaders = keys
	}
}

// AllowTraceResponseHeaders creates an option to set allowed headers for tracing.
func AllowTraceResponseHeaders(keys []string) Option {
	return func(co *ClientOptions) {
		co.AllowedTraceResponseHeaders = keys
	}
}

// WithUserAgent creates an option to set the user agent.
func WithUserAgent(userAgent string) Option {
	return func(co *ClientOptions) {
		co.UserAgent = userAgent
	}
}

// WithCreateRequestFunc creates an option to set the request constructor function.
func WithCreateRequestFunc(fn CreateRequestFunc) Option {
	return func(co *ClientOptions) {
		co.CreateRequest = fn
	}
}

// CreateRequestFunc defines a function interface to create a *http.Request.
type CreateRequestFunc func(ctx context.Context, r *Request, body io.Reader) (*http.Request, error)

func createRequest(ctx context.Context, r *Request, body io.Reader) (*http.Request, error) {
	return http.NewRequestWithContext(ctx, r.Method, r.URL, body)
}
