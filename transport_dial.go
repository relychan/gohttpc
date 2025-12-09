package gohttpc

import (
	"context"
	"net"
	"time"

	"github.com/hasura/gotel/otelutils"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.38.0"
)

func transportDialContext(
	dialer *net.Dialer,
) func(context.Context, string, string) (net.Conn, error) {
	return func(ctx context.Context, network string, address string) (net.Conn, error) {
		createdTime := time.Now()

		conn, err := dialer.DialContext(ctx, network, address)
		if err != nil {
			return nil, err
		}

		_, port, _ := otelutils.SplitHostPort(address, "")

		connMetric := &connWithMetric{
			Conn:        conn,
			createdTime: createdTime,
			metricAttrSet: metric.WithAttributeSet(attribute.NewSet(
				semconv.ServerAddress(address),
				semconv.ServerPort(port),
				semconv.NetworkPeerAddress(conn.RemoteAddr().String()),
			)),
		}

		GetHTTPClientMetrics().OpenConnections.Add(ctx, 1, connMetric.metricAttrSet)

		return connMetric, nil
	}
}

// connWithMetric wraps a net.Conn to decrement the counter on close.
type connWithMetric struct {
	net.Conn

	createdTime   time.Time
	metricAttrSet metric.MeasurementOption
}

func (c *connWithMetric) Close() error {
	metrics := GetHTTPClientMetrics()
	metrics.OpenConnections.Add(context.TODO(), -1, c.metricAttrSet)
	metrics.ConnectionDuration.Record(
		context.TODO(),
		time.Since(c.createdTime).Seconds(),
		c.metricAttrSet,
	)

	return c.Conn.Close()
}
