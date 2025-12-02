package gohttpc

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/failsafe-go/failsafe-go/retrypolicy"
	"github.com/relychan/gohttpc/authc/authscheme"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// RequestOptionsGetter abstracts an interface to get the [RequestOptions].
type RequestOptionsGetter interface {
	GetRequestOptions() *RequestOptions
}

// RequestOptions defines options for the request.
type RequestOptions struct {
	Logger                      *slog.Logger
	Tracer                      trace.Tracer
	TraceHighCardinalityPath    bool
	MetricHighCardinalityPath   bool
	CustomAttributesFunc        CustomAttributesFunc
	Retry                       retrypolicy.RetryPolicy[*http.Response]
	Timeout                     time.Duration
	Authenticator               authscheme.HTTPClientAuthenticator
	ClientTraceEnabled          bool
	UserAgent                   string
	AllowedTraceRequestHeaders  []string
	AllowedTraceResponseHeaders []string
}

var _ RequestOptionsGetter = (*RequestOptions)(nil)

// GetRequestOptions gets the inner [RequestOptions].
func (ro *RequestOptions) GetRequestOptions() *RequestOptions {
	return ro
}

// IsTraceRequestHeadersEnabled checks if the trace request headers are enabled.
func (ro *RequestOptions) IsTraceRequestHeadersEnabled() bool {
	return ro.AllowedTraceRequestHeaders == nil || len(ro.AllowedTraceRequestHeaders) > 0
}

// IsTraceResponseHeadersEnabled checks if the trace request headers are enabled.
func (ro *RequestOptions) IsTraceResponseHeadersEnabled() bool {
	return ro.AllowedTraceResponseHeaders == nil || len(ro.AllowedTraceResponseHeaders) > 0
}

// ClientOptions defines options for the client.
type ClientOptions struct {
	RequestOptions

	HTTPClient *http.Client
}

// NewClientOptions create a new [ClientOptions] instance.
func NewClientOptions(options ...ClientOption) *ClientOptions {
	opts := ClientOptions{
		RequestOptions: RequestOptions{
			Logger:             slog.Default(),
			Tracer:             clientTracer,
			UserAgent:          "gohttpc/" + getBuildVersion(),
			ClientTraceEnabled: os.Getenv("HTTP_CLIENT_TRACE_ENABLED") == "true",
		},
	}

	for _, opt := range options {
		opt(&opts)
	}

	return &opts
}

var _ RequestOptionsGetter = (*ClientOptions)(nil)

// GetRequestOptions gets the inner [RequestOptions].
func (co *ClientOptions) GetRequestOptions() *RequestOptions {
	return &co.RequestOptions
}

// Clone creates a new ClientOptions instance with copied values.
func (co *ClientOptions) Clone(options ...ClientOption) *ClientOptions {
	newOptions := *co

	for _, opt := range options {
		opt(&newOptions)
	}

	return &newOptions
}

// CustomAttributesFunc abstracts a function to add custom attributes to spans and metrics.
type CustomAttributesFunc func(*Request) []attribute.KeyValue

// ClientOption abstracts a function to modify client options.
type ClientOption func(*ClientOptions)

// RequestOption abstracts a function to modify request options.
type RequestOption func(*RequestOptions)

// WithHTTPClient create an option to set the HTTP client.
func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(co *ClientOptions) {
		co.HTTPClient = httpClient
	}
}

// WithLogger create an option to set the logger.
func WithLogger(logger *slog.Logger) ClientOption {
	return func(co *ClientOptions) {
		if logger != nil {
			co.Logger = logger
		}
	}
}

// WithTracer create an option to set the tracer.
func WithTracer(tracer trace.Tracer) ClientOption {
	return func(co *ClientOptions) {
		co.Tracer = tracer
	}
}

// WithTraceHighCardinalityPath enables high cardinality path on traces.
func WithTraceHighCardinalityPath(enabled bool) ClientOption {
	return func(co *ClientOptions) {
		co.TraceHighCardinalityPath = enabled
	}
}

// WithMetricHighCardinalityPath enables high cardinality path on metrics.
func WithMetricHighCardinalityPath(enabled bool) ClientOption {
	return func(co *ClientOptions) {
		co.MetricHighCardinalityPath = enabled
	}
}

// WithCustomAttributesFunc sets the function to add custom attributes to spans and metrics.
func WithCustomAttributesFunc(fn CustomAttributesFunc) ClientOption {
	return func(co *ClientOptions) {
		co.CustomAttributesFunc = fn
	}
}

// WithRetry creates an option to set the default retry policy.
func WithRetry(retry retrypolicy.RetryPolicy[*http.Response]) ClientOption {
	return func(co *ClientOptions) {
		co.Retry = retry
	}
}

// WithTimeout creates an option to set the default timeout.
func WithTimeout(timeout time.Duration) ClientOption {
	return func(co *ClientOptions) {
		co.Timeout = timeout
	}
}

// WithAuthenticator creates an option to set the default authenticator.
func WithAuthenticator(authenticator authscheme.HTTPClientAuthenticator) ClientOption {
	return func(co *ClientOptions) {
		co.Authenticator = authenticator
	}
}

// EnableClientTrace creates an option to enable the HTTP client trace.
func EnableClientTrace(enabled bool) ClientOption {
	return func(co *ClientOptions) {
		co.ClientTraceEnabled = enabled
	}
}

// AllowTraceRequestHeaders creates an option to set allowed headers for tracing.
func AllowTraceRequestHeaders(keys []string) ClientOption {
	return func(co *ClientOptions) {
		co.AllowedTraceRequestHeaders = keys
	}
}

// AllowTraceResponseHeaders creates an option to set allowed headers for tracing.
func AllowTraceResponseHeaders(keys []string) ClientOption {
	return func(co *ClientOptions) {
		co.AllowedTraceResponseHeaders = keys
	}
}

// WithUserAgent creates an option to set the user agent.
func WithUserAgent(userAgent string) ClientOption {
	return func(co *ClientOptions) {
		co.UserAgent = userAgent
	}
}
