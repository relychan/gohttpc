package httpconfig

import (
	"errors"
	"time"

	"github.com/cenkalti/backoff/v5"
	"github.com/hasura/goenvconf"
	"github.com/relychan/gohttpc"
)

var (
	errRetryPolicyTimesPositive     = errors.New("retry policy times must be positive")
	errRetryPolicyDelayPositive     = errors.New("retry delay must be larger than 0")
	errRetryPolicyInvalidHTTPStatus = errors.New("retry http status must be in between 400 and 599")
	errRetryPolicyInvalidJitter     = errors.New("jitter must be in range (0, 1)")
	errRetryPolicyInvalidMultiplier = errors.New("retry multiplier must be >= 1")
)

// HTTPRetryConfig represents retry policy settings.
type HTTPRetryConfig struct {
	// Maximum number of retry attempts.
	Times *goenvconf.EnvInt `json:"times,omitempty" mapstructure:"times" yaml:"times,omitempty"`
	// The initial wait time in milliseconds before a retry is attempted.
	// Must be >0. Defaults to 1 second.
	Delay *goenvconf.EnvInt `json:"delay,omitempty" mapstructure:"delay" yaml:"delay,omitempty"`
	// HTTPStatus retries if the remote service returns one of these http status
	HTTPStatus []int `json:"httpStatus,omitempty" mapstructure:"httpStatus" yaml:"httpStatus,omitempty"`

	// How much does the reconnection time vary relative to the base value.
	// This is useful to prevent multiple clients to reconnect at the exact
	// same time, as it makes the wait times distinct.
	// Must be in range (0, 1); Defaults to 0.5.
	Jitter *float64 `json:"jitter,omitempty" jsonschema:"nullable,min=0,max=1" mapstructure:"jitter" yaml:"jitter,omitempty"`
	// How much should the reconnection time grow on subsequent attempts.
	// Must be >=1; 1 = constant interval. Defaults to 1.5.
	Multiplier float64 `json:"multiplier,omitempty" jsonschema:"min=1" mapstructure:"multiplier" yaml:"multiplier,omitempty"`
}

// Validate if the current instance is valid.
func (rs HTTPRetryConfig) Validate() (*gohttpc.RetryPolicy, error) {
	var (
		errs         []error
		err          error
		times, delay int64
	)

	if rs.Times != nil {
		times, err = rs.Times.Get()
		if err != nil {
			errs = append(errs, err)
		} else if times < 0 {
			errs = append(errs, errRetryPolicyTimesPositive)
		}
	}

	if rs.Delay != nil {
		delay, err = rs.Delay.Get()
		if err != nil {
			errs = append(errs, err)
		} else if delay < 0 {
			errs = append(errs, errRetryPolicyDelayPositive)
		}
	}

	for _, status := range rs.HTTPStatus {
		if status < 400 || status >= 600 {
			errs = append(errs, errRetryPolicyInvalidHTTPStatus)

			break
		}
	}

	result := &gohttpc.RetryPolicy{
		Times:                 uint(times), //nolint:gosec
		Delay:                 uint(delay), //nolint:gosec
		HTTPStatus:            rs.HTTPStatus,
		Multiplier:            backoff.DefaultMultiplier,
		MaxElapsedTimeSeconds: uint(backoff.DefaultMaxElapsedTime / time.Second),
	}

	if rs.Jitter != nil {
		if *rs.Jitter < 0 || *rs.Jitter > 1 {
			errs = append(errs, errRetryPolicyInvalidJitter)
		} else {
			result.Jitter = rs.Jitter
		}
	}

	if rs.Multiplier != 0 {
		if rs.Multiplier < 1 {
			errs = append(errs, errRetryPolicyInvalidMultiplier)
		} else {
			result.Multiplier = rs.Multiplier
		}
	}

	if len(errs) > 0 {
		return result, errors.Join(errs...)
	}

	return result, nil
}
