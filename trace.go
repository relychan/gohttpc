package gohttpc

import (
	"context"
	"crypto/tls"
	"errors"
	"log/slog"
	"net"
	"net/http/httptrace"
	"net/textproto"
	"net/url"
	"runtime/debug"
	"strings"
	"time"

	"github.com/relychan/goutils/httpheader"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.30.0"
	"go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("gohttpc")

// LogLevelTrace is the constant enum for the TRACE log level.
const LogLevelTrace = slog.Level(-8)

const millisecond = float64(time.Millisecond)

// HTTPClientTracer abstracts an interface to collect traces and metrics data of an HTTP request.
type HTTPClientTracer interface {
	trace.Span

	// Context returns the internal context.
	Context() context.Context
	// TotalTime returns the total time.
	TotalTime() time.Duration
	// RemoteAddress gets the remote address if exists.
	RemoteAddress() string
	// SetMetricAttributes sets common attributes for metrics.
	SetMetricAttributes(attrs []attribute.KeyValue)
}

type simpleClientTrace struct {
	trace.Span

	context     context.Context //nolint:containedctx
	tracer      trace.Tracer
	metrics     *HTTPClientMetrics
	metricAttrs []attribute.KeyValue
	startTime   time.Time
	totalTime   time.Duration
}

var _ HTTPClientTracer = (*simpleClientTrace)(nil)

func startSimpleClientTrace(
	parentContext context.Context,
	name string,
	client *Client,
) *simpleClientTrace {
	t := &simpleClientTrace{
		startTime: time.Now(),
		tracer:    tracer,
		metrics:   client.options.Metrics,
	}

	spanContext, span := t.tracer.Start( //nolint:spancheck
		parentContext,
		name,
		trace.WithSpanKind(trace.SpanKindClient),
	)
	t.context = spanContext
	t.Span = span

	return t //nolint:spancheck
}

// Context returns the internal context.
func (sct *simpleClientTrace) Context() context.Context {
	return sct.context
}

// SetMetricAttributes sets common attributes for metrics.
func (sct *simpleClientTrace) SetMetricAttributes(attrs []attribute.KeyValue) {
	sct.metricAttrs = attrs
}

// RemoteAddress gets the remote address if exists.
func (*simpleClientTrace) RemoteAddress() string {
	return ""
}

// TotalTime gets the start time.
func (sct *simpleClientTrace) TotalTime() time.Duration {
	return sct.totalTime
}

// End the tracer and record metrics.
func (sct *simpleClientTrace) End(options ...trace.SpanEndOption) {
	if sct.totalTime > 0 {
		return
	}

	sct.Span.End(options...)
	sct.totalTime = time.Since(sct.startTime)

	sct.metrics.ServerDuration.Record(
		sct.context,
		sct.totalTime.Seconds(),
		metric.WithAttributeSet(attribute.NewSet(sct.metricAttrs...)),
	)
}

// clientTrace struct maps the [httptrace.ClientTrace] hooks into Fields
// with the same naming for easy understanding. Plus additional insights
// [Request].
type clientTrace struct {
	trace.Span

	context              context.Context //nolint:containedctx
	metrics              *HTTPClientMetrics
	metricAttrs          []attribute.KeyValue
	logger               *slog.Logger
	startTime            time.Time
	getConn              time.Time
	dnsStart             time.Time
	dnsDone              time.Time
	connectDone          time.Time
	tlsHandshakeStart    time.Time
	tlsHandshakeDone     time.Time
	gotConn              time.Time
	gotFirstResponseByte time.Time
	totalTime            time.Duration
	host                 string
	remoteAddr           string
}

var _ HTTPClientTracer = (*clientTrace)(nil)

func startClientTrace(
	ctx context.Context,
	name string,
	client *Client,
	logger *slog.Logger,
) *clientTrace {
	ct := &clientTrace{
		metrics: client.options.Metrics,
		logger:  logger,
	}

	spanContext, span := client.options.Tracer.Start( //nolint:spancheck
		ctx,
		name,
		trace.WithSpanKind(trace.SpanKindClient),
	)
	ct.context = ct.createContext(spanContext)
	ct.Span = span

	return ct //nolint:spancheck
}

// Context returns the internal context.
func (t *clientTrace) Context() context.Context {
	return t.context
}

// SetMetricAttributes sets common attributes for metrics.
func (t *clientTrace) SetMetricAttributes(attrs []attribute.KeyValue) {
	t.metricAttrs = attrs
}

// StartTime returns the start time.
func (t *clientTrace) StartTime() time.Time {
	return t.startTime
}

// TotalTime returns the total time.
func (t *clientTrace) TotalTime() time.Duration {
	return t.totalTime
}

// End the tracer and record metrics.
func (t *clientTrace) End(options ...trace.SpanEndOption) {
	if t.totalTime > 0 {
		return
	}

	requestStartTime := t.StartTime()
	endTime := time.Now()

	span := t.Span

	t.totalTime = endTime.Sub(requestStartTime)

	metricAttrSet := metric.WithAttributeSet(attribute.NewSet(t.metricAttrs...))

	if t.gotFirstResponseByte.IsZero() {
		if !t.gotConn.IsZero() {
			requestStartTime = t.getConn
		}

		t.metrics.ServerDuration.Record(
			t.context,
			endTime.Sub(requestStartTime).Seconds(),
			metricAttrSet,
		)
	} else {
		responseTime := endTime.Sub(t.gotFirstResponseByte)
		span.SetAttributes(
			attribute.Float64(
				"http.stats.response_time_ms",
				float64(responseTime)/millisecond,
			),
		)
	}

	span.End(options...)
}

// RemoteAddress returns the remote address if exists.
func (t *clientTrace) RemoteAddress() string {
	return t.remoteAddr
}

func (t *clientTrace) createContext( //nolint:gocognit,funlen,maintidx
	ctx context.Context,
) context.Context {
	t.startTime = time.Now()
	isTraceLogLevelEnabled := t.logger.Enabled(ctx, LogLevelTrace)

	ct := &httptrace.ClientTrace{
		DNSStart: func(info httptrace.DNSStartInfo) {
			if isTraceLogLevelEnabled {
				t.logger.LogAttrs(
					ctx,
					LogLevelTrace,
					"DNSStart",
					slog.String("host", info.Host),
				)
			}

			// Calculate the total time accordingly when connection is reused,
			// and DNS start and get conn time may be zero if the request is invalid.
			t.host = info.Host
			t.dnsStart = time.Now()
			t.startTime = t.dnsStart
		},
		DNSDone: func(info httptrace.DNSDoneInfo) {
			if isTraceLogLevelEnabled {
				t.logger.LogAttrs(ctx, LogLevelTrace, "DNSDone", slog.Any("info", info))
			}

			t.dnsDone = time.Now()

			if t.dnsStart.IsZero() {
				return
			}

			dnsLookupDuration := time.Since(t.dnsStart)

			t.SetAttributes(
				attribute.Float64(
					"http.stats.dns_lookup_time_ms",
					float64(dnsLookupDuration)/millisecond,
				),
			)

			metricAttrs := []attribute.KeyValue{
				semconv.DNSQuestionName(t.host),
			}

			if info.Err != nil {
				metricAttrs = append(
					metricAttrs,
					semconv.ErrorTypeKey.String(classifyDNSError(info.Err)),
				)
			}

			t.metrics.DNSLookupDuration.Record(
				ctx,
				dnsLookupDuration.Seconds(),
				metric.WithAttributeSet(attribute.NewSet(metricAttrs...)),
			)
		},
		ConnectStart: func(network, addr string) {
			if isTraceLogLevelEnabled {
				t.logger.LogAttrs(
					ctx,
					LogLevelTrace,
					"ConnectStart",
					slog.String("network", network),
					slog.String("address", addr),
				)
			}

			if t.dnsDone.IsZero() {
				t.dnsDone = time.Now()
			}

			if t.dnsStart.IsZero() {
				t.dnsStart = t.dnsDone
			}
		},
		ConnectDone: func(network, addr string, err error) {
			if isTraceLogLevelEnabled {
				t.logger.LogAttrs(
					ctx,
					LogLevelTrace,
					"ConnectDone",
					slog.String("network", network),
					slog.String("address", addr),
					slog.Any("error", err),
				)
			}

			t.connectDone = time.Now()
			tcpConnTime := t.connectDone.Sub(t.dnsDone)

			t.SetAttributes(
				attribute.Float64(
					"http.stats.tcp_connection_time_ms",
					float64(tcpConnTime)/millisecond,
				),
			)
		},
		GetConn: func(hostPort string) {
			if isTraceLogLevelEnabled {
				t.logger.LogAttrs(
					ctx,
					LogLevelTrace,
					"GetConn",
					slog.String("hostPort", hostPort),
				)
			}

			t.getConn = time.Now()
		},
		GotConn: func(ci httptrace.GotConnInfo) {
			if ci.Reused {
				// Calculate the total time accordingly when connection is reused,
				// and DNS start and get conn time may be zero if the request is invalid.
				t.startTime = t.getConn
			}

			if isTraceLogLevelEnabled {
				t.logger.LogAttrs(ctx, LogLevelTrace, "GotConn",
					slog.String("idleTime", ci.IdleTime.String()),
					slog.Bool("reused", ci.Reused),
					slog.Bool("wasIdle", ci.WasIdle),
				)
			}

			t.gotConn = time.Now()
			t.remoteAddr = ci.Conn.RemoteAddr().String()

			connTime := time.Since(t.getConn)

			if ci.WasIdle {
				t.metrics.IdleConnectionDuration.Record(
					ctx,
					ci.IdleTime.Seconds(),
					metric.WithAttributeSet(attribute.NewSet(t.metricAttrs...)),
				)
				t.SetAttributes(
					attribute.Float64(
						"http.stats.idle_connection_time_ms",
						float64(ci.IdleTime)/millisecond,
					),
				)
			}

			t.SetAttributes(
				attribute.Float64(
					"http.stats.connection_acquire_time_ms",
					float64(connTime)/millisecond,
				),
				attribute.Bool("http.stats.is_connection_reused", ci.Reused),
				attribute.Bool("http.stats.is_connection_was_idle", ci.WasIdle),
			)
		},
		GotFirstResponseByte: func() {
			if isTraceLogLevelEnabled {
				t.logger.LogAttrs(ctx, LogLevelTrace, "GotFirstResponseByte")
			}

			t.gotFirstResponseByte = time.Now()

			if !t.gotConn.IsZero() {
				serverTime := t.gotFirstResponseByte.Sub(t.gotConn)
				t.metrics.ServerDuration.Record(
					ctx,
					serverTime.Seconds(),
					metric.WithAttributeSet(attribute.NewSet(t.metricAttrs...)),
				)
				t.SetAttributes(
					attribute.Float64(
						"http.stats.server_time_ms",
						float64(serverTime)/millisecond,
					),
				)
			}
		},
		TLSHandshakeStart: func() {
			if isTraceLogLevelEnabled {
				t.logger.LogAttrs(ctx, LogLevelTrace, "TLSHandshakeStart")
			}

			t.tlsHandshakeStart = time.Now()
		},
		TLSHandshakeDone: func(connState tls.ConnectionState, err error) {
			if isTraceLogLevelEnabled {
				t.logger.LogAttrs(ctx, LogLevelTrace, "TLSHandshakeDone",
					slog.Int("tlsVersion", int(connState.Version)),
					slog.Bool("handshakeComplete", connState.HandshakeComplete),
					slog.Bool("didResume", connState.DidResume),
					slog.Bool("echAccepted", connState.ECHAccepted),
					slog.String("serverName", connState.ServerName),
					slog.Any("error", err),
				)
			}

			t.tlsHandshakeDone = time.Now()

			if t.tlsHandshakeStart.IsZero() {
				return
			}

			tlsHandshakeDuration := time.Since(t.tlsHandshakeStart)

			t.SetAttributes(
				attribute.Float64(
					"http.stats.tls_handshake_time_ms",
					float64(tlsHandshakeDuration)/millisecond,
				),
			)
		},
	}

	if isTraceLogLevelEnabled {
		ct.WroteHeaders = func() {
			t.logger.LogAttrs(ctx, LogLevelTrace, "WroteHeaders")
		}
		ct.Wait100Continue = func() {
			t.logger.LogAttrs(ctx, LogLevelTrace, "Wait100Continue")
		}
		ct.WroteHeaderField = func(key string, value []string) {
			t.logger.LogAttrs(
				ctx,
				LogLevelTrace,
				"WroteHeaderField",
				slog.String("key", key),
				slog.Any("value", value),
			)
		}
		ct.WroteRequest = func(wri httptrace.WroteRequestInfo) {
			t.logger.LogAttrs(
				ctx,
				LogLevelTrace,
				"WroteRequest",
				slog.Any("error", wri.Err),
			)
		}
		ct.Got1xxResponse = func(code int, header textproto.MIMEHeader) error {
			t.logger.LogAttrs(
				ctx,
				LogLevelTrace,
				"Got1xxResponse",
				slog.Int("code", code),
				slog.Any("headers", header),
			)

			return nil
		}
		ct.Got100Continue = func() {
			t.logger.LogAttrs(ctx, LogLevelTrace, "Got100Continue")
		}
	}

	return httptrace.WithClientTrace(ctx, ct)
}

func httpRequestMethodAttr(method string) attribute.KeyValue {
	return attribute.String("http.request.method", method)
}

func newMetricAttributes(method string, endpoint *url.URL, port int) []attribute.KeyValue {
	return []attribute.KeyValue{
		semconv.ServerAddress(endpoint.Host),
		semconv.ServerPort(port),
		semconv.URLScheme(endpoint.Scheme),
		httpRequestMethodAttr(method),
	}
}

func isContentTypeDebuggable(contentType string) bool {
	return strings.HasPrefix(contentType, httpheader.ContentTypeJSON) ||
		strings.HasPrefix(contentType, "text/") ||
		strings.HasPrefix(contentType, "application/xml") ||
		strings.HasPrefix(contentType, "multipart/form-data")
}

func getBuildVersion() string {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}

	if bi.Main.Version != "" {
		return bi.Main.Version
	}

	for _, s := range bi.Settings {
		if s.Key == "vcs.revision" && s.Value != "" {
			return s.Value
		}
	}

	return "unknown"
}

// classifyDNSError classifies a DNS error into a specific error type for metrics.
// Returns "host_not_found" for DNS not found errors, "timeout" for DNS timeout errors,
// and "_OTHER" for all other errors.
func classifyDNSError(err error) string {
	if err == nil {
		return ""
	}

	var dnsError *net.DNSError

	if errors.As(err, &dnsError) {
		switch {
		case dnsError.IsNotFound:
			return "host_not_found"
		case dnsError.IsTimeout:
			return "timeout"
		}
	}

	return "_OTHER"
}
