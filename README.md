# gohttpc

gohttpc is a Go HTTP client library that wraps Go's standard `http.Client` while adding observability, authentication, and configuration management features commonly needed in production HTTP clients:

- **Reusable configurations** - HTTP client settings via `httpconfig` package
- **Semantic telemetry** - Built-in OpenTelemetry tracing and metrics support
- **Authentication schemes** - Support for basic auth, OAuth2, and other auth methods following OpenAPI 3 spec
- **Request/response wrappers** - Enhanced `Request` and `Response` types with additional functionality
- **Compression support** - Integration with `gocompress` for request/response compression
- **Retry mechanisms** - Built-in retry logic with backoff strategies

## Observability

### Distributed Tracing

The library provides two tracing modes:

#### Simple Client Trace (default):

- Basic span creation and timing
- Minimal overhead

#### Enhanced Client Trace (when ClientTraceEnabled=true):

- Detailed HTTP lifecycle tracking using Go's `httptrace` package.
- Captures granular timing for:
  - DNS lookup
  - TCP connection establishment
  - TLS handshake
  - Connection reuse/idle time
  - Time to first response byte
  - Response reading time

### Metrics

> [!NOTE]
> Prometheus may replace dots in the metric name with underscores.

#### Core Metrics (Always Available)

| Metric                           | Type      | Description                                     |
| -------------------------------- | --------- | ----------------------------------------------- |
| `dns.lookup.duration`            | Histogram | Measures the time taken to perform a DNS lookup |
| `http.client.active_requests`    | Gauge     | Number of active HTTP requests                  |
| `http.client.request.duration`   | Histogram | Total duration of HTTP requests                 |
| `http.client.server.duration`    | Histogram | Server processing time (time to first byte)     |
| `http.client.request.body.size`  | Histogram | Size of request bodies in bytes                 |
| `http.client.response.body.size` | Histogram | Size of response bodies in bytes                |

#### Enhanced Metrics (When ClientTraceEnabled=true)

| Metric                                 | Type      | Description                                   |
| -------------------------------------- | --------- | --------------------------------------------- |
| `http.client.open_connections`         | Gauge     | Number of active or idle outbound connections |
| `http.client.connection.duration`      | Histogram | Duration to establish outbound connections    |
| `http.client.idle_connection.duration` | Histogram | How long connections were idle before reuse   |

