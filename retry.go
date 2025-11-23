package gohttpc

import (
	"net/http"
	"strconv"
	"time"

	backoff "github.com/cenkalti/backoff/v5"
)

var defaultRetryHTTPStatus = []int{408, 429, 500, 502, 503}

// RetryPolicy represents the retry policy of request.
type RetryPolicy struct {
	// Maximum number of retry attempts. Defaults to 0 (no retry).
	Times uint `json:"times,omitempty" mapstructure:"times" yaml:"times,omitempty"`
	// The initial interval is used to calculate the next retry time in milliseconds. Defaults to 1 second.
	Delay uint `json:"delay,omitempty" mapstructure:"delay" yaml:"delay,omitempty"`
	// HTTPStatus retries if the remote service returns one of these http status
	HTTPStatus []int `json:"httpStatus,omitempty" mapstructure:"httpStatus" yaml:"httpStatus,omitempty"`
	// How much does the reconnection time vary relative to the base value.
	// This is useful to prevent multiple clients to reconnect at the exact
	// same time, as it makes the wait times distinct.
	// Must be in range (0, 1); Defaults to 0.5.
	Jitter *float64 `json:"jitter,omitempty" mapstructure:"jitter" yaml:"jitter,omitempty"`
	// How much should the reconnection time grow on subsequent attempts.
	// Must be >=1; 1 = constant interval. Defaults to 1.5.
	Multiplier float64 `json:"multiplier,omitempty" mapstructure:"multiplier" yaml:"multiplier,omitempty"`
	// How much can the wait time grow.
	// If <=0 = the wait time can infinitely grow. Defaults to 60 seconds.
	MaxIntervalSeconds uint `json:"maxIntervalSeconds,omitempty" mapstructure:"maxIntervalSeconds" yaml:"maxIntervalSeconds,omitempty"`
	// Maximum total time in seconds for all retries.
	MaxElapsedTimeSeconds uint `json:"maxElapsedTimeSeconds,omitempty" mapstructure:"maxElapsedTimeSeconds" yaml:"maxElapsedTimeSeconds,omitempty"`
}

// GetMaxElapsedTime returns the max elapsed time duration.
func (rp RetryPolicy) GetMaxElapsedTime() time.Duration {
	if rp.MaxElapsedTimeSeconds > 0 {
		return time.Duration(rp.MaxElapsedTimeSeconds) * time.Second //nolint:gosec
	}

	return backoff.DefaultMaxElapsedTime
}

// GetRetryHTTPStatus returns the http status to be retried.
func (rp RetryPolicy) GetRetryHTTPStatus() []int {
	if len(rp.HTTPStatus) == 0 {
		return defaultRetryHTTPStatus
	}

	return rp.HTTPStatus
}

// GetExponentialBackoff returns a new GetExponentialBackoff config.
func (rp RetryPolicy) GetExponentialBackoff() *backoff.ExponentialBackOff {
	result := backoff.NewExponentialBackOff()

	if rp.Delay > 0 {
		result.InitialInterval = time.Duration(rp.Delay) * time.Millisecond //nolint:gosec
	}

	if rp.Jitter != nil {
		result.RandomizationFactor = *rp.Jitter
	}

	if rp.Multiplier >= 1 {
		result.Multiplier = rp.Multiplier
	}

	if rp.MaxIntervalSeconds > 0 {
		result.MaxInterval = time.Duration(rp.MaxIntervalSeconds) * time.Second //nolint:gosec
	}

	return result
}

// The HTTP [Retry-After] response header indicates how long the user agent should wait before making a follow-up request.
// The client finds this header if exist and decodes to duration.
// If the header doesn't exist or there is any error happened, fallback to the retry delay setting.
//
// [Retry-After]: https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Retry-After
func getRetryAfter(resp *http.Response) int {
	rawRetryAfter := resp.Header.Get("Retry-After")
	if rawRetryAfter == "" {
		return 0
	}

	// A non-negative decimal integer indicating the seconds to delay after the response is received.
	retryAfterSecs, err := strconv.Atoi(rawRetryAfter)
	if err == nil && retryAfterSecs > 0 {
		return retryAfterSecs
	}

	// A date after which to retry, e.g. Tue, 29 Oct 2024 16:56:32 GMT
	retryTime, err := time.Parse(time.RFC1123, rawRetryAfter)
	if err == nil && retryTime.After(time.Now()) {
		duration := time.Until(retryTime)

		return int(duration.Seconds())
	}

	return 0
}
