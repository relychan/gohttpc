package httpconfig

import (
	"errors"
	"net/http"
	"testing"

	"github.com/hasura/goenvconf"
)

func TestHTTPRetryConfig_IsZero(t *testing.T) {
	t.Run("returns true when all fields are nil or empty", func(t *testing.T) {
		config := HTTPRetryConfig{}

		if !config.IsZero() {
			t.Error("expected IsZero to return true")
		}
	})

	t.Run("returns false when MaxAttempts is set", func(t *testing.T) {
		maxAttempts := goenvconf.NewEnvIntValue(3)
		config := HTTPRetryConfig{
			MaxAttempts: &maxAttempts,
		}

		if config.IsZero() {
			t.Error("expected IsZero to return false")
		}
	})

	t.Run("returns false when Delay is set", func(t *testing.T) {
		delay := int64(1000)
		config := HTTPRetryConfig{
			Delay: &delay,
		}

		if config.IsZero() {
			t.Error("expected IsZero to return false")
		}
	})

	t.Run("returns false when MaxDelay is set", func(t *testing.T) {
		maxDelay := int64(5000)
		config := HTTPRetryConfig{
			MaxDelay: &maxDelay,
		}

		if config.IsZero() {
			t.Error("expected IsZero to return false")
		}
	})

	t.Run("returns false when HTTPStatus is set", func(t *testing.T) {
		config := HTTPRetryConfig{
			HTTPStatus: []int{500, 502},
		}

		if config.IsZero() {
			t.Error("expected IsZero to return false")
		}
	})

	t.Run("returns false when Multiplier is set", func(t *testing.T) {
		multiplier := 1.5
		config := HTTPRetryConfig{
			Multiplier: &multiplier,
		}

		if config.IsZero() {
			t.Error("expected IsZero to return false")
		}
	})

	t.Run("returns false when Jitter is set", func(t *testing.T) {
		jitter := int64(100)
		config := HTTPRetryConfig{
			Jitter: &jitter,
		}

		if config.IsZero() {
			t.Error("expected IsZero to return false")
		}
	})

	t.Run("returns false when JitterFactor is set", func(t *testing.T) {
		jitterFactor := 0.25
		config := HTTPRetryConfig{
			JitterFactor: &jitterFactor,
		}

		if config.IsZero() {
			t.Error("expected IsZero to return false")
		}
	})
}

func TestHTTPRetryConfig_Equal(t *testing.T) {
	t.Run("returns true for two empty configs", func(t *testing.T) {
		config1 := HTTPRetryConfig{}
		config2 := HTTPRetryConfig{}

		if !config1.Equal(config2) {
			t.Error("expected Equal to return true for two empty configs")
		}
	})

	t.Run("returns true for identical configs with all fields", func(t *testing.T) {
		maxAttempts := goenvconf.NewEnvIntValue(3)
		delay := int64(1000)
		maxDelay := int64(5000)
		multiplier := 1.5
		jitter := int64(100)
		jitterFactor := 0.25

		config1 := HTTPRetryConfig{
			MaxAttempts:  &maxAttempts,
			Delay:        &delay,
			MaxDelay:     &maxDelay,
			HTTPStatus:   []int{500, 502, 503},
			Multiplier:   &multiplier,
			Jitter:       &jitter,
			JitterFactor: &jitterFactor,
		}

		config2 := HTTPRetryConfig{
			MaxAttempts:  &maxAttempts,
			Delay:        &delay,
			MaxDelay:     &maxDelay,
			HTTPStatus:   []int{500, 502, 503},
			Multiplier:   &multiplier,
			Jitter:       &jitter,
			JitterFactor: &jitterFactor,
		}

		if !config1.Equal(config2) {
			t.Error("expected Equal to return true for identical configs")
		}
	})

	t.Run("returns false for different MaxAttempts", func(t *testing.T) {
		maxAttempts1 := goenvconf.NewEnvIntValue(3)
		maxAttempts2 := goenvconf.NewEnvIntValue(5)

		config1 := HTTPRetryConfig{
			MaxAttempts: &maxAttempts1,
		}
		config2 := HTTPRetryConfig{
			MaxAttempts: &maxAttempts2,
		}

		if config1.Equal(config2) {
			t.Error("expected Equal to return false for different MaxAttempts")
		}
	})

	t.Run("returns false for different Delay", func(t *testing.T) {
		delay1 := int64(1000)
		delay2 := int64(2000)

		config1 := HTTPRetryConfig{
			Delay: &delay1,
		}
		config2 := HTTPRetryConfig{
			Delay: &delay2,
		}

		if config1.Equal(config2) {
			t.Error("expected Equal to return false for different Delay")
		}
	})

	t.Run("returns false for different MaxDelay", func(t *testing.T) {
		maxDelay1 := int64(5000)
		maxDelay2 := int64(10000)

		config1 := HTTPRetryConfig{
			MaxDelay: &maxDelay1,
		}
		config2 := HTTPRetryConfig{
			MaxDelay: &maxDelay2,
		}

		if config1.Equal(config2) {
			t.Error("expected Equal to return false for different MaxDelay")
		}
	})

	t.Run("returns false for different HTTPStatus", func(t *testing.T) {
		config1 := HTTPRetryConfig{
			HTTPStatus: []int{500, 502},
		}
		config2 := HTTPRetryConfig{
			HTTPStatus: []int{500, 503},
		}

		if config1.Equal(config2) {
			t.Error("expected Equal to return false for different HTTPStatus")
		}
	})

	t.Run("returns true for HTTPStatus in different order", func(t *testing.T) {
		config1 := HTTPRetryConfig{
			HTTPStatus: []int{500, 502, 503},
		}
		config2 := HTTPRetryConfig{
			HTTPStatus: []int{503, 500, 502},
		}

		if !config1.Equal(config2) {
			t.Error("expected Equal to return true for HTTPStatus in different order")
		}
	})

	t.Run("returns false for different Multiplier", func(t *testing.T) {
		multiplier1 := 1.5
		multiplier2 := 2.0

		config1 := HTTPRetryConfig{
			Multiplier: &multiplier1,
		}
		config2 := HTTPRetryConfig{
			Multiplier: &multiplier2,
		}

		if config1.Equal(config2) {
			t.Error("expected Equal to return false for different Multiplier")
		}
	})

	t.Run("returns false for different Jitter", func(t *testing.T) {
		jitter1 := int64(100)
		jitter2 := int64(200)

		config1 := HTTPRetryConfig{
			Jitter: &jitter1,
		}
		config2 := HTTPRetryConfig{
			Jitter: &jitter2,
		}

		if config1.Equal(config2) {
			t.Error("expected Equal to return false for different Jitter")
		}
	})

	t.Run("returns false for different JitterFactor", func(t *testing.T) {
		jitterFactor1 := 0.25
		jitterFactor2 := 0.5

		config1 := HTTPRetryConfig{
			JitterFactor: &jitterFactor1,
		}
		config2 := HTTPRetryConfig{
			JitterFactor: &jitterFactor2,
		}

		if config1.Equal(config2) {
			t.Error("expected Equal to return false for different JitterFactor")
		}
	})

	t.Run("returns false when one has field and other doesn't", func(t *testing.T) {
		maxAttempts := goenvconf.NewEnvIntValue(3)

		config1 := HTTPRetryConfig{
			MaxAttempts: &maxAttempts,
		}
		config2 := HTTPRetryConfig{}

		if config1.Equal(config2) {
			t.Error("expected Equal to return false")
		}
	})
}

func TestHTTPRetryConfig_ToRetryPolicy(t *testing.T) {
	t.Run("returns nil when MaxAttempts is nil", func(t *testing.T) {
		config := HTTPRetryConfig{}

		policy, err := config.ToRetryPolicy()

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if policy != nil {
			t.Error("expected policy to be nil")
		}
	})

	t.Run("creates retry policy with valid config", func(t *testing.T) {
		maxAttempts := goenvconf.NewEnvIntValue(3)
		delay := int64(1000)
		maxDelay := int64(5000)
		multiplier := 1.5

		config := HTTPRetryConfig{
			MaxAttempts: &maxAttempts,
			Delay:       &delay,
			MaxDelay:    &maxDelay,
			Multiplier:  &multiplier,
		}

		policy, err := config.ToRetryPolicy()

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if policy == nil {
			t.Error("expected policy to be created")
		}
	})

	t.Run("returns error when MaxAttempts is negative", func(t *testing.T) {
		maxAttempts := goenvconf.NewEnvIntValue(-1)

		config := HTTPRetryConfig{
			MaxAttempts: &maxAttempts,
		}

		policy, err := config.ToRetryPolicy()

		if err == nil {
			t.Error("expected error for negative max attempts")
		}

		if policy != nil {
			t.Error("expected policy to be nil")
		}

		if !errors.Is(err, errRetryPolicyTimesPositive) {
			t.Errorf("expected errRetryPolicyTimesPositive, got %v", err)
		}
	})

	t.Run("returns error when Delay is negative", func(t *testing.T) {
		maxAttempts := goenvconf.NewEnvIntValue(3)
		delay := int64(-100)

		config := HTTPRetryConfig{
			MaxAttempts: &maxAttempts,
			Delay:       &delay,
		}

		_, err := config.ToRetryPolicy()

		if err == nil {
			t.Error("expected error for negative delay")
		}

		if !errors.Is(err, errRetryPolicyDelayPositive) {
			t.Errorf("expected errRetryPolicyDelayPositive, got %v", err)
		}
	})

	t.Run("returns error when Multiplier is less than 1", func(t *testing.T) {
		maxAttempts := goenvconf.NewEnvIntValue(3)
		multiplier := 0.5

		config := HTTPRetryConfig{
			MaxAttempts: &maxAttempts,
			Multiplier:  &multiplier,
		}

		_, err := config.ToRetryPolicy()

		if err == nil {
			t.Error("expected error for invalid multiplier")
		}

		if !errors.Is(err, errRetryPolicyInvalidMultiplier) {
			t.Errorf("expected errRetryPolicyInvalidMultiplier, got %v", err)
		}
	})

	t.Run("returns error when HTTPStatus contains invalid status code", func(t *testing.T) {
		maxAttempts := goenvconf.NewEnvIntValue(3)

		config := HTTPRetryConfig{
			MaxAttempts: &maxAttempts,
			HTTPStatus:  []int{200, 500}, // 200 is invalid for retry
		}

		_, err := config.ToRetryPolicy()

		if err == nil {
			t.Error("expected error for invalid HTTP status")
		}

		if !errors.Is(err, errRetryPolicyInvalidHTTPStatus) {
			t.Errorf("expected errRetryPolicyInvalidHTTPStatus, got %v", err)
		}
	})

	t.Run("creates policy with jitter", func(t *testing.T) {
		maxAttempts := goenvconf.NewEnvIntValue(3)
		jitter := int64(100)

		config := HTTPRetryConfig{
			MaxAttempts: &maxAttempts,
			Jitter:      &jitter,
		}

		policy, err := config.ToRetryPolicy()

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if policy == nil {
			t.Error("expected policy to be created")
		}
	})

	t.Run("creates policy with jitter factor", func(t *testing.T) {
		maxAttempts := goenvconf.NewEnvIntValue(3)
		jitterFactor := 0.25

		config := HTTPRetryConfig{
			MaxAttempts:  &maxAttempts,
			JitterFactor: &jitterFactor,
		}

		policy, err := config.ToRetryPolicy()

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if policy == nil {
			t.Error("expected policy to be created")
		}
	})

	t.Run("creates policy with custom HTTP status codes", func(t *testing.T) {
		maxAttempts := goenvconf.NewEnvIntValue(3)

		config := HTTPRetryConfig{
			MaxAttempts: &maxAttempts,
			HTTPStatus:  []int{408, 429, 503},
		}

		policy, err := config.ToRetryPolicy()

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if policy == nil {
			t.Error("expected policy to be created")
		}
	})

	t.Run("creates policy with constant delay when maxDelay <= delay", func(t *testing.T) {
		maxAttempts := goenvconf.NewEnvIntValue(3)
		delay := int64(1000)
		maxDelay := int64(1000)

		config := HTTPRetryConfig{
			MaxAttempts: &maxAttempts,
			Delay:       &delay,
			MaxDelay:    &maxDelay,
		}

		policy, err := config.ToRetryPolicy()

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if policy == nil {
			t.Error("expected policy to be created")
		}
	})
}

func TestRetryHandleFunc(t *testing.T) {
	t.Run("retries on 429 Too Many Requests", func(t *testing.T) {
		handleFunc := retryHandleFunc([]int{})

		resp := &http.Response{
			StatusCode: http.StatusTooManyRequests,
		}

		if !handleFunc(resp, nil) {
			t.Error("expected to retry on 429")
		}
	})

	t.Run("retries on 5xx errors except 501", func(t *testing.T) {
		handleFunc := retryHandleFunc([]int{})

		testCases := []struct {
			StatusCode  int
			ShouldRetry bool
			Description string
		}{
			{500, true, "500 Internal Server Error"},
			{501, false, "501 Not Implemented"},
			{502, true, "502 Bad Gateway"},
			{503, true, "503 Service Unavailable"},
			{504, true, "504 Gateway Timeout"},
		}

		for _, tc := range testCases {
			t.Run(tc.Description, func(t *testing.T) {
				resp := &http.Response{
					StatusCode: tc.StatusCode,
				}

				result := handleFunc(resp, nil)

				if result != tc.ShouldRetry {
					t.Errorf("expected retry=%v for status %d, got %v", tc.ShouldRetry, tc.StatusCode, result)
				}
			})
		}
	})

	t.Run("retries on custom HTTP status codes", func(t *testing.T) {
		handleFunc := retryHandleFunc([]int{408, 503})

		resp := &http.Response{
			StatusCode: 408,
		}

		if !handleFunc(resp, nil) {
			t.Error("expected to retry on custom status 408")
		}
	})

	t.Run("does not retry on unsupported protocol scheme error", func(t *testing.T) {
		handleFunc := retryHandleFunc([]int{})

		err := errors.New("unsupported protocol scheme")

		if handleFunc(nil, err) {
			t.Error("expected not to retry on unsupported protocol scheme")
		}
	})

	t.Run("does not retry on certificate not trusted error", func(t *testing.T) {
		handleFunc := retryHandleFunc([]int{})

		err := errors.New("certificate is not trusted")

		if handleFunc(nil, err) {
			t.Error("expected not to retry on certificate not trusted")
		}
	})

	t.Run("does not retry on stopped after redirects error", func(t *testing.T) {
		handleFunc := retryHandleFunc([]int{})

		err := errors.New("stopped after 10 redirects")

		if handleFunc(nil, err) {
			t.Error("expected not to retry on stopped after redirects")
		}
	})

	t.Run("retries on other errors", func(t *testing.T) {
		handleFunc := retryHandleFunc([]int{})

		err := errors.New("connection refused")

		if !handleFunc(nil, err) {
			t.Error("expected to retry on connection refused")
		}
	})

	t.Run("does not retry when response is nil and error is nil", func(t *testing.T) {
		handleFunc := retryHandleFunc([]int{})

		if handleFunc(nil, nil) {
			t.Error("expected not to retry when both response and error are nil")
		}
	})

	t.Run("does not retry on 2xx status codes", func(t *testing.T) {
		handleFunc := retryHandleFunc([]int{})

		resp := &http.Response{
			StatusCode: http.StatusOK,
		}

		if handleFunc(resp, nil) {
			t.Error("expected not to retry on 200 OK")
		}
	})
}
