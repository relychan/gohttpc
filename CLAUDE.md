# CLAUDE.md

## Project Overview

`gohttpc` is a production-grade Go HTTP client library (`github.com/relychan/gohttpc`) that wraps `net/http` with:
- OpenTelemetry tracing and metrics (dual modes: simple and enhanced)
- Pluggable authentication (Basic, HTTP Digest/Bearer, OAuth2)
- Retry/circuit breaker via `failsafe-go`
- Load balancing with health checking (round-robin)
- Request/response compression via `gocompress`
- Config-file-driven setup (YAML/JSON)

**Go version**: 1.26
**License**: Apache 2.0

---

## Build & Test

```bash
# Generate TLS certs required for tests (run once)
./testdata/tls/create-certs.sh

# Run all tests with race detector
go test -v -race -timeout 3m -coverpkg=./... -coverprofile=coverage.out ./...

# Format check
gofmt -d -s .

# Vet
go vet ./...

# Lint (uses .golangci.yml)
golangci-lint run
```

CI runs format → vet → lint → test on every PR and push to `main`.

---

## Code Conventions

### Naming
- Interfaces: descriptive nouns (`HTTPClient`, `LoadBalancer`, `HTTPClientAuthenticator`)
- Config/option structs: `XxxConfig` or `XxxOptions`
- Constructors: `NewXxx(...)`
- Option builder funcs: `WithXxx(...)` returning `ClientOption` or `RequestOption`
- Validate interface satisfaction with blank assignments: `var _ Interface = (*Impl)(nil)`

### Error Handling
- Package-level sentinel errors: `var ErrXxx = errors.New("...")`
- Use standard `errors` package; **do not** use `github.com/pkg/errors` (banned by linter)
- HTTP error bodies follow RFC 9457 JSON format (see `error.go`)

### Testing
- Table-driven tests: `[]struct{ name string; ... }`
- External test packages: `package gohttpc_test`
- Use `httptest.NewServer()` for mock HTTP servers
- Use `t.Setenv()` for environment variables in tests
- Race detector is always on (`-race`)

### Observability
- Tracer: `var clientTracer = otel.Tracer("gohttpc")` (package-level)
- Logging: structured `slog` with `slog.Level(-8)` for trace-level logs
- Metrics: `HTTPClientMetrics` struct holds all OTel instruments; use `noop` types for disabled metrics

---

## Architecture

### Key Design Patterns

**Functional options** — all public APIs accept variadic `...ClientOption` / `...RequestOption`:
```go
client := gohttpc.NewClient(
    gohttpc.WithTimeout(30*time.Second),
    gohttpc.WithAuthenticator(auth),
)
```

**Interface abstraction** — `HTTPClient` and `HTTPClientGetter` decouple client creation from usage; enables load balancing and testing.

**Request execution pipeline** — `Request.Execute(ctx, clientGetter)` handles: validation → compression → tracing → retry → response parsing.

**Dual tracing modes**:
- Default: lightweight spans only
- Enhanced (`ClientTraceEnabled=true` or `HTTP_CLIENT_TRACE_ENABLED=true`): full `net/http/httptrace` with DNS/TLS/connection timing

### Package Layout
| Path | Purpose |
|------|---------|
| `*.go` (root) | Core client, request, response, tracing, metrics, transport |
| `authc/` | Authentication schemes (basic, HTTP, OAuth2) |
| `httpconfig/` | YAML/JSON config parsing for clients, TLS, retry |
| `loadbalancer/` | Load balancing, health checks, round-robin strategy |
| `jsonschema/` | JSON schema generation for config types |
| `example/` | Working examples (simple, load balancer, Prometheus) |
| `benchmark/` | Benchmark tests |
| `testdata/` | TLS certs and test fixtures |

---

## Linter Rules (`.golangci.yml`)
- Max cyclomatic complexity: **25**
- Max line length: **270**
- Banned: `math/rand` (use `math/rand/v2`), `github.com/pkg/errors`
- Test files have relaxed rules (e.g., `errcheck` exclusions)

---

## Adding Features

- New auth schemes: implement `authscheme.AuthScheme` interface, add under `authc/`
- New load balancing strategies: implement `loadbalancer.Strategy`, add under `loadbalancer/`
- New config fields: update structs in `httpconfig/`, regenerate JSON schema in `jsonschema/`
- New metrics: add instrument to `HTTPClientMetrics` in `metrics.go`, provide `noop` default
