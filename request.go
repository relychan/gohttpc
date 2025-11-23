package gohttpc

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"time"

	backoff "github.com/cenkalti/backoff/v5"
	"github.com/google/uuid"
	"github.com/hasura/gotel/otelutils"
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
	Method string

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
	URL string

	// Header contains the request header fields either received
	// by the server or to be sent by the client.
	//
	// If a server received a request with header lines,
	//
	//	Host: example.com
	//	accept-encoding: gzip, deflate
	//	Accept-Language: en-us
	//	fOO: Bar
	//	foo: two
	//
	// then
	//
	//	Header = map[string][]string{
	//		"Accept-Encoding": {"gzip, deflate"},
	//		"Accept-Language": {"en-us"},
	//		"Foo": {"Bar", "two"},
	//	}
	//
	// For incoming requests, the Host header is promoted to the
	// Request.Host field and removed from the Header map.
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
	Header http.Header

	// Body is the request's body.
	//
	// For client requests, a nil body means the request has no
	// body, such as a GET request. The HTTP Client's Transport
	// is responsible for calling the Close method.
	//
	// For server requests, the Request Body is always non-nil
	// but will return EOF immediately when no body is present.
	// The Server will close the request body. The ServeHTTP
	// Handler does not need to.
	//
	// Body must allow Read to be called concurrently with Close.
	// In particular, calling Close should unblock a Read waiting
	// for input.
	Body io.Reader

	// Timeout is the maximum timeout for the request.
	Timeout time.Duration
	// RetryPolicy is the retry policy for the request.
	Retry *RetryPolicy

	client        *Client
	compressed    bool
	authenticator authscheme.HTTPClientAuthenticator
}

// SetBody handles the HTTP request to the remote server.
func (r *Request) SetBody(body io.Reader) *Request {
	r.Body = body

	return r
}

// Execute handles the HTTP request to the remote server.
func (r *Request) Execute(ctx context.Context) (*Response, error) { //nolint:funlen,maintidx
	if r.Method == "" {
		return nil, ErrRequestMethodRequired
	}

	startTime := time.Now()
	logger := r.getLogger(ctx)
	isDebug := logger.Enabled(ctx, slog.LevelDebug)

	requestLogAttrs := []slog.Attr{
		slog.String("url", r.URL),
		slog.String("method", r.Method),
	}

	var requestBodyStr string

	if isDebug && r.Body != nil && isContentTypeDebuggable(r.Header.Get(httpheader.ContentType)) {
		body, err := io.ReadAll(r.Body)
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

		r.Body = bytes.NewReader(body)
	}

	endpoint, err := goutils.ParseRelativeOrHTTPURL(r.URL)
	if err != nil {
		logger.Error(
			"invalid request url",
			slog.GroupAttrs("request", requestLogAttrs...),
			slog.String("error", err.Error()),
			slog.Float64("latency", time.Since(startTime).Seconds()),
		)

		return nil, err
	}

	spanContext, span := r.client.options.Tracer.Start(
		ctx,
		"request",
		trace.WithSpanKind(trace.SpanKindInternal),
	)

	span.SetAttributes(
		semconv.NetworkProtocolName("http"),
		httpRequestMethodAttr(r.Method),
		semconv.URLFull(r.URL),
	)

	if r.Timeout > 0 {
		span.SetAttributes(attribute.String("http.request.timeout", r.Timeout.String()))
	}

	if requestBodyStr != "" {
		span.SetAttributes(attribute.String("http.request.body", requestBodyStr))
	}

	commonAttrs := []attribute.KeyValue{
		httpRequestMethodAttr(r.Method),
	}

	if endpoint.Host != "" {
		_, port, _ := otelutils.SplitHostPort(endpoint.Host, endpoint.Scheme)
		commonAttrs = newMetricAttributes(r.Method, endpoint, port)
		span.SetAttributes(commonAttrs...)
	}

	requestDurationAttrs := commonAttrs

	defer func() {
		span.End()
		r.client.options.Metrics.RequestDuration.Record(
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

	var resp *Response

	var debugInfo requestDebugInfo

	if r.Retry == nil || r.Retry.Times == 0 {
		if r.Timeout > 0 {
			contextTimeout, cancel := context.WithTimeout(spanContext, r.Timeout)
			defer cancel()

			spanContext = contextTimeout
		}

		resp, debugInfo, err = r.do(spanContext, endpoint, body, 0, logger)
	} else {
		resp, debugInfo, err = r.executeWithRetries(spanContext, endpoint, body, logger)
	}

	var responseLogAttrs []slog.Attr

	requestLogAttrs = append(
		requestLogAttrs,
		otelutils.NewHeaderLogGroupAttrs("headers", debugInfo.RequestHeaders),
	)

	if resp != nil {
		responseLogAttrs = []slog.Attr{
			slog.Int("status", resp.RawResponse.StatusCode),
			otelutils.NewHeaderLogGroupAttrs("headers", debugInfo.ResponseHeaders),
		}

		responseSize := resp.RawResponse.ContentLength
		statusCodeAttr := semconv.HTTPResponseStatusCode(resp.RawResponse.StatusCode)

		span.SetAttributes(statusCodeAttr)

		requestDurationAttrs = append(
			requestDurationAttrs,
			statusCodeAttr,
			semconv.NetworkProtocolVersion(
				fmt.Sprintf(
					"%d.%d",
					resp.RawResponse.ProtoMajor,
					resp.RawResponse.ProtoMinor,
				),
			),
		)

		if endpoint.Host == "" {
			_, port, _ := otelutils.SplitHostPort(
				resp.RawResponse.Request.URL.Host,
				resp.RawResponse.Request.URL.Scheme,
			)

			attrs := []attribute.KeyValue{
				semconv.ServerAddress(resp.RawResponse.Request.URL.Host),
				semconv.ServerPort(port),
				semconv.URLScheme(resp.RawResponse.Request.URL.Scheme),
			}

			requestDurationAttrs = append(requestDurationAttrs, attrs...)
			span.SetAttributes(attrs...)
		}

		if isDebug && !resp.IsBodyRead() && resp.body != nil &&
			isContentTypeDebuggable(resp.Header().Get(httpheader.ContentType)) {
			body, err := resp.ReadBytes()
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

			resp.body = io.NopCloser(bytes.NewReader(body))
			resp.isBodyRead = false
		}

		responseLogAttrs = append(
			responseLogAttrs,
			slog.Int64("size", responseSize),
		)
	}

	if err != nil {
		errMessage := "http request failed"
		if resp != nil {
			errMessage = resp.RawResponse.Status
		}

		logger.Error(
			errMessage,
			slog.GroupAttrs("request", requestLogAttrs...),
			slog.GroupAttrs("response", responseLogAttrs...),
			slog.Any("error", err),
			slog.Float64("latency", time.Since(startTime).Seconds()),
		)

		span.SetStatus(codes.Error, "http request failed")
		span.RecordError(err)

		return resp, err
	}

	logger.Info(
		resp.RawResponse.Status,
		slog.GroupAttrs("request", requestLogAttrs...),
		slog.GroupAttrs("response", responseLogAttrs...),
		slog.Float64("latency", time.Since(startTime).Seconds()),
	)

	return resp, nil
}

func (r *Request) executeWithRetries( //nolint:funlen
	ctx context.Context,
	endpoint *url.URL,
	body io.Reader,
	logger *slog.Logger,
) (*Response, requestDebugInfo, error) {
	var bodySeeker io.ReadSeeker

	var debugInfo requestDebugInfo

	if body != nil {
		bsk, ok := body.(io.ReadSeeker)
		if ok {
			bodySeeker = bsk
		} else {
			bodyBytes, err := io.ReadAll(body)
			if err != nil {
				return nil, debugInfo, err
			}

			bodySeeker = bytes.NewReader(bodyBytes)
		}
	}

	var httpErr *goutils.RFC9457ErrorWithExtensions

	retryCount := 0

	operation := func() (*Response, error) {
		if bodySeeker != nil {
			_, _ = bodySeeker.Seek(0, io.SeekStart)
		}

		resp, di, err := r.do(
			ctx,
			endpoint,
			bodySeeker,
			retryCount,
			logger.With("attempt", retryCount),
		)

		debugInfo = di

		if err == nil {
			return resp, nil
		}

		if resp == nil {
			return nil, backoff.Permanent(err)
		}

		var he goutils.RFC9457ErrorWithExtensions

		ok := errors.As(err, &he)
		if !ok {
			return nil, backoff.Permanent(err)
		}

		httpErr = &he

		if !slices.Contains(r.Retry.GetRetryHTTPStatus(), resp.RawResponse.StatusCode) {
			return resp, backoff.Permanent(err)
		}

		retryCount++

		retryAfter := getRetryAfter(resp.RawResponse)
		if retryAfter > 0 {
			return resp, backoff.RetryAfter(retryAfter)
		}

		return resp, httpErr
	}

	backoffConfig := r.Retry.GetExponentialBackoff()

	retryOptions := []backoff.RetryOption{
		backoff.WithBackOff(backoffConfig),
		backoff.WithMaxTries(r.Retry.Times + 1),
	}

	if r.Timeout > 0 {
		retryOptions = append(retryOptions, backoff.WithMaxElapsedTime(r.Timeout))
	}

	resp, err := backoff.Retry(
		ctx,
		operation,
		retryOptions...,
	)
	if err == nil {
		return resp, debugInfo, nil
	}

	var permanentErr *backoff.PermanentError
	if errors.As(err, &permanentErr) {
		return resp, debugInfo, permanentErr.Err
	}

	if httpErr != nil {
		return resp, debugInfo, httpErr
	}

	return resp, debugInfo, err
}

func (r *Request) compressBody() (io.Reader, error) {
	if r.compressed {
		return r.Body, nil
	}

	encoding := r.Header.Get(httpheader.ContentEncoding)
	if encoding == "" {
		return r.Body, nil
	}

	// should ignore the compression if the encoding isn't supported.
	if !r.client.compressors.IsEncodingSupported(encoding) {
		r.Header.Del(httpheader.ContentEncoding)

		return r.Body, nil
	}

	var buf bytes.Buffer

	_, err := r.client.compressors.Compress(&buf, encoding, r.Body)
	if err != nil {
		return nil, err
	}

	r.compressed = true

	return &buf, nil
}

func (r *Request) do( //nolint:funlen,maintidx,contextcheck
	parentContext context.Context,
	endpoint *url.URL,
	body io.Reader,
	retryCount int,
	logger *slog.Logger,
) (*Response, requestDebugInfo, error) {
	var span HTTPClientTracer

	spanName := r.Method

	if r.client.options.TraceHighCardinalityPath {
		spanName += " " + endpoint.Path
	}

	if r.client.options.ClientTraceEnabled {
		span = startClientTrace(
			parentContext,
			spanName,
			r.client,
			logger,
		)
	} else {
		span = startSimpleClientTrace(
			parentContext,
			spanName,
			r.client,
		)
	}

	if retryCount > 0 {
		span.SetAttributes(semconv.HTTPRequestResendCount(retryCount))
	}

	span.SetAttributes(
		semconv.NetworkProtocolName("http"),
		semconv.UserAgentName(r.client.options.UserAgent),
	)

	ctx := span.Context()

	req, err := r.client.options.CreateRequest(ctx, r, body) //nolint:contextcheck
	if err != nil {
		msg := "failed to create request"

		span.SetAttributes(
			httpRequestMethodAttr(r.Method),
			semconv.URLFull(req.URL.String()),
		)
		span.SetStatus(codes.Error, msg)
		span.RecordError(err)

		debugInfo := r.logRequestAttempt(
			span,
			logger,
			req,
			nil,
			err,
			msg,
		)

		return nil, debugInfo, err
	}

	_, port, _ := otelutils.SplitHostPort(req.URL.Host, req.URL.Scheme)

	commonAttrs := newMetricAttributes(r.Method, req.URL, port)
	span.SetAttributes(commonAttrs...)
	span.SetAttributes(semconv.URLFull(req.URL.String()))

	activeRequestsAttrSet := metric.WithAttributeSet(attribute.NewSet(commonAttrs...))

	r.client.options.Metrics.ActiveRequests.Add( //nolint:contextcheck
		ctx,
		1,
		activeRequestsAttrSet,
	)

	defer func() {
		span.End()
		r.client.options.Metrics.ActiveRequests.Add(
			ctx,
			-1,
			activeRequestsAttrSet,
		)
	}()

	if r.client.options.MetricHighCardinalityPath {
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

	err = r.applyAuth(req)
	if err != nil {
		msg := "failed to authenticate request"

		span.SetStatus(codes.Error, msg)
		span.RecordError(err)

		debugInfo := r.logRequestAttempt(
			span,
			logger,
			req,
			nil,
			err,
			msg,
		)

		return nil, debugInfo, err
	}

	propagator := otel.GetTextMapPropagator()
	propagator.Inject(ctx, propagation.HeaderCarrier(req.Header)) //nolint:contextcheck
	req.Header.Set(httpheader.UserAgent, r.client.options.UserAgent)

	rawResp, err := r.client.options.HTTPClient.Do(req) //nolint:bodyclose
	if err != nil {
		msg := "failed to execute request"
		span.SetStatus(codes.Error, msg)
		span.RecordError(err)

		debugInfo := r.logRequestAttempt(span, logger, req, rawResp, err, msg)

		return nil, debugInfo, err
	}

	statusCodeAttr := semconv.HTTPResponseStatusCode(rawResp.StatusCode)
	commonAttrs = append(commonAttrs, statusCodeAttr)
	commonAttrsSet := metric.WithAttributeSet(attribute.NewSet(commonAttrs...))

	span.SetAttributes(statusCodeAttr)

	if rawResp.Request.ContentLength > 0 {
		r.client.options.Metrics.RequestBodySize.Record( //nolint:contextcheck
			ctx,
			rawResp.Request.ContentLength,
			commonAttrsSet)
		span.SetAttributes(
			semconv.HTTPRequestBodySize(int(rawResp.Request.ContentLength)),
		)
	}

	if rawResp.ContentLength > 0 {
		r.client.options.Metrics.ResponseBodySize.Record( //nolint:contextcheck
			ctx,
			rawResp.ContentLength,
			commonAttrsSet)
		span.SetAttributes(semconv.HTTPResponseBodySize(int(rawResp.ContentLength)))
	}

	remoteAddr := span.RemoteAddress()

	if remoteAddr != "" {
		peerAddress, peerPort, err := otelutils.SplitHostPort(remoteAddr, endpoint.Scheme)
		if err != nil {
			r.client.options.Logger.
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

	resp := &Response{
		RawResponse: rawResp,
		body:        rawResp.Body,
	}

	if resp.body == nil {
		resp.closed = true

		if rawResp.StatusCode >= http.StatusBadRequest {
			span.SetStatus(codes.Error, rawResp.Status)

			debugInfo := r.logRequestAttempt(span, logger, req, rawResp, nil, rawResp.Status)

			return resp, debugInfo, httpErrorFromNoContentResponse(rawResp)
		}

		span.SetStatus(codes.Ok, "")

		debugInfo := r.logRequestAttempt(span, logger, req, rawResp, nil, rawResp.Status)

		return resp, debugInfo, nil
	}

	responseEncoding := resp.RawResponse.Header.Get(httpheader.ContentEncoding)

	if resp.body != nil && responseEncoding != "" {
		decompressedBody, err := r.client.compressors.Decompress(resp.body, responseEncoding)
		if err != nil {
			msg := "failed to decompress response body"
			span.SetStatus(codes.Error, msg)
			span.RecordError(err)

			debugInfo := r.logRequestAttempt(
				span,
				logger,
				req,
				rawResp,
				err,
				rawResp.Status,
			)

			return resp, debugInfo, err
		}

		resp.body = decompressedBody
	}

	if rawResp.StatusCode >= http.StatusBadRequest {
		span.SetStatus(codes.Error, rawResp.Status)

		err := httpErrorFromResponse(resp)
		debugInfo := r.logRequestAttempt(span, logger, req, rawResp, err, rawResp.Status)

		return resp, debugInfo, err
	}

	span.SetStatus(codes.Ok, "")

	debugInfo := r.logRequestAttempt(
		span,
		logger,
		req,
		rawResp,
		err,
		rawResp.Status,
	)

	return resp, debugInfo, nil
}

func (r *Request) logRequestAttempt(
	span HTTPClientTracer,
	logger *slog.Logger,
	req *http.Request,
	resp *http.Response,
	err error,
	message string,
) requestDebugInfo {
	debugInfo := requestDebugInfo{
		RequestHeaders: otelutils.NewTelemetryHeaders(req.Header),
	}

	otelutils.SetSpanHeaderAttributes(span, "http.request.header", debugInfo.RequestHeaders)

	if resp != nil {
		debugInfo.ResponseHeaders = otelutils.NewTelemetryHeaders(resp.Header)

		otelutils.SetSpanHeaderAttributes(span, "http.response.header", debugInfo.ResponseHeaders)
	}

	span.End()

	if !logger.Enabled(req.Context(), slog.LevelDebug) {
		return debugInfo
	}

	totalTime := span.TotalTime()

	requestLogAttrs := []slog.Attr{
		slog.String("url", r.URL),
		slog.String("method", r.Method),
		otelutils.NewHeaderLogGroupAttrs("headers", debugInfo.RequestHeaders),
	}

	logAttrs := []any{
		slog.GroupAttrs("request", requestLogAttrs...),
		slog.Float64("latency", totalTime.Seconds()),
	}

	if err != nil {
		logAttrs = append(logAttrs, slog.Any("error", err))
	}

	if resp != nil {
		responseLogAttrs := []slog.Attr{
			slog.Int("status", resp.StatusCode),
			slog.Int64("size", resp.ContentLength),
			otelutils.NewHeaderLogGroupAttrs("headers", debugInfo.ResponseHeaders),
		}

		logAttrs = append(logAttrs, slog.GroupAttrs("response", responseLogAttrs...))
	}

	logger.Debug(message, logAttrs...)

	return debugInfo
}

func (r *Request) applyAuth(req *http.Request) error {
	authenticator := r.authenticator

	if authenticator == nil {
		authenticator = r.client.options.Authenticator
	}

	if authenticator == nil {
		return nil
	}

	return authenticator.Authenticate(req)
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

	return r.client.options.Logger.
		With(typeAttr, slog.String("request_id", requestID))
}
