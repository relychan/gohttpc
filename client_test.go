package gohttpc_test

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"log"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/relychan/gohttpc"
	"github.com/relychan/gohttpc/authc/authscheme"
	"github.com/relychan/gohttpc/httpconfig"
	"github.com/relychan/goutils"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

func TestClient(t *testing.T) {
	mockState := createMockServer(t)
	defer mockState.Server.Close()

	t.Setenv("BEARER_TOKEN", mockState.APIKey)
	t.Setenv("BASIC_USER", mockState.Username)
	t.Setenv("BASIC_PASSWORD", mockState.Password)

	testCases := []struct {
		Endpoint   string
		ConfigPath string
	}{
		{
			Endpoint:   "/auth/api-key",
			ConfigPath: "testdata/apiKey.yaml",
		},
		{
			Endpoint:   "/auth/basic",
			ConfigPath: "testdata/basic.yaml",
		},
	}

	clientMetrics, err := gohttpc.NewHTTPClientMetrics(otel.Meter("test"), false)
	if err != nil {
		t.Fatal(err)
	}

	gohttpc.SetHTTPClientMetrics(clientMetrics)

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	for _, tc := range testCases {
		t.Run(tc.ConfigPath, func(t *testing.T) {
			config, err := goutils.ReadJSONOrYAMLFile[httpconfig.HTTPClientConfig](context.TODO(), tc.ConfigPath)
			if err != nil {
				t.Fatal(err.Error())
			}

			client, err := httpconfig.NewClientFromConfig(
				config,
				gohttpc.WithAuthenticator(nil),
				gohttpc.WithCustomAttributesFunc(func(r *gohttpc.Request) []attribute.KeyValue {
					return nil
				}),
				gohttpc.WithHTTPClient(http.DefaultClient),
				gohttpc.WithLogger(logger),
				gohttpc.WithMetricHighCardinalityPath(true),
				gohttpc.WithTraceHighCardinalityPath(true),
				gohttpc.WithTracer(otel.Tracer("test")),
				gohttpc.EnableClientTrace(true),
				gohttpc.WithGetEnvFunc(authscheme.NewHTTPClientAuthenticatorOptions().GetEnv),
			)
			if err != nil {
				t.Fatal("failed to create client: " + err.Error())
			}
			defer goutils.CatchWarnErrorFunc(client.Close)

			resp, err := client.Clone().R(http.MethodGet, mockState.Server.URL+tc.Endpoint).
				Execute(context.TODO())
			if err != nil {
				t.Fatal("failed to get: " + err.Error())
			}
			defer goutils.CatchWarnErrorFunc(resp.Body.Close)

			if resp.StatusCode != http.StatusOK {
				t.Fatalf("expected HTTP 200, get: %d", resp.StatusCode)
			}
		})
	}
}

type mockServerState struct {
	Server     *httptest.Server
	RetryCount int32
	APIKey     string
	Username   string
	Password   string

	counter atomic.Int32
}

func (mss *mockServerState) Increase() int32 {
	return mss.counter.Add(1)
}

func (mss *mockServerState) GetCounter() int32 {
	return mss.counter.Load()
}

func createMockServer(t *testing.T) *mockServerState {
	t.Helper()

	state := mockServerState{
		APIKey:   rand.Text(),
		Username: rand.Text(),
		Password: rand.Text(),
	}

	mux := http.NewServeMux()

	writeResponse := func(w http.ResponseWriter, body string) {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body))
	}

	mux.HandleFunc("/auth/api-key", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodPost:
			counter := state.Increase()

			if counter < 2 {
				w.WriteHeader(http.StatusServiceUnavailable)

				return
			}

			apiKey := r.Header.Get("x-hasura-admin-secret")
			expectedValue := "Bearer " + state.APIKey
			if apiKey != expectedValue {
				t.Errorf("invalid bearer auth, expected %s, got %s", expectedValue, apiKey)
				t.FailNow()
			}

			writeResponse(w, "OK")
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
	})

	mux.HandleFunc("/auth/basic", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodPost:
			expectedValue := "Basic " + base64.StdEncoding.EncodeToString([]byte(state.Username+":"+state.Password))
			headerValue := r.Header.Get("WWW-Authorization")

			if headerValue != expectedValue {
				t.Errorf("invalid bearer auth, expected %s, got %s", expectedValue, headerValue)
				t.FailNow()
			}

			writeResponse(w, "OK")
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
	})

	server := httptest.NewServer(mux)
	state.Server = server

	return &state
}

// NOTE: Run the script at testdata/tls/create-certs.sh before running TLS tests.

func TestTLS(t *testing.T) {
	server := createMockTLSServer(t, false)
	defer server.Close()

	keyPem, err := os.ReadFile(filepath.Join("testdata/tls/certs", "client.key"))
	if err != nil {
		t.Fatalf("failed to load client key: %s", err)
	}

	keyData := base64.StdEncoding.EncodeToString(keyPem)
	t.Setenv("TLS_KEY_PEM", string(keyData))

	testCases := []struct {
		Endpoint   string
		ConfigPath string
	}{
		{
			Endpoint:   "/auth/hello",
			ConfigPath: "testdata/tls.yaml",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.ConfigPath, func(t *testing.T) {
			config, err := goutils.ReadJSONOrYAMLFile[httpconfig.HTTPClientConfig](context.TODO(), tc.ConfigPath)
			if err != nil {
				t.Fatal(err.Error())
			}

			client, err := httpconfig.NewClientFromConfig(config)
			if err != nil {
				t.Fatal("failed to create client: " + err.Error())
			}
			defer client.Close()

			resp, err := client.R(http.MethodGet, server.URL+tc.Endpoint).
				Execute(context.Background())
			if err != nil {
				t.Fatal("failed to get: " + err.Error())
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Fatalf("expected HTTP 200, get: %d", resp.StatusCode)
			}
		})
	}
}

func TestTLSInsecure(t *testing.T) {
	server := createMockTLSServer(t, true)
	defer server.Close()

	t.Setenv("TLS_INSECURE", "true")

	testCases := []struct {
		Endpoint   string
		ConfigPath string
	}{
		{
			Endpoint:   "/auth/hello",
			ConfigPath: "testdata/insecureTLS.yaml",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.ConfigPath, func(t *testing.T) {
			config, err := goutils.ReadJSONOrYAMLFile[httpconfig.HTTPClientConfig](context.TODO(), tc.ConfigPath)
			if err != nil {
				t.Fatal(err.Error())
			}

			client, err := httpconfig.NewClientFromConfig(config)
			if err != nil {
				t.Fatal("failed to create client: " + err.Error())
			}
			defer client.Close()

			resp, err := client.R(http.MethodGet, server.URL+tc.Endpoint).
				Execute(context.Background())
			if err != nil {
				t.Fatal("failed to get: " + err.Error())
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Fatalf("expected HTTP 200, get: %d", resp.StatusCode)
			}
		})
	}
}

func createMockTLSServer(
	t *testing.T,
	insecure bool,
) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	mux.HandleFunc("/auth/hello", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
	})

	var tlsConfig *tls.Config

	dir := "testdata/tls/certs"

	// load CA certificate file and add it to list of client CAs
	caCertFile, err := os.ReadFile(filepath.Join(dir, "ca.crt"))
	if err != nil {
		log.Fatalf("error reading CA certificate: %v", err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCertFile)

	// Create the TLS Config with the CA pool and enable Client certificate validation
	cert, err := tls.LoadX509KeyPair(
		filepath.Join(dir, "server.pem"),
		filepath.Join(dir, "server.key"),
	)

	tlsConfig = &tls.Config{
		ClientCAs:          caCertPool,
		Certificates:       []tls.Certificate{cert},
		ClientAuth:         tls.RequireAndVerifyClientCert,
		InsecureSkipVerify: insecure,
	}

	if insecure {
		tlsConfig.ClientAuth = tls.RequestClientCert
	}

	server := httptest.NewUnstartedServer(mux)
	server.TLS = tlsConfig
	server.StartTLS()

	return server
}
