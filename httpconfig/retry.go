package httpconfig

import (
	"context"
	"crypto/x509"
	"errors"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/failsafe-go/failsafe-go/failsafehttp"
	"github.com/failsafe-go/failsafe-go/retrypolicy"
	"github.com/hasura/goenvconf"
)

var (
	errRetryPolicyTimesPositive     = errors.New("retry policy times must be positive")
	errRetryPolicyDelayPositive     = errors.New("retry delay must be larger than 0")
	errRetryPolicyInvalidHTTPStatus = errors.New("retry http status must be in between 400 and 599")
	errRetryPolicyInvalidMultiplier = errors.New("retry multiplier must be >= 1")
)

var stoppedAfterRedirects = regexp.MustCompile(`stopped after \d+ redirects\z`)

// HTTPRetryConfig represents retry policy settings.
type HTTPRetryConfig struct {
	// Maximum number of retry attempts.
	MaxAttempts *goenvconf.EnvInt `json:"maxAttempts,omitempty" mapstructure:"maxAttempts" yaml:"maxAttempts,omitempty"`
	// The initial wait time in milliseconds before a retry is attempted.
	// Must be >0. Defaults to 1 second.
	Delay *int64 `json:"delay,omitempty" mapstructure:"delay" yaml:"delay,omitempty"`
	// The max delay in milliseconds of the exponentially backing off.
	// If the max delay is smaller or equal the base delay. The delay is constant.
	MaxDelay *int64 `json:"maxDelay,omitempty" mapstructure:"maxDelay" yaml:"maxDelay,omitempty"`
	// HTTPStatus retries if the remote service returns one of these http status
	HTTPStatus []int `json:"httpStatus,omitempty" mapstructure:"httpStatus" yaml:"httpStatus,omitempty"`
	// How much should the reconnection time grow on subsequent attempts.
	// Must be >=1; 1 = constant interval. Defaults to 1.5.
	Multiplier *float64 `json:"multiplier,omitempty" jsonschema:"min=1" mapstructure:"multiplier" yaml:"multiplier,omitempty"`
	// For each retry delay, a random portion of the jitter will be added or subtracted to the delay.
	// For example: a jitter of 100 milliseconds will randomly add between -100 and 100 milliseconds to each retry delay.
	// Replaces any previously configured jitter factor.
	Jitter *int64 `json:"jitter,omitempty" mapstructure:"jitter" yaml:"jitter,omitempty"`
	// For each retry delay, a random portion of the delay multiplied by the jitterFactor will be added or subtracted to the delay.
	// For example: a retry delay of 100 milliseconds and a jitterFactor of .25 will result in a random retry delay between 75 and 125 milliseconds.
	// Replaces any previously configured jitter duration.
	JitterFactor *float64 `json:"jitterFactor,omitempty" mapstructure:"jitterFactor" yaml:"jitterFactor,omitempty"`
}

// ToRetryPolicy validates and create the retry policy.
func (rs HTTPRetryConfig) ToRetryPolicy() ( //nolint:funlen
	retrypolicy.RetryPolicy[*http.Response], error,
) {
	var (
		errs       []error
		err        error
		delay      int64 = 1000
		maxDelay   int64
		multiplier = 1.5
	)

	if rs.MaxAttempts == nil {
		return nil, nil //nolint:nilnil
	}

	maxAttempts, err := rs.MaxAttempts.Get()
	if err != nil {
		errs = append(errs, err)
	} else if maxAttempts < 0 {
		errs = append(errs, errRetryPolicyTimesPositive)
	}

	builder := retrypolicy.NewBuilder[*http.Response]().
		WithMaxAttempts(int(maxAttempts))

	if rs.Delay != nil {
		if *rs.Delay < 0 {
			errs = append(errs, errRetryPolicyDelayPositive)
		}

		delay = *rs.Delay
	}

	if rs.MaxDelay != nil {
		maxDelay = *rs.MaxDelay
	}

	if rs.Multiplier != nil {
		if *rs.Multiplier < 1 {
			return nil, errRetryPolicyInvalidMultiplier
		}

		multiplier = *rs.Multiplier
	}

	if rs.Jitter != nil && *rs.Jitter != 0 {
		builder = builder.WithJitter(time.Duration(*rs.Jitter) * time.Millisecond)
	}

	for _, status := range rs.HTTPStatus {
		if status < 400 || status >= 600 {
			errs = append(errs, errRetryPolicyInvalidHTTPStatus)

			break
		}
	}

	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}

	if rs.JitterFactor != nil {
		builder = builder.WithJitterFactor(*rs.JitterFactor)
	}

	if maxDelay <= delay {
		builder = builder.WithDelay(time.Duration(delay) * time.Millisecond)
	} else {
		builder = builder.WithBackoffFactor(
			time.Duration(delay)*time.Millisecond,
			time.Duration(maxDelay)*time.Millisecond,
			multiplier,
		)
	}

	builder = builder.
		HandleIf(retryHandleFunc(rs.HTTPStatus)).
		AbortOnErrors(context.Canceled, context.DeadlineExceeded).
		WithDelayFunc(failsafehttp.DelayFunc)

	return builder.Build(), nil
}

func retryHandleFunc(httpStatus []int) func(resp *http.Response, err error) bool {
	return func(resp *http.Response, err error) bool {
		// Handle errors
		if err != nil {
			errorMsg := err.Error()
			// Do not retry unsupported protocol scheme error
			// This will be a url.Error when using an http.Client, and an errorString when using a RoundTripper
			if strings.Contains(errorMsg, "unsupported protocol scheme") ||
				strings.Contains(errorMsg, "certificate is not trusted") ||
				stoppedAfterRedirects.MatchString(errorMsg) {
				return false
			}

			var urlError *url.Error

			if errors.As(err, &urlError) {
				var uae x509.UnknownAuthorityError
				// Do not retry on unknown authority errors
				if errors.Is(urlError.Err, &uae) {
					return false
				}
			}
			// Retry on all other url errors
			return true
		}

		// Handle response
		if resp != nil {
			if len(httpStatus) > 0 && slices.Contains(httpStatus, resp.StatusCode) {
				return true
			}

			// Retry on 429
			if resp.StatusCode == http.StatusTooManyRequests {
				return true
			}
			// Retry on most 5xx responses
			if resp.StatusCode >= 500 && resp.StatusCode != http.StatusNotImplemented {
				return true
			}
		}

		return false
	}
}
