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
	hosts                []*Host
	nextFunc             func() (*Host, error)
	startHealthCheckFunc func(ctx context.Context)
	closeFunc            func() error
	healthCheckCalled    bool
	closeCalled          bool
}

func (m *mockLoadBalancer) Hosts() []*Host {
	return m.hosts
}

func (m *mockLoadBalancer) Next() (*Host, error) {
	if m.nextFunc != nil {
		return m.nextFunc()
	}
	if len(m.hosts) == 0 {
		return nil, ErrNoActiveHost
	}

	return m.hosts[0], nil
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
		host, err := NewHost(&http.Client{}, "https://example.com")
		if err != nil {
			t.Fatal(err)
		}

		lb := &mockLoadBalancer{
			hosts: []*Host{host},
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

		sm := client.ServerMetrics()
		if len(sm) != 1 {
			t.Error("expected server metrics to have 1 item")
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
		expectedServer, err := NewHost(&http.Client{}, "https://example.com")
		if err != nil {
			t.Fatal(err)
		}

		lb := &mockLoadBalancer{
			hosts: []*Host{expectedServer},
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
			nextFunc: func() (*Host, error) {
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
			nextFunc: func() (*Host, error) {
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
		server, err := NewHost(&http.Client{}, "https://example.com")
		if err != nil {
			t.Fatal(err)
		}

		lb := &mockLoadBalancer{
			hosts: []*Host{server},
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
		server1, err := NewHost(&http.Client{}, "https://example1.com")
		if err != nil {
			t.Fatal(err)
		}

		server2, err := NewHost(&http.Client{}, "https://example2.com")
		if err != nil {
			t.Fatal(err)
		}

		server3, err := NewHost(&http.Client{}, "https://example3.com")
		if err != nil {
			t.Fatal(err)
		}

		servers := []*Host{server1, server2, server3}
		lb := &mockLoadBalancer{
			hosts: servers,
		}
		client := NewLoadBalancerClient(lb)

		if len(lb.Hosts()) != 3 {
			t.Errorf("expected 3 servers, got %d", len(lb.Hosts()))
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
