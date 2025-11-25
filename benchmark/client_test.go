package benchmark

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/relychan/gohttpc"
	"resty.dev/v3"
)

const serverURL = "http://localhost:8080/mock"

var randomData = strings.Repeat("1", 1000000)

// goos: darwin
// goarch: arm64
// pkg: github.com/relychan/gohttpc/benchmark
// cpu: Apple M3 Pro
// BenchmarkHTTPClientGet-11    	   32130	     34030 ns/op	    3062 B/op	      38 allocs/op
func BenchmarkHTTPClientGet(b *testing.B) {
	client := http.DefaultClient

	for b.Loop() {
		resp, err := client.Get(serverURL)
		if err != nil {
			continue
		}

		if resp.StatusCode != 200 {
			slog.Error(resp.Status)
		}
		_ = resp.Body.Close()
	}
}

// goos: darwin
// goarch: arm64
// pkg: github.com/relychan/gohttpc/benchmark
// cpu: Apple M3 Pro
// BenchmarkRestyGet-11    	   29535	     38122 ns/op	    5365 B/op	      58 allocs/op
func BenchmarkRestyGet(b *testing.B) {
	client := resty.New()

	defer func() {
		_ = client.Close()
	}()

	for b.Loop() {
		resp, err := client.R().Get(serverURL)
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode() != 200 {
			slog.Error(resp.RawResponse.Status)
		}
	}
}

// goos: darwin
// goarch: arm64
// pkg: github.com/relychan/gohttpc/benchmark
// cpu: Apple M3 Pro
// BenchmarkGoHTTPCGet-11    	   26592	     42641 ns/op	   10731 B/op	     120 allocs/op
func BenchmarkGoHTTPCGet(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	client := gohttpc.NewClient(gohttpc.WithLogger(logger))
	defer client.Close()

	for b.Loop() {
		resp, err := client.NewRequest(http.MethodGet, serverURL).
			Execute(context.TODO())
		if err != nil {
			continue
		}

		_ = resp.Close()

		if resp.StatusCode() != 200 {
			slog.Error(resp.RawResponse.Status)
		}
	}
}

// goos: darwin
// goarch: arm64
// pkg: github.com/relychan/gohttpc/benchmark
// cpu: Apple M3 Pro
// BenchmarkHTTPClientPost-11    	    4237	    279921 ns/op	   53054 B/op	     142 allocs/op
func BenchmarkHTTPClientPost(b *testing.B) {
	client := http.DefaultClient

	for b.Loop() {
		resp, err := client.Post(serverURL, "application/json", strings.NewReader(randomData))
		if err != nil {
			continue
		}

		_ = resp.Body.Close()

		if resp.StatusCode != 200 {
			slog.Error(resp.Status)
		}
	}
}

// goos: darwin
// goarch: arm64
// pkg: github.com/relychan/gohttpc/benchmark
// cpu: Apple M3 Pro
// BenchmarkRestyPost-11    	     939	   1567244 ns/op	 5295167 B/op	     183 allocs/op
func BenchmarkRestyPost(b *testing.B) {
	client := resty.New()

	defer func() {
		_ = client.Close()
	}()

	for b.Loop() {
		resp, err := client.R().SetBody(randomData).Post(serverURL)
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode() != 200 {
			slog.Error(resp.RawResponse.Status)
		}
	}
}

// goos: darwin
// goarch: arm64
// pkg: github.com/relychan/gohttpc/benchmark
// cpu: Apple M3 Pro
// BenchmarkGoHTTPCPost-11    	    3888	    297669 ns/op	   60609 B/op	     223 allocs/op
func BenchmarkGoHTTPCPost(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	client := gohttpc.NewClient(gohttpc.WithLogger(logger))
	defer client.Close()

	for b.Loop() {
		resp, err := client.NewRequest(http.MethodPost, serverURL).
			SetBody(strings.NewReader(randomData)).
			Execute(context.TODO())
		if err != nil {
			continue
		}

		if resp.StatusCode() != 200 {
			slog.Error(resp.RawResponse.Status)
		}

		_ = resp.Close()
	}
}

// goos: darwin
// goarch: arm64
// pkg: github.com/relychan/gohttpc/benchmark
// cpu: Apple M3 Pro
// BenchmarkGoHTTPCPostWithClientTrace-11    	    3907	    301464 ns/op	   63491 B/op	     273 allocs/op
func BenchmarkGoHTTPCPostWithClientTrace(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	client := gohttpc.NewClient(gohttpc.EnableClientTrace(true), gohttpc.WithLogger(logger))
	defer client.Close()

	for b.Loop() {
		resp, err := client.NewRequest(http.MethodPost, serverURL).
			SetBody(strings.NewReader(randomData)).
			Execute(context.TODO())
		if err != nil {
			continue
		}

		if resp.StatusCode() != 200 {
			slog.Error(resp.RawResponse.Status)
		}
		_ = resp.Close()
	}
}
