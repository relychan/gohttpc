package loadbalancer

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"path"
	"time"

	"github.com/failsafe-go/failsafe-go/circuitbreaker"
	"github.com/relychan/gohttpc"
	"github.com/relychan/gohttpc/authc/authscheme"
	"github.com/relychan/goutils"
)

// Server represents the host information and its weight to load balance the requests.
type Server struct {
	// An optional unique string to refer to the host designated by the URL.
	name string
	// A URL to the target host.
	url string
	// Defines custom headers to be injected to incoming requests.
	headers map[string]string
	// Defines the weight of the server endpoint for load balancing.
	weight int
	// The HTTP client is used for this server.
	httpClient *http.Client
	// The custom authenticator for the current server.
	authenticator authscheme.HTTPClientAuthenticator
	// The health check policy.
	healthCheckPolicy *HTTPHealthCheckPolicy
	// The current weight of the server.
	currentWeight int
}

var _ gohttpc.HTTPClient = (*Server)(nil)

// NewServer creates a Server with a client base URL.
func NewServer(client *http.Client, baseURL string, weight int) *Server {
	return &Server{
		url:        baseURL,
		httpClient: client,
		weight:     weight,
		healthCheckPolicy: &HTTPHealthCheckPolicy{
			CircuitBreaker: circuitbreaker.NewWithDefaults[int](),
		},
	}
}

// SetURL sets the base URL of this host.
func (s *Server) SetURL(baseURL string) *Server {
	s.url = baseURL

	return s
}

// URL returns the base URL of this host.
func (s *Server) URL() string {
	return s.url
}

// Name returns the unique string of this host.
func (s *Server) Name() string {
	return s.name
}

// SetName sets the name of this host.
func (s *Server) SetName(name string) *Server {
	s.name = name

	return s
}

// Headers return custom headers of this host.
func (s *Server) Headers() map[string]string {
	return s.headers
}

// SetHeaders sets headers of this host.
func (s *Server) SetHeaders(headers map[string]string) *Server {
	s.headers = headers

	return s
}

// Weight returns the weight of this host.
func (s *Server) Weight() int {
	return s.weight
}

// SetWeight sets the weight of this host.
func (s *Server) SetWeight(weight int) *Server {
	s.weight = weight

	return s
}

// AddCurrentWeight adds the weight to the current weight.
func (s *Server) AddCurrentWeight() {
	s.currentWeight += s.weight
}

// ResetCurrentWeight resets the current weight.
func (h *Server) ResetCurrentWeight(totalWeight int) {
	h.currentWeight -= totalWeight
}

// CurrentWeight adds the weight to the current weight.
func (s *Server) CurrentWeight() int {
	return s.currentWeight
}

// HTTPClient returns the HTTP client of this host.
func (s *Server) HTTPClient() *http.Client {
	return s.httpClient
}

// SetHTTPClient sets the HTTP client of this host.
func (s *Server) SetHTTPClient(client *http.Client) *Server {
	s.httpClient = client

	return s
}

// HealthCheckPolicy returns the HTTP health check policy of this host.
func (s *Server) HealthCheckPolicy() *HTTPHealthCheckPolicy {
	return s.healthCheckPolicy
}

// SetHealthCheckPolicy sets the health check policy to this host.
func (s *Server) SetHealthCheckPolicy(policy *HTTPHealthCheckPolicy) *Server {
	s.healthCheckPolicy = policy

	return s
}

// State returns the circuit breaker state of this host.
func (s *Server) State() circuitbreaker.State {
	if s.healthCheckPolicy == nil {
		return circuitbreaker.ClosedState
	}

	return s.healthCheckPolicy.State()
}

// CheckHealth runs an HTTP request to checking the health of the host.
func (s *Server) CheckHealth(ctx context.Context) {
	if s.healthCheckPolicy == nil || s.healthCheckPolicy.interval <= 0 {
		return
	}

	healthURL := path.Join(s.url, s.healthCheckPolicy.path)

	timeout := s.healthCheckPolicy.interval - time.Second
	if timeout <= 0 {
		timeout = 3 * time.Second
	}

	var body io.Reader

	if len(s.healthCheckPolicy.body) > 0 {
		body = bytes.NewBuffer(s.healthCheckPolicy.body)
	}

	requestContext, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := s.NewRequest(
		requestContext,
		s.healthCheckPolicy.method,
		healthURL,
		body,
	)
	if err != nil {
		s.healthCheckPolicy.RecordError(err)

		return
	}

	for key, header := range s.healthCheckPolicy.headers {
		req.Header.Set(key, header)
	}

	resp, err := s.httpClient.Do(req) //nolint:bodyclose
	if resp == nil {
		s.healthCheckPolicy.RecordError(err)

		return
	}

	if resp.Body != nil {
		goutils.CatchWarnErrorFunc(resp.Body.Close)
	}

	s.healthCheckPolicy.RecordResult(resp.StatusCode)
}

// NewRequest returns a new http.Request given a method, URL, and optional body.
func (s *Server) NewRequest(
	ctx context.Context,
	method string,
	url string,
	body io.Reader,
) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}

	for key, header := range s.headers {
		req.Header.Set(key, header)
	}

	if s.authenticator != nil {
		err := s.authenticator.Authenticate(req)
		if err != nil {
			return req, err
		}
	}

	return req, nil
}

// Do sends an HTTP request and returns an HTTP response, following policy
// (such as redirects, cookies, auth) as configured on the client.
func (s *Server) Do(req *http.Request) (*http.Response, error) {
	resp, err := s.httpClient.Do(req)
	if resp != nil && resp.StatusCode > http.StatusNotImplemented {
		s.healthCheckPolicy.RecordFailure()
	}

	return resp, err
}

// Close terminates internal processes.
func (s *Server) Close() {
	if s.httpClient != nil {
		s.httpClient.CloseIdleConnections()
	}

	if s.healthCheckPolicy != nil {
		s.healthCheckPolicy.Close()
	}
}

// ServerMetrics represents the metrics data of a server.
type ServerMetrics struct {
	// Executions returns the number of executions recorded in the current state when the state is ClosedState or
	// HalfOpenState. When the state is OpenState, this returns the executions recorded during the previous ClosedState.
	//
	// For count based thresholding, the max number of executions is limited to the execution threshold. For time based
	// thresholds, the number of executions may vary within the thresholding period.
	Executions uint `json:"executions"`

	// Failures returns the number of failures recorded in the current state when in a ClosedState or HalfOpenState. When
	// in OpenState, this returns the failures recorded during the previous ClosedState.
	//
	// For count based thresholds, the max number of failures is based on the failure threshold. For time based thresholds,
	// the number of failures may vary within the failure thresholding period.
	Failures uint `json:"failures"`

	// FailureRate returns the rate of failed executions in the current state when in a ClosedState or HalfOpenState. When
	// in OpenState, this returns the rate recorded during the previous ClosedState.
	//
	// The rate is based on the configured failure thresholding capacity.
	FailureRate float64 `json:"failure_rate"`

	// Successes returns the number of successes recorded in the current state when in a ClosedState or HalfOpenState.
	// When in OpenState, this returns the successes recorded during the previous ClosedState.
	//
	// The max number of successes is based on the success threshold.
	Successes uint `json:"successes"`

	// SuccessRate returns rate of successful executions in the current state when in a ClosedState or HalfOpenState. When
	// in OpenState, this returns the successes recorded during the previous ClosedState.
	//
	// The rate is based on the configured success thresholding capacity.
	SuccessRate float64 `json:"success_rate"`
}
