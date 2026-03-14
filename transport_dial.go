// Copyright 2026 RelyChan Pte. Ltd
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package gohttpc

import (
	"context"
	"net"
	"time"

	"github.com/hasura/gotel/otelutils"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
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
