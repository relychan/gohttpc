package httpconfig

import (
	"context"
	"net"

	"github.com/hasura/gotel/otelutils"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
)

func transportDialContext(
	dialer *net.Dialer,
	openConnMetric metric.Int64UpDownCounter,
) func(context.Context, string, string) (net.Conn, error) {
	return func(ctx context.Context, network string, address string) (net.Conn, error) {
		conn, err := dialer.DialContext(ctx, network, address)
		if err != nil {
			return nil, err
		}

		_, port, _ := otelutils.SplitHostPort(address, "")

		connMetric := &connWithMetric{
			Conn:           conn,
			openConnMetric: openConnMetric,
			metricAttrSet: metric.WithAttributeSet(attribute.NewSet(
				semconv.ServerAddress(address),
				semconv.ServerPort(port),
				semconv.NetworkPeerAddress(conn.RemoteAddr().String()),
			)),
		}

		connMetric.openConnMetric.Add(ctx, 1, connMetric.metricAttrSet)

		return connMetric, nil
	}
}

// connWithMetric wraps a net.Conn to decrement the counter on close.
type connWithMetric struct {
	net.Conn

	openConnMetric metric.Int64UpDownCounter
	metricAttrSet  metric.MeasurementOption
}

func (c *connWithMetric) Close() error {
	c.openConnMetric.Add(context.TODO(), -1, c.metricAttrSet)

	return c.Conn.Close()
}
