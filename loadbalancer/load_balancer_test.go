package loadbalancer

import (
	"context"
	"errors"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/relychan/gohttpc"
)

// mockLoadBalancer is a mock implementation of LoadBalancer for testing.
type mockLoadBalancer struct {
	servers              []*Server
	nextFunc             func() (*Server, error)
	startHealthCheckFunc func(ctx context.Context)
	closeFunc            func() error
	healthCheckCalled    bool
	closeCalled          bool
}

func (m *mockLoadBalancer) Servers() []*Server {
	return m.servers
}

func (m *mockLoadBalancer) Next() (*Server, error) {
	if m.nextFunc != nil {
		return m.nextFunc()
	}
	if len(m.servers) == 0 {
		return nil, ErrNoActiveHost
	}
	return m.servers[0], nil
}

func (m *mockLoadBalancer) StartHealthCheck(ctx context.Context) {
	m.healthCheckCalled = true
	if m.startHealthCheckFunc != nil {
		m.startHealthCheckFunc(ctx)
	}
}

func (m *mockLoadBalancer) Close() error {
	m.closeCalled = true
	if m.closeFunc != nil {
		return m.closeFunc()
	}
	return nil
}

func TestNewLoadBalancerClient(t *testing.T) {
	t.Run("creates client with default options", func(t *testing.T) {
		lb := &mockLoadBalancer{
			servers: []*Server{
				NewServer(&http.Client{}, "https://example.com", 1),
			},
		}

		client := NewLoadBalancerClient(lb)

		if client == nil {
			t.Fatal("expected client to be created")
		}

		if client.loadBalancer != lb {
			t.Error("expected load balancer to be set")
		}

		if client.options == nil {
			t.Error("expected options to be initialized")
		}
	})

	t.Run("creates client with custom options", func(t *testing.T) {
		lb := &mockLoadBalancer{}
		timeout := 5 * time.Second

		client := NewLoadBalancerClient(lb, gohttpc.WithTimeout(timeout))

		if client == nil {
			t.Fatal("expected client to be created")
		}

		if client.options.Timeout != timeout {
			t.Errorf("expected timeout %v, got %v", timeout, client.options.Timeout)
		}
	})
}

func TestNewLoadBalancerClientWithOptions(t *testing.T) {
	t.Run("creates client with explicit options", func(t *testing.T) {
		lb := &mockLoadBalancer{}
		opts := gohttpc.NewClientOptions(gohttpc.WithTimeout(10 * time.Second))

		client := NewLoadBalancerClientWithOptions(lb, opts)

		if client == nil {
			t.Fatal("expected client to be created")
		}

		if client.options != opts {
			t.Error("expected options to match provided options")
		}

		if client.loadBalancer != lb {
			t.Error("expected load balancer to be set")
		}
	})
}

func TestLoadBalancerClient_R(t *testing.T) {
	t.Run("creates request with client", func(t *testing.T) {
		lb := &mockLoadBalancer{}
		client := NewLoadBalancerClient(lb)

		req := client.R(http.MethodGet, "https://example.com/api")

		if req == nil {
			t.Fatal("expected request to be created")
		}

		if req.Method() != http.MethodGet {
			t.Errorf("expected method GET, got %s", req.Method())
		}

		if req.URL() != "https://example.com/api" {
			t.Errorf("expected URL https://example.com/api, got %s", req.URL())
		}
	})

	t.Run("creates POST request", func(t *testing.T) {
		lb := &mockLoadBalancer{}
		client := NewLoadBalancerClient(lb)

		req := client.R(http.MethodPost, "https://example.com/api/create")

		if req == nil {
			t.Fatal("expected request to be created")
		}

		if req.Method() != http.MethodPost {
			t.Errorf("expected method POST, got %s", req.Method())
		}
	})
}

func TestLoadBalancerClient_HTTPClient(t *testing.T) {
	t.Run("returns server from load balancer", func(t *testing.T) {
		expectedServer := NewServer(&http.Client{}, "https://example.com", 1)
		lb := &mockLoadBalancer{
			servers: []*Server{expectedServer},
		}
		client := NewLoadBalancerClient(lb)

		httpClient, err := client.HTTPClient()

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if httpClient != expectedServer {
			t.Error("expected to get the server from load balancer")
		}
	})

	t.Run("returns error when no active host", func(t *testing.T) {
		lb := &mockLoadBalancer{
			nextFunc: func() (*Server, error) {
				return nil, ErrNoActiveHost
			},
		}
		client := NewLoadBalancerClient(lb)

		_, err := client.HTTPClient()

		if !errors.Is(err, ErrNoActiveHost) {
			t.Errorf("expected ErrNoActiveHost, got %v", err)
		}
	})

	t.Run("returns custom error from load balancer", func(t *testing.T) {
		customErr := errors.New("custom load balancer error")
		lb := &mockLoadBalancer{
			nextFunc: func() (*Server, error) {
				return nil, customErr
			},
		}
		client := NewLoadBalancerClient(lb)

		_, err := client.HTTPClient()

		if !errors.Is(err, customErr) {
			t.Errorf("expected custom error, got %v", err)
		}
	})
}

func TestLoadBalancerClient_StartHealthCheck(t *testing.T) {
	t.Run("starts health check on load balancer", func(t *testing.T) {
		lb := &mockLoadBalancer{}
		client := NewLoadBalancerClient(lb)
		ctx := context.Background()

		client.StartHealthCheck(ctx)

		if !lb.healthCheckCalled {
			t.Error("expected StartHealthCheck to be called on load balancer")
		}
	})

	t.Run("handles nil load balancer gracefully", func(t *testing.T) {
		client := &LoadBalancerClient{
			loadBalancer: nil,
			options:      gohttpc.NewClientOptions(),
		}
		ctx := context.Background()

		// Should not panic
		client.StartHealthCheck(ctx)
	})

	t.Run("passes context to load balancer", func(t *testing.T) {
		var receivedCtx context.Context
		lb := &mockLoadBalancer{
			startHealthCheckFunc: func(ctx context.Context) {
				receivedCtx = ctx
			},
		}
		client := NewLoadBalancerClient(lb)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		client.StartHealthCheck(ctx)

		if receivedCtx != ctx {
			t.Error("expected context to be passed to load balancer")
		}
	})
}

func TestLoadBalancerClient_Close(t *testing.T) {
	t.Run("closes load balancer successfully", func(t *testing.T) {
		lb := &mockLoadBalancer{}
		client := NewLoadBalancerClient(lb)

		err := client.Close()

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if !lb.closeCalled {
			t.Error("expected Close to be called on load balancer")
		}
	})

	t.Run("handles nil load balancer gracefully", func(t *testing.T) {
		client := &LoadBalancerClient{
			loadBalancer: nil,
			options:      gohttpc.NewClientOptions(),
		}

		err := client.Close()

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("returns error from load balancer", func(t *testing.T) {
		expectedErr := errors.New("close error")
		lb := &mockLoadBalancer{
			closeFunc: func() error {
				return expectedErr
			},
		}
		client := NewLoadBalancerClient(lb)

		err := client.Close()

		if !errors.Is(err, expectedErr) {
			t.Errorf("expected error %v, got %v", expectedErr, err)
		}
	})
}

func TestErrNoActiveHost(t *testing.T) {
	t.Run("error message is correct", func(t *testing.T) {
		expected := "no active host"
		if ErrNoActiveHost.Error() != expected {
			t.Errorf("expected error message %q, got %q", expected, ErrNoActiveHost.Error())
		}
	})
}

// mockServer is a mock implementation of Server for integration testing.
type mockServer struct {
	url        string
	doFunc     func(req *http.Request) (*http.Response, error)
	newReqFunc func(ctx context.Context, method, url string, body io.Reader) (*http.Request, error)
}

func (m *mockServer) NewRequest(ctx context.Context, method, url string, body io.Reader) (*http.Request, error) {
	if m.newReqFunc != nil {
		return m.newReqFunc(ctx, method, url, body)
	}
	return http.NewRequestWithContext(ctx, method, url, body)
}

func (m *mockServer) Do(req *http.Request) (*http.Response, error) {
	if m.doFunc != nil {
		return m.doFunc(req)
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       http.NoBody,
	}, nil
}

func TestLoadBalancerClient_Integration(t *testing.T) {
	t.Run("full request flow with load balancer", func(t *testing.T) {
		server := NewServer(&http.Client{}, "https://example.com", 1)
		lb := &mockLoadBalancer{
			servers: []*Server{server},
		}
		client := NewLoadBalancerClient(lb)

		httpClient, err := client.HTTPClient()
		if err != nil {
			t.Fatalf("unexpected error getting http client: %v", err)
		}

		if httpClient == nil {
			t.Fatal("expected http client to be returned")
		}
	})

	t.Run("multiple servers in load balancer", func(t *testing.T) {
		servers := []*Server{
			NewServer(&http.Client{}, "https://example1.com", 1),
			NewServer(&http.Client{}, "https://example2.com", 1),
			NewServer(&http.Client{}, "https://example3.com", 1),
		}
		lb := &mockLoadBalancer{
			servers: servers,
		}
		client := NewLoadBalancerClient(lb)

		if len(lb.Servers()) != 3 {
			t.Errorf("expected 3 servers, got %d", len(lb.Servers()))
		}

		httpClient, err := client.HTTPClient()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if httpClient == nil {
			t.Fatal("expected http client to be returned")
		}
	})
}
