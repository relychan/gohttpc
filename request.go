package gohttpc

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"net/http"
	"net/url"
	"time"

	"github.com/failsafe-go/failsafe-go"
	"github.com/failsafe-go/failsafe-go/retrypolicy"
	"github.com/google/uuid"
	"github.com/hasura/gotel/otelutils"
	"github.com/relychan/gocompress"
	"github.com/relychan/gohttpc/authc/authscheme"
	"github.com/relychan/goutils"
	"github.com/relychan/goutils/httpheader"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.30.0"
	"go.opentelemetry.io/otel/trace"
)

// Request represents the details and configuration for an individual HTTP(S)
// request. It encompasses URL, headers, method, body, proxy settings,
// timeouts, and other configurations necessary for customizing the request
// and its execution.
type Request struct {
	// Method specifies the HTTP method (GET, POST, PUT, etc.).
	// For client requests, an empty string means GET.
	method string

	// URL specifies either the URI being requested (for server
	// requests) or the URL to access (for client requests).
	//
	// For server requests, the URL is parsed from the URI
	// supplied on the Request-Line as stored in RequestURI.  For
	// most requests, fields other than Path and RawQuery will be
	// empty. (See RFC 7230, Section 5.3)
	//
	// For client requests, the URL's Host specifies the server to
	// connect to, while the Request's Host field optionally
	// specifies the Host header value to send in the HTTP
	// request.
	url string

	// Body is the request's body.
	//
	// For client requests, a nil body means the request has no
	// body, such as a GET request. The HTTP Client's Transport
	// is responsible for calling the Close method.
	//
	// Body must allow Read to be called concurrently with Close.
	// In particular, calling Close should unblock a Read waiting
	// for input.
	body io.Reader

	// Timeout is the maximum timeout for the request.
	timeout time.Duration

	// RetryPolicy is the retry policy for the request.
	retry         retrypolicy.RetryPolicy[*http.Response]
	authenticator authscheme.HTTPClientAuthenticator
	header        http.Header
	retryAttempts int
	options       *RequestOptions
}

// NewRequest creates a raw request without client options.
func NewRequest(method string, url string, options *RequestOptions) *Request {
	return &Request{
		method:  method,
		url:     url,
		options: options,
	}
}

// Header returns the request header fields to be sent by the client.
//
// HTTP defines that header names are case-insensitive. The
// request parser implements this by using CanonicalHeaderKey,
// making the first character and any characters following a
// hyphen uppercase and the rest lowercase.
//
// For client requests, certain headers such as Content-Length
// and Connection are automatically written when needed and
// values in Header may be ignored. See the documentation
// for the Request.Write method.
func (r *Request) Header() http.Header {
	if r.header == nil {
		r.header = make(http.Header)
	}

	return r.header
}

// Clone creates a new request. The body can be nil if it was already read.
func (r *Request) Clone() *Request {
	newRequest := *r

	if newRequest.header != nil {
		newRequest.header = maps.Clone(r.header)
	}

	return &newRequest
}

// URL returns the request URL.
func (r *Request) URL() string {
	return r.url
}

// SetURL sets the request URL.
func (r *Request) SetURL(value string) *Request {
	r.url = value

	return r
}

// Method returns the request method.
func (r *Request) Method() string {
	return r.method
}

// SetMethod sets the request method.
func (r *Request) SetMethod(method string) *Request {
	r.method = method

	return r
}

// Timeout returns the request timeout.
func (r *Request) Timeout() time.Duration {
	return r.timeout
}

// SetTimeout sets the request timeout.
func (r *Request) SetTimeout(timeout time.Duration) *Request {
	r.timeout = timeout

	return r
}

// Body returns the request body.
func (r *Request) Body() io.Reader {
	return r.body
}

// SetBody sets the request body.
func (r *Request) SetBody(body io.Reader) *Request {
	r.body = body

	return r
}

// Retry returns the retry policy.
func (r *Request) Retry() retrypolicy.RetryPolicy[*http.Response] {
	return r.retry
}

// SetRetry sets the retry policy.
func (r *Request) SetRetry(retry retrypolicy.RetryPolicy[*http.Response]) *Request {
	r.retry = retry

	return r
}

// Authenticator returns the HTTP client authenticator.
func (r *Request) Authenticator() authscheme.HTTPClientAuthenticator {
	return r.authenticator
}

// SetAuthenticator sets the HTTP authenticator.
func (r *Request) SetAuthenticator(authenticator authscheme.HTTPClientAuthenticator) *Request {
	r.authenticator = authenticator

	return r
}

// Execute handles the HTTP request to the remote server.
func (r *Request) Execute( //nolint:gocognit,funlen,maintidx
	ctx context.Context,
	client HTTPClientGetter,
) (*http.Response, error) {
	if r.method == "" {
		return nil, ErrRequestMethodRequired
	}

	startTime := time.Now()
	logger := r.getLogger(ctx)
	isDebug := logger.Enabled(ctx, slog.LevelDebug)

	requestLogAttrs := make([]slog.Attr, 2, 5)
	requestLogAttrs = append(requestLogAttrs, slog.String("method", r.method))

	var requestBodyStr string

	if isDebug && r.body != nil && isContentTypeDebuggable(r.Header().Get(httpheader.ContentType)) {
		body, err := io.ReadAll(r.body)
		if err != nil {
			logger.Error(
				"failed to read request body",
				slog.String("error", err.Error()),
				slog.Float64("latency", time.Since(startTime).Seconds()),
				slog.GroupAttrs("request", requestLogAttrs...),
			)

			return nil, err
		}

		requestBodyStr = string(body)
		requestLogAttrs = append(
			requestLogAttrs,
			slog.Int("size", len(requestBodyStr)),
			slog.String("body", requestBodyStr),
		)

		r.body = bytes.NewReader(body)
	}

	endpoint, err := goutils.ParseRelativeOrHTTPURL(r.URL())
	if err != nil {
		requestLogAttrs = append(requestLogAttrs, slog.String("url", r.url))
		logger.Error(
			"invalid request url",
			slog.GroupAttrs("request", requestLogAttrs...),
			slog.String("error", err.Error()),
			slog.Float64("latency", time.Since(startTime).Seconds()),
		)

		return nil, err
	}

	spanContext, span := r.options.Tracer.Start(
		ctx,
		"request",
		trace.WithSpanKind(trace.SpanKindInternal),
	)

	span.SetAttributes(httpRequestMethodAttr(r.method))

	if requestBodyStr != "" {
		span.SetAttributes(attribute.String("http.request.body", requestBodyStr))
	}

	commonAttrs := []attribute.KeyValue{
		httpRequestMethodAttr(r.method),
	}

	// the request URL may not be a full URI.
	if endpoint.Host != "" {
		_, port, _ := otelutils.SplitHostPort(endpoint.Host, endpoint.Scheme)
		commonAttrs = newMetricAttributes(r.method, endpoint, port)
		span.SetAttributes(commonAttrs...)
		span.SetAttributes(semconv.URLFull(r.url))
		requestLogAttrs = append(requestLogAttrs, slog.String("url", r.url))
	} else {
		requestLogAttrs = append(requestLogAttrs, slog.String("request_path", r.url))
		span.SetAttributes(semconv.URLPath(r.url))
	}

	requestDurationAttrs := commonAttrs

	defer func() {
		span.End()
		r.options.RequestMetrics.RequestDuration.Record(
			ctx,
			time.Since(startTime).Seconds(),
			metric.WithAttributeSet(attribute.NewSet(requestDurationAttrs...)),
		)
	}()

	body, err := r.compressBody()
	if err != nil {
		msg := "failed to compress request body"
		logger.Error(
			msg,
			slog.GroupAttrs("request", requestLogAttrs...),
			slog.String("error", err.Error()),
			slog.Float64("latency", time.Since(startTime).Seconds()),
		)

		span.SetStatus(codes.Error, msg)
		span.RecordError(err)

		return nil, err
	}

	var resp *http.Response

	timeout := r.getTimeout()
	if timeout > 0 {
		span.SetAttributes(attribute.String("http.request.timeout", timeout.String()))

		contextTimeout, cancel := context.WithTimeout(spanContext, timeout)
		defer cancel()

		spanContext = contextTimeout
	}

	if r.getRetryPolicy() == nil {
		resp, err = r.doRequest(spanContext, client, endpoint, body, logger)
	} else {
		resp, err = r.executeWithRetries(spanContext, client, endpoint, body, logger)
	}

	responseLogAttrs := make([]slog.Attr, 0, 4)

	if resp != nil {
		if endpoint.Host == "" {
			requestURL := resp.Request.URL.String()
			requestLogAttrs = append(requestLogAttrs, slog.String("url", requestURL))
			span.SetAttributes(semconv.URLFull(requestURL))
		}

		if r.options.IsTraceRequestHeadersEnabled() {
			requestHeaders := otelutils.NewTelemetryHeaders(
				resp.Request.Header,
				r.options.AllowedTraceRequestHeaders...,
			)
			otelutils.SetSpanHeaderAttributes(span, "http.request.header", requestHeaders)
			requestLogAttrs = append(
				requestLogAttrs,
				otelutils.NewHeaderLogGroupAttrs("headers", requestHeaders),
			)
		}

		if r.options.IsTraceResponseHeadersEnabled() {
			responseHeaders := otelutils.NewTelemetryHeaders(
				resp.Header,
				r.options.AllowedTraceResponseHeaders...,
			)
			otelutils.SetSpanHeaderAttributes(span, "http.response.header", responseHeaders)
			responseLogAttrs = append(
				responseLogAttrs,
				slog.Int("status", resp.StatusCode),
				otelutils.NewHeaderLogGroupAttrs("headers", responseHeaders),
			)
		}

		responseSize := resp.ContentLength
		statusCodeAttr := semconv.HTTPResponseStatusCode(resp.StatusCode)

		span.SetAttributes(statusCodeAttr)

		requestDurationAttrs = append(
			requestDurationAttrs,
			statusCodeAttr,
			semconv.NetworkProtocolVersion(
				fmt.Sprintf(
					"%d.%d",
					resp.ProtoMajor,
					resp.ProtoMinor,
				),
			),
		)

		if endpoint.Host == "" {
			_, port, _ := otelutils.SplitHostPort(
				resp.Request.URL.Host,
				resp.Request.URL.Scheme,
			)

			attrs := []attribute.KeyValue{
				semconv.ServerAddress(resp.Request.URL.Host),
				semconv.ServerPort(port),
				semconv.URLScheme(resp.Request.URL.Scheme),
			}

			requestDurationAttrs = append(requestDurationAttrs, attrs...)
			span.SetAttributes(attrs...)
		}

		if isDebug && resp.Body != nil &&
			isContentTypeDebuggable(resp.Header.Get(httpheader.ContentType)) {
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				logger.Error(
					"failed to read response body",
					slog.GroupAttrs("request", requestLogAttrs...),
					slog.GroupAttrs("response", responseLogAttrs...),
					slog.String("error", err.Error()),
					slog.Float64("latency", time.Since(startTime).Seconds()),
				)

				return nil, err
			}

			respBodyString := string(body)
			responseLogAttrs = append(
				responseLogAttrs,
				slog.String("body", respBodyString),
			)
			responseSize = int64(len(respBodyString))
			span.SetAttributes(attribute.String("http.response.body", respBodyString))

			resp.Body = io.NopCloser(bytes.NewReader(body))
		}

		responseLogAttrs = append(
			responseLogAttrs,
			slog.Int64("size", responseSize),
		)
	}

	if err != nil {
		errMessage := "http request failed"
		if resp != nil {
			errMessage = resp.Status
		}

		logger.Error(
			errMessage,
			slog.GroupAttrs("request", requestLogAttrs...),
			slog.GroupAttrs("response", responseLogAttrs...),
			slog.Any("error", err),
			slog.Float64("latency", time.Since(startTime).Seconds()),
		)

		span.SetStatus(codes.Error, errMessage)
		span.RecordError(err)

		return resp, err
	}

	logger.Info(
		resp.Status,
		slog.GroupAttrs("request", requestLogAttrs...),
		slog.GroupAttrs("response", responseLogAttrs...),
		slog.Float64("latency", time.Since(startTime).Seconds()),
	)
	span.SetStatus(codes.Ok, "")

	return resp, nil
}

func (r *Request) executeWithRetries(
	ctx context.Context,
	client HTTPClientGetter,
	endpoint *url.URL,
	body io.Reader,
	logger *slog.Logger,
) (*http.Response, error) {
	var bodySeeker io.ReadSeeker

	if body != nil {
		bsk, ok := body.(io.ReadSeeker)
		if ok {
			bodySeeker = bsk
		} else {
			bodyBytes, err := io.ReadAll(body)
			if err != nil {
				return nil, err
			}

			bodySeeker = bytes.NewReader(bodyBytes)
		}
	}

	operation := func() (*http.Response, error) {
		if bodySeeker != nil {
			_, _ = bodySeeker.Seek(0, io.SeekStart)
		}

		resp, err := r.doRequest(
			ctx,
			client,
			endpoint,
			bodySeeker,
			logger.With("attempt", r.retryAttempts),
		)
		if err != nil {
			r.retryAttempts++
		}

		return resp, nil
	}

	return failsafe.With(r.getRetryPolicy()).Get(operation)
}

func (r *Request) compressBody() (io.Reader, error) {
	body := r.body
	r.body = nil

	// Optimization: check r.header directly to avoid initialization if no headers were set
	if body == nil || len(r.header) == 0 {
		return body, nil
	}

	encoding := r.Header().Get(httpheader.ContentEncoding)
	if encoding == "" {
		return body, nil
	}

	// should ignore the compression if the encoding isn't supported.
	if !gocompress.DefaultCompressor.IsEncodingSupported(encoding) {
		r.Header().Del(httpheader.ContentEncoding)

		return body, nil
	}

	var buf bytes.Buffer

	_, err := gocompress.DefaultCompressor.Compress(&buf, encoding, body)
	if err != nil {
		return nil, err
	}

	return &buf, nil
}

func (r *Request) doRequest( //nolint:funlen,maintidx,contextcheck
	parentContext context.Context,
	clientGetter HTTPClientGetter,
	endpoint *url.URL,
	body io.Reader,
	logger *slog.Logger,
) (*http.Response, error) {
	client, err := clientGetter.HTTPClient()
	if err != nil {
		return nil, err
	}

	var span HTTPClientTracer

	spanName := r.method

	if r.options.TraceHighCardinalityPath {
		spanName += " " + endpoint.Path
	}

	if r.options.ClientTraceEnabled {
		span = startClientTrace(
			parentContext,
			spanName,
			r.options.Tracer,
			r.options.RequestMetrics,
			logger,
		)
	} else {
		span = startSimpleClientTrace(
			parentContext,
			spanName,
			r.options.Tracer,
			r.options.RequestMetrics,
		)
	}

	if r.retryAttempts > 0 {
		span.SetAttributes(semconv.HTTPRequestResendCount(r.retryAttempts))
	}

	ctx := span.Context()

	req, err := client.NewRequest(ctx, r.method, r.url, body) //nolint:contextcheck
	if err != nil {
		msg := "failed to create request"

		span.SetAttributes(
			httpRequestMethodAttr(r.method),
			semconv.URLFull(req.URL.String()),
		)
		span.SetStatus(codes.Error, msg)
		span.RecordError(err)

		r.logRequestAttempt(
			span,
			logger,
			req,
			nil,
			err,
			msg,
		)

		return nil, err
	}

	_, port, _ := otelutils.SplitHostPort(req.URL.Host, req.URL.Scheme)

	commonAttrs := newMetricAttributes(r.method, req.URL, port)
	span.SetAttributes(commonAttrs...)
	span.SetAttributes(semconv.URLFull(req.URL.String()))

	activeRequestsAttrSet := metric.WithAttributeSet(attribute.NewSet(commonAttrs...))

	r.options.RequestMetrics.ActiveRequests.Add( //nolint:contextcheck
		ctx,
		1,
		activeRequestsAttrSet,
	)

	defer func() {
		span.End()
		r.options.RequestMetrics.ActiveRequests.Add(
			ctx,
			-1,
			activeRequestsAttrSet,
		)
	}()

	if r.options.MetricHighCardinalityPath {
		commonAttrs = append(
			commonAttrs,
			semconv.URLPath(req.URL.Path),
		)
	}

	protocolVersionAttr := semconv.NetworkProtocolVersion(
		fmt.Sprintf(
			"%d.%d",
			req.ProtoMajor,
			req.ProtoMinor,
		),
	)
	commonAttrs = append(commonAttrs, protocolVersionAttr)

	span.SetAttributes(protocolVersionAttr)
	span.SetMetricAttributes(commonAttrs)

	maps.Copy(req.Header, r.header)

	err = r.applyAuth(req)
	if err != nil {
		msg := "failed to authenticate request"

		span.SetStatus(codes.Error, msg)
		span.RecordError(err)

		r.logRequestAttempt(
			span,
			logger,
			req,
			nil,
			err,
			msg,
		)

		return nil, err
	}

	propagator := otel.GetTextMapPropagator()
	propagator.Inject(ctx, propagation.HeaderCarrier(req.Header)) //nolint:contextcheck
	req.Header.Set(httpheader.UserAgent, r.options.UserAgent)

	rawResp, err := client.Do(req)
	if err != nil {
		msg := "failed to execute request"
		span.SetStatus(codes.Error, msg)
		span.RecordError(err)

		r.logRequestAttempt(span, logger, req, rawResp, err, msg)

		return nil, err
	}

	statusCodeAttr := semconv.HTTPResponseStatusCode(rawResp.StatusCode)
	commonAttrs = append(commonAttrs, statusCodeAttr)
	commonAttrsSet := metric.WithAttributeSet(attribute.NewSet(commonAttrs...))

	span.SetAttributes(statusCodeAttr)

	if rawResp.Request.ContentLength > 0 {
		r.options.RequestMetrics.RequestBodySize.Record( //nolint:contextcheck
			ctx,
			rawResp.Request.ContentLength,
			commonAttrsSet)
		span.SetAttributes(
			semconv.HTTPRequestBodySize(int(rawResp.Request.ContentLength)),
		)
	}

	if rawResp.ContentLength > 0 {
		r.options.RequestMetrics.ResponseBodySize.Record( //nolint:contextcheck
			ctx,
			rawResp.ContentLength,
			commonAttrsSet)
		span.SetAttributes(semconv.HTTPResponseBodySize(int(rawResp.ContentLength)))
	}

	remoteAddr := span.RemoteAddress()

	if remoteAddr != "" {
		peerAddress, peerPort, err := otelutils.SplitHostPort(remoteAddr, endpoint.Scheme)
		if err != nil {
			r.options.Logger.
				Warn(
					"failed to split hostname and port from remote address",
					slog.String("remote_addr", remoteAddr),
					slog.String("error", err.Error()),
				)
		}

		if peerAddress != "" {
			span.SetAttributes(semconv.NetworkPeerAddress(peerAddress))

			if peerPort > 0 {
				span.SetAttributes(semconv.NetworkPeerPort(peerPort))
			}
		}
	}

	if rawResp.Body == nil || rawResp.Body == http.NoBody {
		if rawResp.StatusCode >= http.StatusBadRequest {
			span.SetStatus(codes.Error, rawResp.Status)

			r.logRequestAttempt(span, logger, req, rawResp, nil, rawResp.Status)

			return rawResp, httpErrorFromNoContentResponse(rawResp)
		}

		span.SetStatus(codes.Ok, "")

		r.logRequestAttempt(span, logger, req, rawResp, nil, rawResp.Status)

		return rawResp, nil
	}

	responseEncoding := rawResp.Header.Get(httpheader.ContentEncoding)

	if rawResp.Body != nil && responseEncoding != "" {
		decompressedBody, err := gocompress.DefaultCompressor.Decompress(
			rawResp.Body,
			responseEncoding,
		)
		if err != nil {
			goutils.CatchWarnErrorFunc(rawResp.Body.Close)

			msg := "failed to decompress response body"
			span.SetStatus(codes.Error, msg)
			span.RecordError(err)

			r.logRequestAttempt(
				span,
				logger,
				req,
				rawResp,
				err,
				rawResp.Status,
			)

			return rawResp, err
		}

		rawResp.Body = decompressedBody
	}

	if rawResp.StatusCode >= http.StatusBadRequest {
		span.SetStatus(codes.Error, rawResp.Status)

		err := httpErrorFromResponse(rawResp)
		goutils.CatchWarnErrorFunc(rawResp.Body.Close)
		r.logRequestAttempt(span, logger, req, rawResp, err, rawResp.Status)

		return rawResp, err
	}

	span.SetStatus(codes.Ok, "")

	r.logRequestAttempt(
		span,
		logger,
		req,
		rawResp,
		err,
		rawResp.Status,
	)

	return rawResp, nil
}

func (r *Request) logRequestAttempt(
	span HTTPClientTracer,
	logger *slog.Logger,
	req *http.Request,
	resp *http.Response,
	err error,
	message string,
) {
	defer span.End()

	if !logger.Enabled(req.Context(), slog.LevelDebug) {
		return
	}

	requestHeaders := otelutils.NewTelemetryHeaders(req.Header)
	otelutils.SetSpanHeaderAttributes(span, "http.request.header", requestHeaders)

	totalTime := span.TotalTime()

	requestLogAttrs := []slog.Attr{
		slog.String("url", r.url),
		slog.String("method", r.method),
		otelutils.NewHeaderLogGroupAttrs("headers", requestHeaders),
	}

	logAttrs := []any{
		slog.GroupAttrs("request", requestLogAttrs...),
		slog.Float64("latency", totalTime.Seconds()),
	}

	if err != nil {
		logAttrs = append(logAttrs, slog.Any("error", err))
	}

	if resp != nil {
		responseHeaders := otelutils.NewTelemetryHeaders(resp.Header)

		otelutils.SetSpanHeaderAttributes(span, "http.response.header", responseHeaders)

		responseLogAttrs := []slog.Attr{
			slog.Int("status", resp.StatusCode),
			slog.Int64("size", resp.ContentLength),
			otelutils.NewHeaderLogGroupAttrs("headers", responseHeaders),
		}

		logAttrs = append(logAttrs, slog.GroupAttrs("response", responseLogAttrs...))
	}

	logger.Debug(message, logAttrs...)
}

func (r *Request) applyAuth(req *http.Request) error {
	authenticator := r.authenticator

	if authenticator == nil {
		authenticator = r.options.Authenticator
	}

	if authenticator == nil {
		return nil
	}

	return authenticator.Authenticate(req)
}

func (r *Request) getRetryPolicy() retrypolicy.RetryPolicy[*http.Response] {
	if r.retry != nil {
		return r.retry
	}

	return r.options.Retry
}

func (r *Request) getTimeout() time.Duration {
	if r.timeout > 0 {
		return r.timeout
	}

	return r.options.Timeout
}

func (r *Request) getLogger(ctx context.Context) *slog.Logger {
	typeAttr := slog.String("type", "http-client")

	value := ctx.Value(otelutils.LoggerContextKey)
	if value != nil {
		if logger, ok := value.(*slog.Logger); ok {
			return logger.With(typeAttr)
		}
	}

	var requestID string

	spanContext := trace.SpanContextFromContext(ctx)
	if spanContext.HasTraceID() {
		requestID = spanContext.TraceID().String()
	} else {
		requestID = uuid.NewString()
	}

	return r.options.Logger.
		With(typeAttr, slog.String("request_id", requestID))
}

// RequestWithClient embeds the [Request] with an [HTTPClient] to make the Execute method shorter.
type RequestWithClient struct {
	*Request

	client HTTPClientGetter
}

// NewRequestWithClient creates a new [RequestWithClient] instance.
func NewRequestWithClient(req *Request, client HTTPClientGetter) *RequestWithClient {
	return &RequestWithClient{
		Request: req,
		client:  client,
	}
}

// Execute handles the HTTP request to the remote server.
func (rwc *RequestWithClient) Execute(ctx context.Context) (*http.Response, error) {
	return rwc.Request.Execute(ctx, rwc.client)
}
