package roundrobin

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/failsafe-go/failsafe-go/circuitbreaker"
	"github.com/relychan/gohttpc/loadbalancer"
)

func TestWeightedRoundRobin(t *testing.T) {
	t.Run("3 hosts with weight {5,2,1}", func(t *testing.T) {
		server1, err := loadbalancer.NewHost(nil, "https://example1.com", loadbalancer.WithWeight(5))
		if err != nil {
			t.Fatal(err)
		}

		server2, err := loadbalancer.NewHost(nil, "https://example2.com", loadbalancer.WithWeight(2))
		if err != nil {
			t.Fatal(err)
		}

		server3, err := loadbalancer.NewHost(nil, "https://example3.com")
		if err != nil {
			t.Fatal(err)
		}

		hosts := []*loadbalancer.Host{server1, server2, server3}

		wrr, err := NewWeightedRoundRobin(hosts)
		if err != nil {
			t.Fatal(err)
		}
		defer wrr.Close()

		runCount := 5
		var result []string
		for i := 0; i < runCount; i++ {
			server, err := wrr.Next()
			if err != nil {
				t.Fatal(err)
			}

			result = append(result, server.URL())
		}

		expected := []string{
			"https://example1.com", "https://example2.com", "https://example1.com",
			"https://example1.com", "https://example3.com",
		}

		if len(expected) != len(result) {
			t.Fatal("server results aren't equal")
		}

		if fmt.Sprint(expected) != fmt.Sprint(result) {
			t.Fatal("server results aren't equal")
		}
	})

	t.Run("2 hosts with weight {5,5} and refresh", func(t *testing.T) {
		server1, err := loadbalancer.NewHost(nil, "https://example1.com", loadbalancer.WithWeight(5))
		if err != nil {
			t.Fatal(err)
		}

		server2, err := loadbalancer.NewHost(nil, "https://example2.com", loadbalancer.WithWeight(5))
		if err != nil {
			t.Fatal(err)
		}

		wrr, err := NewWeightedRoundRobin(
			[]*loadbalancer.Host{server1, server2},
		)
		if err != nil {
			t.Fatal(err)
		}
		defer wrr.Close()

		host3, err := loadbalancer.NewHost(nil, "https://example3.com", loadbalancer.WithWeight(5))
		if err != nil {
			t.Fatal(err)
		}
		host4, err := loadbalancer.NewHost(nil, "https://example4.com", loadbalancer.WithWeight(5))
		if err != nil {
			t.Fatal(err)
		}

		err = wrr.Refresh([]*loadbalancer.Host{host3, host4})
		if err != nil {
			t.Fatal(err)
		}

		runCount := 5
		var result []string
		for i := 0; i < runCount; i++ {
			server, err := wrr.Next()
			if err != nil {
				t.Fatal(err)
			}

			result = append(result, server.URL())
		}

		expected := []string{
			"https://example3.com", "https://example4.com", "https://example3.com",
			"https://example4.com", "https://example3.com",
		}

		if len(expected) != len(result) {
			t.Fatalf("expected: %v; got: %v", expected, result)
		}

		if fmt.Sprint(expected) != fmt.Sprint(result) {
			t.Fatalf("expected: %v; got: %v", expected, result)
		}
	})

	t.Run("no active hosts error", func(t *testing.T) {
		wrr, err := NewWeightedRoundRobin([]*loadbalancer.Host{})
		if err != nil {
			t.Fatal(err)
		}
		defer wrr.Close()

		_, err = wrr.Next()
		if !errors.Is(err, loadbalancer.ErrNoActiveHost) {
			t.Fatalf("expected error: %v; got: %v", loadbalancer.ErrNoActiveHost, err)
		}
	})
}

func TestWeightedRoundRobinIntegration(t *testing.T) {
	counter1 := atomic.Int32{}
	counter2 := atomic.Int32{}
	counter3 := atomic.Int32{}

	handler1 := http.NewServeMux()
	handler1.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		counter1.Add(1)
		w.WriteHeader(http.StatusOK)
	})

	handler2 := http.NewServeMux()
	handler2.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		counter2.Add(1)
		w.WriteHeader(http.StatusOK)
	})

	handler3 := http.NewServeMux()
	handler3.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		counter3.Add(1)
		w.WriteHeader(http.StatusBadGateway)
	})

	testServer1 := httptest.NewServer(handler1)
	defer testServer1.Close()

	testServer2 := httptest.NewServer(handler2)
	defer testServer2.Close()

	testServer3 := httptest.NewServer(handler3)
	defer testServer3.Close()

	host1, err := loadbalancer.NewHost(http.DefaultClient, testServer1.URL, loadbalancer.WithWeight(2))
	if err != nil {
		t.Fatal(err)
	}

	host2, err := loadbalancer.NewHost(http.DefaultClient, testServer2.URL)
	if err != nil {
		t.Fatal(err)
	}

	host3, err := loadbalancer.NewHost(http.DefaultClient, testServer3.URL)
	if err != nil {
		t.Fatal(err)
	}

	hosts := []*loadbalancer.Host{host1, host2, host3}

	hosts[2].HealthCheckPolicy().CircuitBreaker = circuitbreaker.NewBuilder[int]().
		WithDelay(100 * time.Millisecond).
		WithFailureThreshold(1).Build()

	wrr, err := NewWeightedRoundRobin(hosts, WithHealthCheckInterval(100*time.Millisecond))
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.TODO(), 15*time.Second)
	lb := loadbalancer.NewLoadBalancerClient(wrr)
	go lb.StartHealthCheck(ctx)
	defer lb.Close()

	for range 10 {
		lb.R(http.MethodGet, "/").Execute(context.TODO())
	}

	if counter1.Load() != 6 {
		t.Errorf("expected 6 requests to host 1; got: %d", counter1.Load())
	}

	if counter2.Load() != 3 {
		t.Errorf("expected 3 requests to host 2; got: %d", counter2.Load())
	}

	if counter3.Load() != 1 {
		t.Errorf("expected 1 requests to host 3; got: %d", counter3.Load())
	}

	if hosts[2].State() != circuitbreaker.OpenState {
		t.Errorf("expected open state on host 3; got: %s", hosts[2].State().String())
	}

	time.Sleep(100 * time.Millisecond)
	cancel()

	for range 4 {
		wrr.nextRoundRobin()
	}

	if hosts[2].State() != circuitbreaker.HalfOpenState {
		t.Errorf("expected half-open state on host 3; got: %s", hosts[2].State().String())
	}
}

func TestRoundRobinIntegration(t *testing.T) {
	counter1 := atomic.Int32{}
	counter2 := atomic.Int32{}
	counter3 := atomic.Int32{}

	handler1 := http.NewServeMux()
	handler1.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		counter1.Add(1)
		w.WriteHeader(http.StatusOK)
	})

	handler2 := http.NewServeMux()
	handler2.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		counter2.Add(1)
		w.WriteHeader(http.StatusOK)
	})

	handler3 := http.NewServeMux()
	handler3.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		counter3.Add(1)
		w.WriteHeader(http.StatusBadGateway)
	})

	testServer1 := httptest.NewServer(handler1)
	defer testServer1.Close()

	testServer2 := httptest.NewServer(handler2)
	defer testServer2.Close()

	testServer3 := httptest.NewServer(handler3)
	defer testServer3.Close()

	host1, err := loadbalancer.NewHost(http.DefaultClient, testServer1.URL)
	if err != nil {
		t.Fatal(err)
	}

	host2, err := loadbalancer.NewHost(http.DefaultClient, testServer2.URL)
	if err != nil {
		t.Fatal(err)
	}

	host3, err := loadbalancer.NewHost(
		http.DefaultClient,
		testServer3.URL,
		loadbalancer.WithHTTPHealthCheckPolicyBuilder(loadbalancer.NewHTTPHealthCheckPolicyBuilder().
			WithInterval(100*time.Millisecond).
			WithFailureThreshold(1)))
	if err != nil {
		t.Fatal(err)
	}

	hosts := []*loadbalancer.Host{host1, host2, host3}

	wrr, err := NewWeightedRoundRobin(hosts, WithHealthCheckInterval(100*time.Millisecond))
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.TODO(), 15*time.Second)
	lb := loadbalancer.NewLoadBalancerClient(wrr)
	go lb.StartHealthCheck(ctx)

	for range 10 {
		lb.R(http.MethodGet, "/").Execute(context.TODO())
	}

	if counter1.Load() != 5 {
		t.Errorf("expected 5 requests to host 1; got: %d", counter1.Load())
	}

	if counter2.Load() != 4 {
		t.Errorf("expected 4 requests to host 2; got: %d", counter2.Load())
	}

	if counter3.Load() != 1 {
		t.Errorf("expected 1 requests to host 3; got: %d", counter3.Load())
	}

	if hosts[2].State() != circuitbreaker.OpenState {
		t.Errorf("expected open state on host 3; got: %s", hosts[2].State().String())
	}

	time.Sleep(100 * time.Millisecond)
	cancel()

	for range 3 {
		wrr.nextRoundRobin()
	}

	if hosts[2].State() != circuitbreaker.HalfOpenState {
		t.Errorf("expected half-open state on host 3; got: %s", hosts[2].State().String())
	}
}
