package loadbalancer

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/failsafe-go/failsafe-go/circuitbreaker"
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
	// SuccessStatus is expected successful HTTP status. Default to HTTP 200 OK.
	SuccessStatus *int `json:"successStatus,omitempty" yaml:"successStatus,omitempty" jsonschema:"default=200,enum=200,enum=201,enum=204"`
	// SuccessThreshold is the minimum consecutive successes for the probe to be considered successful after having failed. Defaults to 1. Minimum value is 1.
	SuccessThreshold *int `json:"successThreshold,omitempty" yaml:"successThreshold,omitempty" jsonschema:"default=1,min=1"`
	// Failure threshold. After a probe fails threshold times in a row, the HTTP client considers that the overall check has failed. Default to 5. Minimum value is 1
	FailureThreshold *int `json:"failureThreshold,omitempty" yaml:"failureThreshold,omitempty" jsonschema:"default=3,min=1"`
}

// ToPolicy validates the health check config and create the policy.
func (hc HTTPHealthCheckConfig) ToPolicy() (*HTTPHealthCheckPolicy, error) { //nolint:funlen
	successStatus := http.StatusOK

	if hc.SuccessStatus != nil {
		successStatus = *hc.SuccessStatus

		if successStatus != http.StatusOK && successStatus != http.StatusCreated &&
			successStatus != http.StatusNoContent {
			return nil, ErrInvalidHealthCheckSuccessStatus
		}
	}

	builder := circuitbreaker.NewBuilder[int]().
		HandleIf(func(i int, err error) bool {
			return err != nil || i != successStatus
		})

	if hc.SuccessThreshold != nil && *hc.SuccessThreshold > 1 {
		builder = builder.WithSuccessThreshold(uint(*hc.SuccessThreshold))
	}

	if hc.FailureThreshold != nil {
		if *hc.FailureThreshold < 1 {
			return nil, ErrInvalidHealthCheckFailureThreshold
		}

		builder = builder.WithFailureThreshold(uint(*hc.FailureThreshold))
	}

	// If the health check internal, the circuit breaking still runs with runtime HTTP requests.
	if hc.Interval != nil && *hc.Interval <= 0 {
		return &HTTPHealthCheckPolicy{
			interval:       -1,
			CircuitBreaker: builder.Build(),
		}, nil
	}

	policy := NewHTTPHealthCheckPolicy(http.MethodGet, "/", time.Minute, nil)
	policy.headers = hc.Headers

	if hc.Path != "" {
		policy.path = hc.Path
	}

	if hc.Method != "" {
		if hc.Method != http.MethodGet && hc.Method != http.MethodPost {
			return nil, ErrInvalidHealthCheckMethod
		}

		policy.method = hc.Method
	}

	if hc.Body != nil {
		buffer := new(bytes.Buffer)

		enc := json.NewEncoder(buffer)
		enc.SetEscapeHTML(false)

		err := enc.Encode(hc.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to encode health check request body: %w", err)
		}

		policy.body = buffer.Bytes()
	}

	if hc.Interval != nil {
		policy.interval = time.Duration(*hc.Interval) * time.Second
		builder = builder.WithDelay(policy.interval - time.Millisecond)
	}

	policy.CircuitBreaker = builder.Build()

	return policy, nil
}

// HTTPHealthCheckPolicy represents an HTTP health check policy state.
type HTTPHealthCheckPolicy struct {
	circuitbreaker.CircuitBreaker[int]

	path     string
	method   string
	headers  map[string]string
	body     []byte
	interval time.Duration
}

// NewHTTPHealthCheckPolicy creates a new [HTTPHealthCheckPolicy] instance.
func NewHTTPHealthCheckPolicy(
	method string,
	healthPath string,
	interval time.Duration,
	circuitBreaker circuitbreaker.CircuitBreaker[int],
) *HTTPHealthCheckPolicy {
	if circuitBreaker == nil {
		circuitBreaker = circuitbreaker.NewBuilder[int]().
			WithDelay(interval).Build()
	}

	return &HTTPHealthCheckPolicy{
		CircuitBreaker: circuitBreaker,
		path:           healthPath,
		method:         method,
		interval:       interval,
	}
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

// Interval returns the health check interval.
func (hcp *HTTPHealthCheckPolicy) Interval() time.Duration {
	return hcp.interval
}

// SetInterval sets the health check interval.
func (hcp *HTTPHealthCheckPolicy) SetInterval(value time.Duration) *HTTPHealthCheckPolicy {
	hcp.interval = value

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
