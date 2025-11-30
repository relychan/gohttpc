package roundrobin

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/relychan/gohttpc/loadbalancer"
)

func TestWeightedRoundRobin(t *testing.T) {
	t.Run("3 hosts with weight {5,2,1}", func(t *testing.T) {
		hosts := []*loadbalancer.Server{
			loadbalancer.NewServer(nil, "https://example1.com", 5),
			loadbalancer.NewServer(nil, "https://example2.com", 2),
			loadbalancer.NewServer(nil, "https://example3.com", 1),
		}

		wrr, err := NewWeightedRoundRobin(200*time.Millisecond, hosts)
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
		wrr, err := NewWeightedRoundRobin(
			200*time.Millisecond,
			[]*loadbalancer.Server{
				loadbalancer.NewServer(nil, "https://example1.com", 5),
				loadbalancer.NewServer(nil, "https://example2.com", 5),
			},
		)
		if err != nil {
			t.Fatal(err)
		}
		defer wrr.Close()

		err = wrr.Refresh(
			[]*loadbalancer.Server{
				loadbalancer.NewServer(nil, "https://example3.com", 5),
				loadbalancer.NewServer(nil, "https://example4.com", 5),
			},
		)
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
		wrr, err := NewWeightedRoundRobin(200*time.Millisecond, []*loadbalancer.Server{})
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
