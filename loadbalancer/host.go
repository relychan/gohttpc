package loadbalancer

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"github.com/failsafe-go/failsafe-go/circuitbreaker"
	"github.com/relychan/gohttpc"
	"github.com/relychan/gohttpc/authc/authscheme"
	"github.com/relychan/goutils"
)

// Host represents the host information and its weight to load balance the requests.
type Host struct {
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
	// Cache the last HTTP Error status of the host.
	lastHTTPErrorStatus atomic.Int32
}

var _ gohttpc.HTTPClient = (*Host)(nil)

// NewHost creates an [Host] with a client base URL.
func NewHost(
	client *http.Client,
	baseURL string,
	options ...HostOption,
) (*Host, error) {
	opts := &hostOptions{
		weight: 1,
	}

	for _, opt := range options {
		opt(opts)
	}

	host := &Host{
		httpClient: client,
		weight:     opts.weight,
	}

	u, err := host.SetURL(baseURL)
	if err != nil {
		return nil, err
	}

	if opts.healthCheckPolicyBuilder == nil {
		opts.healthCheckPolicyBuilder = NewHTTPHealthCheckPolicyBuilder()
	}

	host.healthCheckPolicy = opts.healthCheckPolicyBuilder.Build(u)

	return host, nil
}

// SetURL sets the base URL of this host.
// NOTE: the name won't be updated if it is not empty.
func (s *Host) SetURL(baseURL string) (*url.URL, error) {
	u, err := goutils.ParseHTTPURL(baseURL)
	if err != nil {
		return nil, err
	}

	s.url = strings.TrimRight(baseURL, "/")

	if s.name == "" {
		s.name = u.Host
	}

	return u, nil
}

// URL returns the base URL of this host.
func (s *Host) URL() string {
	return s.url
}

// Name returns the unique string of this host.
func (s *Host) Name() string {
	return s.name
}

// SetName sets the name of this host.
func (s *Host) SetName(name string) *Host {
	s.name = name

	return s
}

// Headers return custom headers of this host.
func (s *Host) Headers() map[string]string {
	return s.headers
}

// SetHeaders sets headers of this host.
func (s *Host) SetHeaders(headers map[string]string) *Host {
	s.headers = headers

	return s
}

// Weight returns the weight of this host.
func (s *Host) Weight() int {
	return s.weight
}

// SetWeight sets the weight of this host.
func (s *Host) SetWeight(weight int) *Host {
	s.weight = weight

	return s
}

// AddCurrentWeight adds the weight to the current weight.
func (s *Host) AddCurrentWeight() {
	s.currentWeight += s.weight
}

// ResetCurrentWeight resets the current weight.
func (s *Host) ResetCurrentWeight(totalWeight int) {
	s.currentWeight -= totalWeight
}

// CurrentWeight adds the weight to the current weight.
func (s *Host) CurrentWeight() int {
	return s.currentWeight
}

// HTTPClient returns the HTTP client of this host.
func (s *Host) HTTPClient() *http.Client {
	return s.httpClient
}

// SetHTTPClient sets the HTTP client of this host.
func (s *Host) SetHTTPClient(client *http.Client) *Host {
	s.httpClient = client

	return s
}

// HealthCheckPolicy returns the HTTP health check policy of this host.
func (s *Host) HealthCheckPolicy() *HTTPHealthCheckPolicy {
	return s.healthCheckPolicy
}

// SetHealthCheckPolicy sets the health check policy to this host.
func (s *Host) SetHealthCheckPolicy(policy *HTTPHealthCheckPolicy) *Host {
	s.healthCheckPolicy = policy

	return s
}

// State returns the circuit breaker state of this host.
func (s *Host) State() circuitbreaker.State {
	if s.healthCheckPolicy == nil {
		return circuitbreaker.ClosedState
	}

	return s.healthCheckPolicy.State()
}

// CheckHealth runs an HTTP request to checking the health of the host.
func (s *Host) CheckHealth(ctx context.Context) {
	if s.healthCheckPolicy == nil {
		return
	}

	healthURL := s.url + s.healthCheckPolicy.path

	timeout := s.healthCheckPolicy.timeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	var body io.Reader

	if len(s.healthCheckPolicy.body) > 0 {
		body = bytes.NewBuffer(s.healthCheckPolicy.body)
	}

	requestContext, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := s.newRequest(
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

// GetLastHTTPErrorStatus returns the last HTTP error status,
// and the flag to determine if it is the server outage status.
func (s *Host) GetLastHTTPErrorStatus() (int32, bool) {
	lastHTTPErrorStatus := s.lastHTTPErrorStatus.Load()
	// The gateway timeout status may be caused by the slow backend. It may not be server outage.
	isServerOutage := lastHTTPErrorStatus >= http.StatusBadGateway &&
		lastHTTPErrorStatus != http.StatusGatewayTimeout

	return lastHTTPErrorStatus, isServerOutage
}

// NewRequest returns a new http.Request given a method, URL, and optional body.
func (s *Host) NewRequest(
	ctx context.Context,
	method string,
	url string,
	body io.Reader,
) (*http.Request, error) {
	if s.healthCheckPolicy != nil && s.healthCheckPolicy.State() == circuitbreaker.OpenState {
		lastHTTPErrorStatus, isOutage := s.GetLastHTTPErrorStatus()
		if isOutage {
			// Returns error directly if HTTP status >= 502, except 504.
			return nil, goutils.NewRFC9457Error(int(lastHTTPErrorStatus), "")
		}
	}

	return s.newRequest(ctx, method, url, body)
}

// Do sends an HTTP request and returns an HTTP response, following policy
// (such as redirects, cookies, auth) as configured on the client.
func (s *Host) Do(req *http.Request) (*http.Response, error) {
	resp, err := s.httpClient.Do(req)

	if s.healthCheckPolicy == nil {
		return resp, err
	}

	if resp != nil {
		if resp.StatusCode >= http.StatusInternalServerError {
			s.lastHTTPErrorStatus.Store(int32(resp.StatusCode))
			s.healthCheckPolicy.RecordFailure()
		} else {
			s.healthCheckPolicy.RecordSuccess()
		}
	} else if err != nil {
		s.healthCheckPolicy.RecordFailure()
	}

	return resp, err
}

// Close terminates internal processes.
func (s *Host) Close() {
	if s.httpClient != nil {
		s.httpClient.CloseIdleConnections()
	}

	if s.healthCheckPolicy != nil {
		s.healthCheckPolicy.Close()
	}
}

func (s *Host) newRequest(
	ctx context.Context,
	method string,
	url string,
	body io.Reader,
) (*http.Request, error) {
	reqURL := url

	switch {
	case url == "" || url == "/":
		reqURL = s.url
	case !strings.HasPrefix(url, "http"):
		if url[0] == '/' {
			reqURL = s.url + url
		} else {
			reqURL = s.url + "/" + url
		}

		reqURL = strings.TrimRight(reqURL, "/")
	}

	req, err := http.NewRequestWithContext(ctx, method, reqURL, body)
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

type hostOptions struct {
	weight                   int
	healthCheckPolicyBuilder *HTTPHealthCheckPolicyBuilder
}

// HostOption represents a function to modify host options.
type HostOption func(*hostOptions)

// WithWeight sets the weight for the host.
func WithWeight(weight int) HostOption {
	return func(ho *hostOptions) {
		if weight > 0 {
			ho.weight = weight
		}
		// If weight is not positive, ignore the value.
	}
}

// WithHTTPHealthCheckPolicyBuilder sets the http health check builder for the host.
func WithHTTPHealthCheckPolicyBuilder(builder *HTTPHealthCheckPolicyBuilder) HostOption {
	return func(ho *hostOptions) {
		if builder != nil {
			ho.healthCheckPolicyBuilder = builder
		}
	}
}
