package loadbalancer

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/failsafe-go/failsafe-go/circuitbreaker"
)

func TestHost_GetLastHTTPErrorStatus(t *testing.T) {
	t.Run("returns zero status when no error has occurred", func(t *testing.T) {
		host, err := NewHost(&http.Client{}, "https://example.com")
		if err != nil {
			t.Fatalf("failed to create host: %v", err)
		}

		status, isOutage := host.GetLastHTTPErrorStatus()

		if status != 0 {
			t.Errorf("expected status 0, got %d", status)
		}

		if isOutage {
			t.Error("expected isOutage to be false when status is 0")
		}
	})

	t.Run("identifies 502 Bad Gateway as server outage", func(t *testing.T) {
		host, err := NewHost(&http.Client{}, "https://example.com")
		if err != nil {
			t.Fatalf("failed to create host: %v", err)
		}

		host.lastHTTPErrorStatus.Store(http.StatusBadGateway)

		status, isOutage := host.GetLastHTTPErrorStatus()

		if status != http.StatusBadGateway {
			t.Errorf("expected status %d, got %d", http.StatusBadGateway, status)
		}

		if !isOutage {
			t.Error("expected isOutage to be true for 502 Bad Gateway")
		}
	})

	t.Run("identifies 503 Service Unavailable as server outage", func(t *testing.T) {
		host, err := NewHost(&http.Client{}, "https://example.com")
		if err != nil {
			t.Fatalf("failed to create host: %v", err)
		}

		host.lastHTTPErrorStatus.Store(http.StatusServiceUnavailable)

		status, isOutage := host.GetLastHTTPErrorStatus()

		if status != http.StatusServiceUnavailable {
			t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, status)
		}

		if !isOutage {
			t.Error("expected isOutage to be true for 503 Service Unavailable")
		}
	})

	t.Run("does not identify 504 Gateway Timeout as server outage", func(t *testing.T) {
		host, err := NewHost(&http.Client{}, "https://example.com")
		if err != nil {
			t.Fatalf("failed to create host: %v", err)
		}

		host.lastHTTPErrorStatus.Store(http.StatusGatewayTimeout)

		status, isOutage := host.GetLastHTTPErrorStatus()

		if status != http.StatusGatewayTimeout {
			t.Errorf("expected status %d, got %d", http.StatusGatewayTimeout, status)
		}

		if isOutage {
			t.Error("expected isOutage to be false for 504 Gateway Timeout")
		}
	})

	t.Run("does not identify 500 Internal Server Error as server outage", func(t *testing.T) {
		host, err := NewHost(&http.Client{}, "https://example.com")
		if err != nil {
			t.Fatalf("failed to create host: %v", err)
		}

		host.lastHTTPErrorStatus.Store(http.StatusInternalServerError)

		status, isOutage := host.GetLastHTTPErrorStatus()

		if status != http.StatusInternalServerError {
			t.Errorf("expected status %d, got %d", http.StatusInternalServerError, status)
		}

		if isOutage {
			t.Error("expected isOutage to be false for 500 Internal Server Error (< 502)")
		}
	})

	t.Run("identifies 505 HTTP Version Not Supported as server outage", func(t *testing.T) {
		host, err := NewHost(&http.Client{}, "https://example.com")
		if err != nil {
			t.Fatalf("failed to create host: %v", err)
		}

		host.lastHTTPErrorStatus.Store(http.StatusHTTPVersionNotSupported)

		status, isOutage := host.GetLastHTTPErrorStatus()

		if status != http.StatusHTTPVersionNotSupported {
			t.Errorf("expected status %d, got %d", http.StatusHTTPVersionNotSupported, status)
		}

		if !isOutage {
			t.Error("expected isOutage to be true for 505 HTTP Version Not Supported")
		}
	})
}

func TestHost_Do_StatusTracking(t *testing.T) {
	t.Run("stores status code >= 500 in lastHTTPErrorStatus", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		host, err := NewHost(&http.Client{}, server.URL)
		if err != nil {
			t.Fatalf("failed to create host: %v", err)
		}

		req, err := http.NewRequest(http.MethodGet, server.URL, nil)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}

		_, err = host.Do(req)
		if err != nil {
			t.Logf("request error (expected): %v", err)
		}

		status, _ := host.GetLastHTTPErrorStatus()
		if status != http.StatusInternalServerError {
			t.Errorf("expected lastHTTPErrorStatus to be %d, got %d", http.StatusInternalServerError, status)
		}
	})

	t.Run("stores 502 Bad Gateway status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadGateway)
		}))
		defer server.Close()

		host, err := NewHost(&http.Client{}, server.URL)
		if err != nil {
			t.Fatalf("failed to create host: %v", err)
		}

		req, err := http.NewRequest(http.MethodGet, server.URL, nil)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}

		_, err = host.Do(req)
		if err != nil {
			t.Logf("request error (expected): %v", err)
		}

		status, isOutage := host.GetLastHTTPErrorStatus()
		if status != http.StatusBadGateway {
			t.Errorf("expected lastHTTPErrorStatus to be %d, got %d", http.StatusBadGateway, status)
		}

		if !isOutage {
			t.Error("expected isOutage to be true for 502 Bad Gateway")
		}
	})

	t.Run("stores 503 Service Unavailable status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
		defer server.Close()

		host, err := NewHost(&http.Client{}, server.URL)
		if err != nil {
			t.Fatalf("failed to create host: %v", err)
		}

		req, err := http.NewRequest(http.MethodGet, server.URL, nil)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}

		_, err = host.Do(req)
		if err != nil {
			t.Logf("request error (expected): %v", err)
		}

		status, isOutage := host.GetLastHTTPErrorStatus()
		if status != http.StatusServiceUnavailable {
			t.Errorf("expected lastHTTPErrorStatus to be %d, got %d", http.StatusServiceUnavailable, status)
		}

		if !isOutage {
			t.Error("expected isOutage to be true for 503 Service Unavailable")
		}
	})

	t.Run("does not overwrite status on successful request", func(t *testing.T) {
		callCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			if callCount == 1 {
				w.WriteHeader(http.StatusBadGateway)
			} else {
				w.WriteHeader(http.StatusOK)
			}
		}))
		defer server.Close()

		host, err := NewHost(&http.Client{}, server.URL)
		if err != nil {
			t.Fatalf("failed to create host: %v", err)
		}

		// First request - should store 502
		req1, _ := http.NewRequest(http.MethodGet, server.URL, nil)
		_, _ = host.Do(req1)

		status, _ := host.GetLastHTTPErrorStatus()
		if status != http.StatusBadGateway {
			t.Errorf("expected lastHTTPErrorStatus to be %d after first request, got %d", http.StatusBadGateway, status)
		}

		// Second request - should not overwrite the error status
		req2, _ := http.NewRequest(http.MethodGet, server.URL, nil)
		_, _ = host.Do(req2)

		status, isOutage := host.GetLastHTTPErrorStatus()
		if status != http.StatusBadGateway {
			t.Errorf("expected lastHTTPErrorStatus to remain %d after successful request, got %d", http.StatusBadGateway, status)
		}

		if !isOutage {
			t.Error("expected isOutage to remain true after successful request")
		}
	})

	t.Run("does not store status < 500", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
		}))
		defer server.Close()

		host, err := NewHost(&http.Client{}, server.URL)
		if err != nil {
			t.Fatalf("failed to create host: %v", err)
		}

		req, err := http.NewRequest(http.MethodGet, server.URL, nil)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}

		_, err = host.Do(req)
		if err != nil {
			t.Logf("request error (expected): %v", err)
		}

		status, _ := host.GetLastHTTPErrorStatus()
		if status != 0 {
			t.Errorf("expected lastHTTPErrorStatus to be 0 for status < 500, got %d", status)
		}
	})
}

func TestHost_NewRequest_CircuitBreakerIntegration(t *testing.T) {
	t.Run("returns error when circuit breaker is open and status indicates outage", func(t *testing.T) {
		// Create a host with a health check policy that has low failure threshold
		builder := NewHTTPHealthCheckPolicyBuilder().
			WithFailureThreshold(1).
			WithSuccessThreshold(1)

		host, err := NewHost(
			&http.Client{},
			"https://example.com",
			WithHTTPHealthCheckPolicyBuilder(builder),
		)
		if err != nil {
			t.Fatalf("failed to create host: %v", err)
		}

		// Simulate a 502 error to open the circuit breaker
		host.lastHTTPErrorStatus.Store(http.StatusBadGateway)
		host.healthCheckPolicy.RecordFailure()

		// Verify circuit breaker is open
		if host.State() != circuitbreaker.OpenState {
			t.Fatalf("expected circuit breaker to be open, got %v", host.State())
		}

		// Try to create a new request - should fail
		req, err := host.NewRequest(context.Background(), http.MethodGet, "/api/test", nil)

		if err == nil {
			t.Error("expected error when circuit breaker is open with server outage status")
		}

		if req != nil {
			t.Error("expected nil request when circuit breaker is open with server outage status")
		}
	})

	t.Run("allows request when circuit breaker is open but status is 504", func(t *testing.T) {
		// Create a host with a health check policy that has low failure threshold
		builder := NewHTTPHealthCheckPolicyBuilder().
			WithFailureThreshold(1).
			WithSuccessThreshold(1)

		host, err := NewHost(
			&http.Client{},
			"https://example.com",
			WithHTTPHealthCheckPolicyBuilder(builder),
		)
		if err != nil {
			t.Fatalf("failed to create host: %v", err)
		}

		// Simulate a 504 error (not considered server outage)
		host.lastHTTPErrorStatus.Store(http.StatusGatewayTimeout)
		host.healthCheckPolicy.RecordFailure()

		// Verify circuit breaker is open
		if host.State() != circuitbreaker.OpenState {
			t.Fatalf("expected circuit breaker to be open, got %v", host.State())
		}

		// Try to create a new request - should succeed because 504 is not considered outage
		req, err := host.NewRequest(context.Background(), http.MethodGet, "/api/test", nil)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if req == nil {
			t.Error("expected request to be created when status is 504")
		}
	})

	t.Run("allows request when circuit breaker is closed", func(t *testing.T) {
		host, err := NewHost(&http.Client{}, "https://example.com")
		if err != nil {
			t.Fatalf("failed to create host: %v", err)
		}

		// Circuit breaker should be closed by default
		if host.State() != circuitbreaker.ClosedState {
			t.Fatalf("expected circuit breaker to be closed, got %v", host.State())
		}

		req, err := host.NewRequest(context.Background(), http.MethodGet, "/api/test", nil)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if req == nil {
			t.Error("expected request to be created when circuit breaker is closed")
		}
	})

	t.Run("returns error with correct status code for 503", func(t *testing.T) {
		builder := NewHTTPHealthCheckPolicyBuilder().
			WithFailureThreshold(1).
			WithSuccessThreshold(1)

		host, err := NewHost(
			&http.Client{},
			"https://example.com",
			WithHTTPHealthCheckPolicyBuilder(builder),
		)
		if err != nil {
			t.Fatalf("failed to create host: %v", err)
		}

		// Simulate a 503 error
		host.lastHTTPErrorStatus.Store(http.StatusServiceUnavailable)
		host.healthCheckPolicy.RecordFailure()

		// Verify circuit breaker is open
		if host.State() != circuitbreaker.OpenState {
			t.Fatalf("expected circuit breaker to be open, got %v", host.State())
		}

		_, err = host.NewRequest(context.Background(), http.MethodGet, "/api/test", nil)

		if err == nil {
			t.Fatal("expected error when circuit breaker is open with 503 status")
		}

		// Check if the error contains the status code
		var rfc9457Err interface{ Status() int }
		if errors.As(err, &rfc9457Err) {
			if rfc9457Err.Status() != http.StatusServiceUnavailable {
				t.Errorf("expected error status %d, got %d", http.StatusServiceUnavailable, rfc9457Err.Status())
			}
		}
	})

	t.Run("allows request when circuit breaker is open but no outage status", func(t *testing.T) {
		builder := NewHTTPHealthCheckPolicyBuilder().
			WithFailureThreshold(1).
			WithSuccessThreshold(1)

		host, err := NewHost(
			&http.Client{},
			"https://example.com",
			WithHTTPHealthCheckPolicyBuilder(builder),
		)
		if err != nil {
			t.Fatalf("failed to create host: %v", err)
		}

		// Simulate a 500 error (not >= 502, so not considered outage)
		host.lastHTTPErrorStatus.Store(http.StatusInternalServerError)
		host.healthCheckPolicy.RecordFailure()

		// Verify circuit breaker is open
		if host.State() != circuitbreaker.OpenState {
			t.Fatalf("expected circuit breaker to be open, got %v", host.State())
		}

		// Try to create a new request - should succeed because 500 is not >= 502
		req, err := host.NewRequest(context.Background(), http.MethodGet, "/api/test", nil)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if req == nil {
			t.Error("expected request to be created when status is 500")
		}
	})
}
