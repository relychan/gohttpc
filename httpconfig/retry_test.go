package httpconfig

import (
	"errors"
	"net/http"
	"testing"

	"github.com/hasura/goenvconf"
)

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
