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
	"bytes"
	"context"
	"io"
	"log/slog"
	"maps"
	"net/http"
	"net/url"
	"time"

	"github.com/failsafe-go/failsafe-go"
	"github.com/hasura/gotel/otelutils"
	"github.com/relychan/gocompress"
	"github.com/relychan/goutils"
	"github.com/relychan/goutils/httpheader"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
	"go.opentelemetry.io/otel/trace"
)

// Execute handles the HTTP request to the remote server.
func (r *Request) Execute( //nolint:gocognit,funlen,maintidx
	ctx context.Context,
	client HTTPClientGetter,
) (*http.Response, error) {
	if r.method == "" {
		return nil, ErrRequestMethodRequired
	}

	r.retryAttempts = 0
	startTime := time.Now()
	logger := r.getLogger(ctx)
	isDebug := logger.Enabled(ctx, slog.LevelDebug)

	requestLogAttrs := make([]slog.Attr, 0, 5)
	requestLogAttrs = append(requestLogAttrs, slog.String("method", r.method))

	var requestBodyStr string

	if isDebug && r.body != nil &&
		otelutils.IsContentTypeDebuggable(r.Header().Get(httpheader.ContentType)) {
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

	endpoint, err := goutils.ParsePathOrHTTPURL(r.url)
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

	spanContext, span := clientTracer.Start(
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

	if r.options.CustomAttributesFunc != nil {
		commonAttrs = append(
			commonAttrs,
			r.options.CustomAttributesFunc(r)...,
		)
	}

	// the request URL may not be a full URI.
	if endpoint.Host != "" {
		_, port, _ := otelutils.SplitHostPort(endpoint.Host, endpoint.Scheme)
		commonAttrs = newRequestMetricAttributes(4, r.method, endpoint, port)
		span.SetAttributes(commonAttrs...)
		span.SetAttributes(semconv.URLFull(r.url))
		requestLogAttrs = append(requestLogAttrs, slog.String("url", r.url))
	} else {
		requestLogAttrs = append(requestLogAttrs, slog.String("request_path", r.url))
		span.SetAttributes(semconv.URLPath(r.url))
	}

	requestDurationAttrs := make([]attribute.KeyValue, 0, 9)
	copy(requestDurationAttrs, commonAttrs)

	defer func() {
		span.End()
		GetHTTPClientMetrics().RequestDuration.Record(
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

	var cancel context.CancelFunc

	timeout := r.getTimeout()
	if timeout > 0 {
		span.SetAttributes(attribute.String("http.request.timeout", timeout.String()))
		// The cancel function will be wrapped in the response body.
		// Canceling the context before reading body may cause context canceled error.
		spanContext, cancel = context.WithTimeout(spanContext, timeout) //nolint:govet
	}

	if r.getRetryPolicy() == nil {
		resp, err = r.doRequest(spanContext, client, endpoint, body, logger)
	} else {
		resp, err = r.executeWithRetries(spanContext, client, endpoint, body, logger)
	}

	responseLogAttrs := make([]slog.Attr, 0, 4)

	if resp != nil { //nolint:nestif
		responseLogAttrs = append(responseLogAttrs, slog.Int("status", resp.StatusCode))

		if endpoint.Host == "" {
			requestURL := resp.Request.URL.String()
			requestLogAttrs = append(requestLogAttrs, slog.String("url", requestURL))
			span.SetAttributes(semconv.URLFull(requestURL))
		}

		if r.options.IsTraceRequestHeadersEnabled() {
			requestHeaders := otelutils.ExtractTelemetryHeaders(
				resp.Request.Header,
				r.options.AllowedTraceRequestHeaders...,
			)
			otelutils.SetSpanHeaderMatrixAttributes(span, "http.request.header", requestHeaders)
			requestLogAttrs = append(
				requestLogAttrs,
				otelutils.NewHeaderMatrixLogGroupAttrs("headers", requestHeaders),
			)
		}

		if r.options.IsTraceResponseHeadersEnabled() {
			responseHeaders := otelutils.ExtractTelemetryHeaders(
				resp.Header,
				r.options.AllowedTraceResponseHeaders...,
			)
			otelutils.SetSpanHeaderMatrixAttributes(span, "http.response.header", responseHeaders)
			responseLogAttrs = append(
				responseLogAttrs,
				otelutils.NewHeaderMatrixLogGroupAttrs("headers", responseHeaders),
			)
		}

		responseSize := resp.ContentLength
		statusCodeAttr := semconv.HTTPResponseStatusCode(resp.StatusCode)

		span.SetAttributes(statusCodeAttr)

		requestDurationAttrs = append(
			requestDurationAttrs,
			statusCodeAttr,
			newNetworkProtocolVersion(resp.ProtoMajor, resp.ProtoMinor),
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

		if resp.Body != nil {
			if isDebug &&
				otelutils.IsContentTypeDebuggable(resp.Header.Get(httpheader.ContentType)) {
				body, err := io.ReadAll(resp.Body)

				goutils.CatchWarnErrorFunc(resp.Body.Close)

				if cancel != nil {
					cancel()
				}

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
			} else if cancel != nil {
				resp.Body = &responseBodyWithCancel{
					ReadCloser: resp.Body,
					cancel:     cancel,
				}
			}
		} else if cancel != nil {
			cancel()
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

		return resp, err //nolint:govet
	}

	if logger.Enabled(ctx, r.options.LogLevel) {
		logger.LogAttrs(
			ctx,
			r.options.LogLevel,
			resp.Status,
			slog.GroupAttrs("request", requestLogAttrs...),
			slog.GroupAttrs("response", responseLogAttrs...),
			slog.Float64("latency", time.Since(startTime).Seconds()),
		)
	}

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

		return resp, err
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

func (r *Request) doRequest( //nolint:funlen,maintidx
	ctx context.Context,
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
		ctx, span = startClientTrace(
			ctx,
			spanName,
			logger,
		)
	} else {
		ctx, span = startSimpleClientTrace(
			ctx,
			spanName,
		)
	}

	if r.retryAttempts > 0 {
		span.SetAttributes(semconv.HTTPRequestResendCount(r.retryAttempts))
	}

	req, err := client.NewRequest(ctx, r.method, r.url, body)
	if err != nil {
		msg := "failed to create request"

		span.SetAttributes(
			httpRequestMethodAttr(r.method),
			semconv.URLFull(req.URL.String()),
		)
		span.SetStatus(codes.Error, msg)
		span.RecordError(err)

		r.logRequestAttempt(
			ctx,
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

	commonAttrs := newRequestMetricAttributes(8, r.method, req.URL, port)

	if r.options.CustomAttributesFunc != nil {
		commonAttrs = append(
			commonAttrs,
			r.options.CustomAttributesFunc(r)...,
		)
	}

	span.SetAttributes(commonAttrs...)
	span.SetAttributes(semconv.URLFull(req.URL.String()))

	activeRequestsAttrSet := metric.WithAttributeSet(attribute.NewSet(commonAttrs...))

	metrics := GetHTTPClientMetrics()

	metrics.ActiveRequests.Add(
		ctx,
		1,
		activeRequestsAttrSet,
	)

	defer func() {
		metrics.ActiveRequests.Add(
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

	protocolVersionAttr := newNetworkProtocolVersion(req.ProtoMajor, req.ProtoMinor)
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
			ctx,
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
	propagator.Inject(ctx, propagation.HeaderCarrier(req.Header))
	req.Header.Set(httpheader.UserAgent, r.options.UserAgent)

	rawResp, err := client.Do(req)
	if err != nil {
		msg := "failed to execute request"
		span.SetStatus(codes.Error, msg)
		span.RecordError(err)

		r.logRequestAttempt(ctx, span, logger, req, rawResp, err, msg)

		return nil, err
	}

	statusCodeAttr := semconv.HTTPResponseStatusCode(rawResp.StatusCode)
	commonAttrs = append(commonAttrs, statusCodeAttr)
	commonAttrsSet := metric.WithAttributeSet(attribute.NewSet(commonAttrs...))

	span.SetAttributes(statusCodeAttr)

	if rawResp.Request.ContentLength > 0 {
		metrics.RequestBodySize.Record(
			ctx,
			rawResp.Request.ContentLength,
			commonAttrsSet)
		span.SetAttributes(
			semconv.HTTPRequestBodySize(int(rawResp.Request.ContentLength)),
		)
	}

	if rawResp.ContentLength > 0 {
		metrics.ResponseBodySize.Record(
			ctx,
			rawResp.ContentLength,
			commonAttrsSet)
		span.SetAttributes(semconv.HTTPResponseBodySize(int(rawResp.ContentLength)))
	}

	remoteAddr := span.RemoteAddress()

	if remoteAddr != "" {
		peerAddress, peerPort, err := otelutils.SplitHostPort(remoteAddr, endpoint.Scheme)
		if err != nil {
			logger.
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

			r.logRequestAttempt(ctx, span, logger, req, rawResp, nil, rawResp.Status)

			return rawResp, httpErrorFromNoContentResponse(rawResp)
		}

		span.SetStatus(codes.Ok, "")

		r.logRequestAttempt(ctx, span, logger, req, rawResp, nil, rawResp.Status)

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
				ctx,
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
		r.logRequestAttempt(ctx, span, logger, req, rawResp, err, rawResp.Status)

		return rawResp, err
	}

	span.SetStatus(codes.Ok, "")

	r.logRequestAttempt(
		ctx,
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
	ctx context.Context,
	span HTTPClientTracer,
	logger *slog.Logger,
	req *http.Request,
	resp *http.Response,
	err error,
	message string,
) {
	if !logger.Enabled(ctx, slog.LevelDebug) {
		span.EndSpan(ctx)

		return
	}

	logAttrs := make([]any, 0, 4)

	if req != nil {
		requestHeaders := otelutils.ExtractTelemetryHeaders(req.Header)
		otelutils.SetSpanHeaderMatrixAttributes(span, "http.request.header", requestHeaders)

		requestLogAttrs := []slog.Attr{
			slog.String("url", r.url),
			slog.String("method", r.method),
			otelutils.NewHeaderMatrixLogGroupAttrs("headers", requestHeaders),
		}

		logAttrs = append(logAttrs, slog.GroupAttrs("request", requestLogAttrs...))
	}

	if resp != nil {
		responseHeaders := otelutils.ExtractTelemetryHeaders(resp.Header)

		otelutils.SetSpanHeaderMatrixAttributes(span, "http.response.header", responseHeaders)

		responseLogAttrs := []slog.Attr{
			slog.Int("status", resp.StatusCode),
			slog.Int64("size", resp.ContentLength),
			otelutils.NewHeaderMatrixLogGroupAttrs("headers", responseHeaders),
		}

		logAttrs = append(logAttrs, slog.GroupAttrs("response", responseLogAttrs...))
	}

	totalTime := span.EndSpan(ctx)

	logAttrs = append(logAttrs, slog.Float64("latency", totalTime.Seconds()))

	if err != nil {
		logAttrs = append(logAttrs, slog.Any("error", err))
	}

	logger.Debug(message, logAttrs...)
}
