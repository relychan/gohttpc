package gohttpc

import (
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
)

// HTTPClientMetrics hold metrics of an HTTP client.
type HTTPClientMetrics struct {
	// The duration of how long the connection was previously idle.
	IdleConnectionDuration metric.Float64Histogram
	// The duration of the server for responding to the first byte.
	ServerDuration metric.Float64Histogram
	// Number of active HTTP requests.
	ActiveRequests metric.Int64UpDownCounter
	// Number of outbound HTTP connections that are currently active or idle on the client.
	OpenConnections metric.Int64UpDownCounter
	// Histogram metrics of the request body size.
	RequestBodySize metric.Int64Histogram
	// Histogram metrics of the response body size.
	ResponseBodySize metric.Int64Histogram
	// Duration of HTTP client requests.
	RequestDuration metric.Float64Histogram
	// The duration of the successfully established outbound HTTP connections.
	ConnectionDuration metric.Float64Histogram
}

// NewHTTPClientMetrics creates an HTTPClientMetrics instance from the OpenTelemetry meter.
func NewHTTPClientMetrics( //nolint:funlen
	meter metric.Meter,
	clientTraceEnabled bool,
) (*HTTPClientMetrics, error) {
	metrics := HTTPClientMetrics{
		IdleConnectionDuration: noop.Float64Histogram{},
		ServerDuration:         noop.Float64Histogram{},
		OpenConnections:        noop.Int64UpDownCounter{},
	}

	var err error

	metrics.ActiveRequests, err = meter.Int64UpDownCounter(
		"http.client.active_requests",
		metric.WithDescription("Number of active HTTP requests."),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		return nil, err
	}

	metrics.RequestBodySize, err = meter.Int64Histogram(
		"http.client.request.body.size",
		metric.WithDescription("Size of HTTP client request bodies."),
		metric.WithUnit("By"),
	)
	if err != nil {
		return nil, err
	}

	requestDurationBucketBoundaries := metric.WithExplicitBucketBoundaries(
		0.005,
		0.01,
		0.025,
		0.05,
		0.075,
		0.1,
		0.25,
		0.5,
		0.75,
		1,
		2.5,
		5,
		7.5,
		10,
	)

	metrics.RequestDuration, err = meter.Float64Histogram(
		"http.client.request.duration",
		metric.WithDescription("Duration of HTTP client requests."),
		metric.WithUnit("s"),
		requestDurationBucketBoundaries,
	)
	if err != nil {
		return nil, err
	}

	metrics.ResponseBodySize, err = meter.Int64Histogram(
		"http.client.response.body.size",
		metric.WithDescription("Size of HTTP client response bodies."),
		metric.WithUnit("By"),
	)
	if err != nil {
		return nil, err
	}

	if !clientTraceEnabled {
		return &metrics, nil
	}

	connectionDurationBucketBoundaries := metric.WithExplicitBucketBoundaries(
		0.01,
		0.02,
		0.05,
		0.1,
		0.2,
		0.5,
		1,
		2,
		5,
		10,
		30,
		60,
		120,
		300,
	)

	metrics.IdleConnectionDuration, err = meter.Float64Histogram(
		"http.client.idle_connection.duration",
		metric.WithDescription("The duration of how long the connection was previously idle."),
		metric.WithUnit("s"),
		connectionDurationBucketBoundaries,
	)
	if err != nil {
		return nil, err
	}

	metrics.ServerDuration, err = meter.Float64Histogram(
		"http.client.server.duration",
		metric.WithDescription("The duration of the server for responding to the first byte."),
		metric.WithUnit("s"),
		requestDurationBucketBoundaries,
	)
	if err != nil {
		return nil, err
	}

	metrics.OpenConnections, err = meter.Int64UpDownCounter(
		"http.client.open_connections",
		metric.WithDescription(
			"Number of outbound HTTP connections that are currently active or idle on the client.",
		),
		metric.WithUnit("{connection}"),
	)
	if err != nil {
		return nil, err
	}

	metrics.ConnectionDuration, err = meter.Float64Histogram(
		"http.client.connection.duration",
		metric.WithDescription(
			"The duration of the successfully established outbound HTTP connections.",
		),
		metric.WithUnit("s"),
		connectionDurationBucketBoundaries,
	)
	if err != nil {
		return nil, err
	}

	return &metrics, nil
}

var noopHTTPClientMetrics = HTTPClientMetrics{
	IdleConnectionDuration: noop.Float64Histogram{},
	ServerDuration:         noop.Float64Histogram{},
	ConnectionDuration:     noop.Float64Histogram{},
	ActiveRequests:         noop.Int64UpDownCounter{},
	OpenConnections:        noop.Int64UpDownCounter{},
	RequestBodySize:        noop.Int64Histogram{},
	ResponseBodySize:       noop.Int64Histogram{},
	RequestDuration:        noop.Float64Histogram{},
}
