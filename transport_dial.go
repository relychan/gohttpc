package gohttpc

import (
	"context"
	"net"
	"time"

	"github.com/hasura/gotel/otelutils"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
)

func transportDialContext(
	dialer *net.Dialer,
	metrics *HTTPClientMetrics,
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
			metrics:     metrics,
			createdTime: createdTime,
			metricAttrSet: metric.WithAttributeSet(attribute.NewSet(
				semconv.ServerAddress(address),
				semconv.ServerPort(port),
				semconv.NetworkPeerAddress(conn.RemoteAddr().String()),
			)),
		}

		connMetric.metrics.OpenConnections.Add(ctx, 1, connMetric.metricAttrSet)

		return connMetric, nil
	}
}

// connWithMetric wraps a net.Conn to decrement the counter on close.
type connWithMetric struct {
	net.Conn

	createdTime   time.Time
	metrics       *HTTPClientMetrics
	metricAttrSet metric.MeasurementOption
}

func (c *connWithMetric) Close() error {
	c.metrics.OpenConnections.Add(context.TODO(), -1, c.metricAttrSet)
	c.metrics.ConnectionDuration.Record(
		context.TODO(),
		time.Since(c.createdTime).Seconds(),
		c.metricAttrSet,
	)

	return c.Conn.Close()
}
