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
// BenchmarkHTTPClientGet-11    	   25221	     45798 ns/op	    3206 B/op	      39 allocs/op
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
// BenchmarkRestyGet-11    	   22992	     48696 ns/op	    5518 B/op	      59 allocs/op
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
// BenchmarkGoHTTPCGet-11    	   20902	     54623 ns/op	   10584 B/op	     118 allocs/op
func BenchmarkGoHTTPCGet(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	client := gohttpc.NewClient(gohttpc.WithLogger(logger))
	defer client.Close()

	for b.Loop() {
		resp, err := client.R(http.MethodGet, serverURL).
			Execute(context.TODO())
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
// BenchmarkHTTPClientPost-11    	    3142	    343308 ns/op	   53783 B/op	     144 allocs/op
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
// BenchmarkRestyPost-11    	     609	   2613460 ns/op	 2243928 B/op	     182 allocs/op
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
// BenchmarkGoHTTPCPost-11    	    3477	    331348 ns/op	   61018 B/op	     225 allocs/op
func BenchmarkGoHTTPCPost(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	client := gohttpc.NewClient(gohttpc.WithLogger(logger))
	defer client.Close()

	for b.Loop() {
		resp, err := client.R(http.MethodPost, serverURL).
			SetBody(strings.NewReader(randomData)).
			Execute(context.TODO(), client)
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
// BenchmarkGoHTTPCPostWithClientTrace-11    	    2932	    379294 ns/op	   63995 B/op	     264 allocs/op
func BenchmarkGoHTTPCPostWithClientTrace(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	client := gohttpc.NewClient(gohttpc.EnableClientTrace(true), gohttpc.WithLogger(logger))
	defer client.Close()

	for b.Loop() {
		req := client.R(http.MethodPost, serverURL)
		req.SetBody(strings.NewReader(randomData))

		resp, err := req.Execute(context.Background())
		if err != nil {
			continue
		}

		if resp.StatusCode != 200 {
			slog.Error(resp.Status)
		}
		_ = resp.Body.Close()
	}
}
