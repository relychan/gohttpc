package loadbalancer

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/failsafe-go/failsafe-go/circuitbreaker"
	"github.com/relychan/gohttpc"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.30.0"
)

var (
	// ErrInvalidHealthCheckMethod occurs when the HTTP method of the health check config is invalid.
	ErrInvalidHealthCheckMethod = errors.New(
		"invalid health check method. Expects one of GET or POST",
	)
	// ErrInvalidHealthCheckSuccessStatus occurs when the HTTP success status of the health check config is invalid.
	ErrInvalidHealthCheckSuccessStatus = errors.New(
		"invalid status of HTTP health check. Expects one of 200, 201 or 204",
	)
	// ErrInvalidHealthCheckFailureThreshold occurs when the failure threshold of the health check config is invalid.
	ErrInvalidHealthCheckFailureThreshold = errors.New(
		"failure threshold of HTTP health check must be positive",
	)
)

// HTTPHealthCheckConfig holds configurations for health checking the server and recovery.
type HTTPHealthCheckConfig struct {
	// Health check path, e.g, /healthz.
	Path string `json:"path" yaml:"path"`
	// Health check method. Default to GET
	Method string `json:"method,omitempty" yaml:"method,omitempty" jsonschema:"default=GET,enum=GET,enum=POST"`
	// Request body is used if the method is POST.
	Body any `json:"body,omitempty" yaml:"body,omitempty"`
	// Request headers to be sent to health check requests.
	Headers map[string]string `json:"headers,omitempty" yaml:"headers,omitempty"`
	// Health check interval in seconds. Disabled if the interval is negative or equals 0. Default to 60 seconds
	Interval *int `json:"interval,omitempty" yaml:"interval,omitempty" jsonschema:"default=60,min=0"`
	// Timeout in seconds. Disabled if the timeout is negative or equals 0. Default to 5 seconds
	Timeout *int `json:"timeout,omitempty" yaml:"timeout,omitempty" jsonschema:"default=5,min=0"`
	// SuccessStatus is expected successful HTTP status. Default to HTTP 200 OK.
	SuccessStatus *int `json:"successStatus,omitempty" yaml:"successStatus,omitempty" jsonschema:"default=200,enum=200,enum=201,enum=204"`
	// SuccessThreshold is the minimum consecutive successes for the probe to be considered successful after having failed. Defaults to 1. Minimum value is 1.
	SuccessThreshold *int `json:"successThreshold,omitempty" yaml:"successThreshold,omitempty" jsonschema:"default=1,min=1"`
	// Failure threshold. After a probe fails threshold times in a row, the HTTP client considers that the overall check has failed. Default to 5. Minimum value is 1
	FailureThreshold *int `json:"failureThreshold,omitempty" yaml:"failureThreshold,omitempty" jsonschema:"default=3,min=1"`
}

// ToPolicyBuilder validates the health check config and create the policy builder.
func (hc HTTPHealthCheckConfig) ToPolicyBuilder() (*httpHealthCheckPolicyBuilder, error) {
	builder := NewHTTPHealthCheckPolicyBuilder()

	if hc.SuccessStatus != nil {
		builder.successStatus = *hc.SuccessStatus

		if builder.successStatus != http.StatusOK && builder.successStatus != http.StatusCreated &&
			builder.successStatus != http.StatusNoContent {
			return nil, ErrInvalidHealthCheckSuccessStatus
		}
	}

	if hc.SuccessThreshold != nil && *hc.SuccessThreshold > 1 {
		builder.successThreshold = uint(*hc.SuccessThreshold)
	}

	if hc.FailureThreshold != nil {
		if *hc.FailureThreshold < 1 {
			return nil, ErrInvalidHealthCheckFailureThreshold
		}

		builder.failureThreshold = uint(*hc.FailureThreshold)
	}

	// If no health check interval is set, the circuit breaker still runs with runtime HTTP requests.
	if hc.Interval != nil && *hc.Interval > 0 {
		builder.interval = time.Duration(*hc.Interval) * time.Second
	}

	builder.headers = hc.Headers

	if hc.Path != "" {
		builder.path = hc.Path
	}

	if hc.Method != "" {
		if hc.Method != http.MethodGet && hc.Method != http.MethodPost {
			return nil, ErrInvalidHealthCheckMethod
		}

		builder.method = hc.Method
	}

	if hc.Body != nil {
		buffer := new(bytes.Buffer)

		enc := json.NewEncoder(buffer)
		enc.SetEscapeHTML(false)

		err := enc.Encode(hc.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to encode health check request body: %w", err)
		}

		builder.body = buffer.Bytes()
	}

	if hc.Timeout != nil && *hc.Timeout > 0 {
		builder.timeout = time.Duration(*hc.Timeout) * time.Second
	}

	return builder, nil
}

// ToPolicy validates the health check config and create the policy.
func (hc HTTPHealthCheckConfig) ToPolicy(endpoint *url.URL) (*HTTPHealthCheckPolicy, error) {
	builder, err := hc.ToPolicyBuilder()
	if err != nil {
		return nil, err
	}

	return builder.Build(endpoint), nil
}

// HTTPHealthCheckPolicy represents an HTTP health check policy state.
type HTTPHealthCheckPolicy struct {
	circuitbreaker.CircuitBreaker[int]

	path    string
	method  string
	headers map[string]string
	body    []byte
	timeout time.Duration
}

// Path returns the health check path.
func (hcp *HTTPHealthCheckPolicy) Path() string {
	return hcp.path
}

// SetPath sets the health check path.
func (hcp *HTTPHealthCheckPolicy) SetPath(value string) *HTTPHealthCheckPolicy {
	hcp.path = value

	return hcp
}

// Method returns the health check method.
func (hcp *HTTPHealthCheckPolicy) Method() string {
	return hcp.method
}

// SetMethod sets the health check method.
func (hcp *HTTPHealthCheckPolicy) SetMethod(value string) *HTTPHealthCheckPolicy {
	hcp.method = value

	return hcp
}

// Body returns the health check body.
func (hcp *HTTPHealthCheckPolicy) Body() []byte {
	return hcp.body
}

// SetBody sets the health check body.
func (hcp *HTTPHealthCheckPolicy) SetBody(value []byte) *HTTPHealthCheckPolicy {
	hcp.body = value

	return hcp
}

// Headers returns the health check headers.
func (hcp *HTTPHealthCheckPolicy) Headers() map[string]string {
	return hcp.headers
}

// SetHeaders sets the health check headers.
func (hcp *HTTPHealthCheckPolicy) SetHeaders(value map[string]string) *HTTPHealthCheckPolicy {
	hcp.headers = value

	return hcp
}

// Timeout returns the health check timeout duration.
func (hcp *HTTPHealthCheckPolicy) Timeout() time.Duration {
	return hcp.timeout
}

// SetTimeout sets the health check timeout duration.
func (hcp *HTTPHealthCheckPolicy) SetTimeout(value time.Duration) *HTTPHealthCheckPolicy {
	hcp.timeout = value

	return hcp
}

type httpHealthCheckPolicyBuilder struct {
	*HTTPHealthCheckPolicy

	successStatus    int
	successThreshold uint
	failureThreshold uint
	interval         time.Duration
}

// NewHTTPHealthCheckPolicyBuilder creates an HTTP health check policy builder.
func NewHTTPHealthCheckPolicyBuilder() *httpHealthCheckPolicyBuilder {
	return &httpHealthCheckPolicyBuilder{
		HTTPHealthCheckPolicy: &HTTPHealthCheckPolicy{
			method:  http.MethodGet,
			path:    "/",
			timeout: 5 * time.Second,
		},
		successStatus:    http.StatusOK,
		successThreshold: 1,
		failureThreshold: 3,
		interval:         time.Minute,
	}
}

// WithInterval sets the health check interval.
func (hb *httpHealthCheckPolicyBuilder) WithInterval(
	value time.Duration,
) *httpHealthCheckPolicyBuilder {
	hb.interval = value

	return hb
}

// WithSuccessStatus sets the expected success status of the health check.
func (hb *httpHealthCheckPolicyBuilder) WithSuccessStatus(
	status int,
) *httpHealthCheckPolicyBuilder {
	hb.successStatus = status

	return hb
}

// WithSuccessThreshold sets the success threshold of the health check.
func (hb *httpHealthCheckPolicyBuilder) WithSuccessThreshold(
	value uint,
) *httpHealthCheckPolicyBuilder {
	hb.successThreshold = value

	return hb
}

// WithFailureThreshold sets the failure threshold of the health check.
func (hb *httpHealthCheckPolicyBuilder) WithFailureThreshold(
	value uint,
) *httpHealthCheckPolicyBuilder {
	hb.failureThreshold = value

	return hb
}

// Build builds the [HTTPHealthCheckPolicy].
func (hb *httpHealthCheckPolicyBuilder) Build(endpoint *url.URL) *HTTPHealthCheckPolicy {
	metrics := gohttpc.GetHTTPClientMetrics()
	urlScheme := "http"

	if endpoint.Scheme != "" {
		urlScheme = endpoint.Scheme
	}

	metricsAttrs := metric.WithAttributeSet(attribute.NewSet(
		semconv.ServerAddress(endpoint.Host),
		semconv.URLScheme(urlScheme),
	))

	builder := circuitbreaker.NewBuilder[int]().
		HandleIf(func(i int, err error) bool {
			return err != nil || i != hb.successStatus
		}).WithSuccessThreshold(hb.successThreshold).
		WithFailureThreshold(hb.failureThreshold).
		OnStateChanged(func(sce circuitbreaker.StateChangedEvent) {
			metrics.ServerState.Record(context.TODO(), int64(sce.NewState), metricsAttrs)
		})

	if hb.interval > 0 {
		builder = builder.WithDelay(hb.interval - time.Millisecond)
	}

	policy := *hb.HTTPHealthCheckPolicy
	policy.CircuitBreaker = builder.Build()

	// change the initial state to half-open,
	// so the first request will trigger the OnStateChanged event to push metrics.
	policy.HalfOpen()

	return &policy
}
