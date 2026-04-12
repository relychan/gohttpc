package benchmark

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hasura/gotel/otelutils"
	"github.com/relychan/gohttpc"
	"resty.dev/v3"
)

// goos: darwin
// goarch: arm64
// pkg: github.com/relychan/gohttpc/example
// cpu: Apple M3 Pro
// BenchmarkHTTPClient/http_get-11         	   39042	     29450 ns/op	    4690 B/op	      54 allocs/op
// BenchmarkHTTPClient/resty_get-11        	   35859	     34325 ns/op	    6903 B/op	      72 allocs/op
// BenchmarkHTTPClient/gohttpc_get-11      	   35144	     33978 ns/op	    9543 B/op	     117 allocs/op
// BenchmarkHTTPClient/http_post-11        	    3519	    336920 ns/op	   53786 B/op	     185 allocs/op
// BenchmarkHTTPClient/resty_post-11       	    2394	    442824 ns/op	 2241756 B/op	     222 allocs/op
// BenchmarkHTTPClient/gohttpc_post-11     	    3493	    370145 ns/op	   57652 B/op	     250 allocs/op
// BenchmarkHTTPClient/gohttpc_post_trace-11    3541	    370773 ns/op	   59462 B/op	     284 allocs/op
func BenchmarkHTTPClient(b *testing.B) {
	server := startHTTPServer()
	defer server.Close()

	randomData := strings.Repeat("1234567890", 100000)

	b.Run("http_get", func(b *testing.B) {
		client := http.DefaultClient

		for b.Loop() {
			resp, err := client.Get(server.URL)
			if err != nil {
				b.Fatal(err)
			}

			if resp.StatusCode != 200 {
				slog.Error(resp.Status)
			}
			_ = resp.Body.Close()
		}
	})

	b.Run("resty_get", func(b *testing.B) {
		client := resty.New()

		defer func() {
			_ = client.Close()
		}()

		for b.Loop() {
			resp, err := client.R().Get(server.URL)
			if err != nil {
				b.Fatal(err)
			}
			defer resp.Body.Close()

			if resp.StatusCode() != 200 {
				slog.Error(resp.RawResponse.Status)
			}
		}
	})

	b.Run("gohttpc_get", func(b *testing.B) {
		client := gohttpc.NewClient()
		defer func() {
			_ = client.Close()
		}()

		logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))
		ctx := otelutils.NewContextWithLogger(context.Background(), logger)

		for b.Loop() {
			resp, err := client.R(http.MethodGet, server.URL).
				Execute(ctx)
			if err != nil {
				b.Fatal(err)
			}

			gohttpc.CloseResponse(resp)

			if resp.StatusCode != 200 {
				slog.Error(resp.Status)
			}
		}
	})

	b.Run("http_post", func(b *testing.B) {
		client := http.DefaultClient

		for b.Loop() {
			resp, err := client.Post(server.URL, "application/json", strings.NewReader(randomData))
			if err != nil {
				b.Fatal(err)
			}

			_, err = io.Copy(io.Discard, resp.Body)
			if err != nil {
				slog.Error("failed to read response", "error", err)
			}

			_ = resp.Body.Close()

			if resp.StatusCode != 200 {
				slog.Error(resp.Status)
			}
		}
	})

	b.Run("resty_post", func(b *testing.B) {
		client := resty.New()
		defer func() {
			_ = client.Close()
		}()

		for b.Loop() {
			resp, err := client.R().SetBody(randomData).Post(server.URL)
			if err != nil {
				b.Fatal(err)
			}
			defer resp.Body.Close()

			if resp.StatusCode() != 200 {
				slog.Error(resp.RawResponse.Status)
			}
		}
	})

	b.Run("gohttpc_post", func(b *testing.B) {
		logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))
		client := gohttpc.NewClient()
		defer func() {
			_ = client.Close()
		}()

		ctx := otelutils.NewContextWithLogger(context.Background(), logger)

		for b.Loop() {
			req := client.R(http.MethodPost, server.URL)
			req.SetBody(strings.NewReader(randomData))
			resp, err := req.Execute(ctx)
			if err != nil {
				b.Fatal(err)
			}

			_, err = io.Copy(io.Discard, resp.Body)
			if err != nil {
				slog.Error(err.Error())
			}

			if resp.StatusCode != 200 {
				slog.Error(resp.Status)
			}

			_ = resp.Body.Close()
		}
	})

	b.Run("gohttpc_post_trace", func(b *testing.B) {
		logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))

		client := gohttpc.NewClient(gohttpc.EnableClientTrace(true))
		defer func() {
			_ = client.Close()
		}()

		ctx := otelutils.NewContextWithLogger(context.Background(), logger)

		for b.Loop() {
			req := client.R(http.MethodPost, server.URL)
			req.SetBody(strings.NewReader(randomData))

			resp, err := req.Execute(ctx)
			if err != nil {
				b.Fatal(err)
			}

			_, err = io.Copy(io.Discard, resp.Body)
			if err != nil {
				slog.Error(err.Error())
			}

			if resp.StatusCode != 200 {
				slog.Error(resp.Status)
			}
			_ = resp.Body.Close()
		}
	})
}

func startHTTPServer() *httptest.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			w.WriteHeader(http.StatusOK)
		case http.MethodPost:
			w.WriteHeader(http.StatusOK)

			_, err := io.Copy(w, r.Body)
			if err != nil {
				slog.Error(err.Error())
			}
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	return httptest.NewServer(mux)
}
