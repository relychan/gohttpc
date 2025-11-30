package gohttpc

import (
	"net"
	"net/http"
	"runtime"
	"time"

	"github.com/relychan/goutils"
)

// HTTPDialerConfig contains options the http.Dialer to connect to an address.
type HTTPDialerConfig struct {
	// The maximum amount of time a dial will wait for a connect to complete.
	// If Deadline is also set, it may fail earlier.
	Timeout *goutils.Duration `json:"timeout,omitempty" jsonschema:"oneof_ref=#/$defs/Duration,oneof_type=null" yaml:"timeout"`
	// Keep-alive probes are enabled by default.
	KeepAliveEnabled *bool `json:"keepAliveEnabled,omitempty" yaml:"keepAliveEnabled"`
	// KeepAliveInterval is the time between keep-alive probes. If zero, a default value of 15 seconds is used.
	KeepAliveInterval *goutils.Duration `json:"keepAliveInterval,omitempty" jsonschema:"oneof_ref=#/$defs/Duration,oneof_type=null" yaml:"keepAliveInterval"`
	// KeepAliveCount is the maximum number of keep-alive probes that can go unanswered before dropping a connection.
	// If zero, a default value of 9 is used.
	KeepAliveCount *int `json:"keepAliveCount,omitempty" jsonschema:"nullable,min=0" yaml:"keepAliveCount"`
	// KeepAliveIdle is the time that the connection must be idle before the first keep-alive probe is sent.
	// If zero, a default value of 15 seconds is used.
	KeepAliveIdle *goutils.Duration `json:"keepAliveIdle,omitempty" jsonschema:"oneof_ref=#/$defs/Duration,oneof_type=null" yaml:"keepAliveIdle"`
	// FallbackDelay specifies the length of time to wait before spawning a RFC 6555 Fast Fallback connection.
	// That is, this is the amount of time to wait for IPv6 to succeed before assuming that IPv6 is misconfigured and falling back to IPv4.
	// If zero, a default delay of 300ms is used. A negative value disables Fast Fallback support.
	FallbackDelay *goutils.Duration `json:"fallbackDelay,omitempty" jsonschema:"oneof_ref=#/$defs/Duration,oneof_type=null" yaml:"fallbackDelay"`
}

// HTTPTransportConfig stores the http.Transport configuration for the http client.
type HTTPTransportConfig struct {
	// Options the http.Dialer to connect to an address
	Dialer *HTTPDialerConfig `json:"dialer,omitempty" yaml:"dialer"`
	// Idle connection timeout. The maximum amount of time an idle (keep-alive) connection will remain idle before closing itself. Zero means no limit.
	IdleConnTimeout *goutils.Duration `json:"idleConnTimeout,omitempty" jsonschema:"oneof_ref=#/$defs/Duration,oneof_type=null" yaml:"idleConnTimeout"`
	// Response header timeout, if non-zero, specifies the amount of time to wait for a server's response headers after fully writing the request (including its body, if any).
	// This time does not include the time to read the response body.
	// This timeout is used to cover cases where the tcp connection works but the server never answers.
	ResponseHeaderTimeout *goutils.Duration `json:"responseHeaderTimeout,omitempty" jsonschema:"oneof_ref=#/$defs/Duration,oneof_type=null" yaml:"responseHeaderTimeout"`
	// TLS handshake timeout is the maximum amount of time to wait for a TLS handshake. Zero means no timeout.
	TLSHandshakeTimeout *goutils.Duration `json:"tlsHandshakeTimeout,omitempty" jsonschema:"oneof_ref=#/$defs/Duration,oneof_type=null" yaml:"tlsHandshakeTimeout"`
	// Expect continue timeout, if non-zero, specifies the amount of time to wait for a server's first response headers after fully writing the request headers if the request has an "Expect: 100-continue" header.
	ExpectContinueTimeout *goutils.Duration `json:"expectContinueTimeout,omitempty" jsonschema:"oneof_ref=#/$defs/Duration,oneof_type=null" yaml:"expectContinueTimeout"`
	// MaxIdleConns controls the maximum number of idle (keep-alive) connections across all hosts. Zero means no limit.
	MaxIdleConns *int `json:"maxIdleConns,omitempty" jsonschema:"nullable,min=0" yaml:"maxIdleConns"`
	// MaxIdleConnsPerHost, if non-zero, controls the maximum idle (keep-alive) connections to keep per-host.
	MaxIdleConnsPerHost *int `json:"maxIdleConnsPerHost,omitempty" jsonschema:"nullable,min=0" yaml:"maxIdleConnsPerHost"`
	// MaxConnsPerHost optionally limits the total number of connections per host, including connections in the dialing, active, and idle states.
	// On limit violation, dials will block. Zero means no limit.
	MaxConnsPerHost *int `json:"maxConnsPerHost,omitempty" jsonschema:"nullable,min=0" yaml:"maxConnsPerHost"`
	// MaxResponseHeaderBytes specifies a limit on how many response bytes are allowed in the server's response header.
	// Zero means to use a default limit.
	MaxResponseHeaderBytes *int64 `json:"maxResponseHeaderBytes,omitempty" jsonschema:"nullable,min=0" yaml:"maxResponseHeaderBytes"`
	// ReadBufferSize specifies the size of the read buffer used when reading from the transport.
	// If zero, a default (currently 4KB) is used.
	ReadBufferSize *int `json:"readBufferSize,omitempty" jsonschema:"nullable,min=0" yaml:"readBufferSize"`
	// WriteBufferSize specifies the size of the write buffer used when writing to the transport.
	// If zero, a default (currently 4KB) is used.
	WriteBufferSize *int `json:"writeBufferSize,omitempty" jsonschema:"nullable,min=0" yaml:"writeBufferSize"`
	// DisableKeepAlives, if true, disables HTTP keep-alives and will only use the connection to the server for a single HTTP request.
	// This is unrelated to the similarly named TCP keep-alives.
	DisableKeepAlives bool `json:"disableKeepAlives,omitempty" yaml:"disableKeepAlives"`
}

// TransportFromConfig creates an http transport from the configuration.
func TransportFromConfig(
	ttc *HTTPTransportConfig,
	clientOptions *ClientOptions,
) *http.Transport {
	var dialerConf *HTTPDialerConfig

	if ttc != nil {
		dialerConf = ttc.Dialer
	}

	dialer := DialerFromConfig(dialerConf)

	defaultTransport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		MaxIdleConns:          100,
		ResponseHeaderTimeout: time.Minute,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 10 * time.Second,
		ForceAttemptHTTP2:     true,
		DisableCompression:    true,
	}

	defaultTransport.DialContext = transportDialContext(
		dialer,
	)

	if ttc == nil {
		return defaultTransport
	}

	return applyTransport(ttc, defaultTransport)
}

func applyTransport(ttc *HTTPTransportConfig, defaultTransport *http.Transport) *http.Transport {
	if ttc.DisableKeepAlives {
		defaultTransport.DisableKeepAlives = true
	}

	if ttc.ExpectContinueTimeout != nil {
		defaultTransport.ExpectContinueTimeout = time.Duration(*ttc.ExpectContinueTimeout)
	}

	if ttc.IdleConnTimeout != nil {
		defaultTransport.IdleConnTimeout = time.Duration(*ttc.IdleConnTimeout)
	}

	if ttc.MaxConnsPerHost != nil {
		defaultTransport.MaxConnsPerHost = *ttc.MaxConnsPerHost
	}

	if ttc.MaxIdleConns != nil {
		defaultTransport.MaxIdleConns = *ttc.MaxIdleConns
	}

	if ttc.MaxIdleConnsPerHost != nil && *ttc.MaxIdleConnsPerHost > 0 {
		defaultTransport.MaxIdleConnsPerHost = *ttc.MaxIdleConnsPerHost
	} else {
		defaultTransport.MaxIdleConnsPerHost = runtime.GOMAXPROCS(0) + 1
	}

	if ttc.ResponseHeaderTimeout != nil {
		defaultTransport.ResponseHeaderTimeout = time.Duration(*ttc.ResponseHeaderTimeout)
	}

	if ttc.TLSHandshakeTimeout != nil {
		defaultTransport.TLSHandshakeTimeout = time.Duration(*ttc.TLSHandshakeTimeout)
	}

	if ttc.MaxResponseHeaderBytes != nil && *ttc.MaxResponseHeaderBytes > 0 {
		defaultTransport.MaxResponseHeaderBytes = *ttc.MaxResponseHeaderBytes
	}

	if ttc.ReadBufferSize != nil && *ttc.ReadBufferSize > 0 {
		defaultTransport.ReadBufferSize = *ttc.ReadBufferSize
	}

	if ttc.WriteBufferSize != nil && *ttc.WriteBufferSize > 0 {
		defaultTransport.WriteBufferSize = *ttc.WriteBufferSize
	}

	return defaultTransport
}

// DialerFromConfig creates a net dialer from the configuration.
func DialerFromConfig(conf *HTTPDialerConfig) *net.Dialer {
	dialer := &net.Dialer{
		Timeout: 30 * time.Second,
		KeepAliveConfig: net.KeepAliveConfig{
			Enable:   true,
			Interval: 30 * time.Second,
		},
	}

	if conf == nil {
		return dialer
	}

	if conf.Timeout != nil {
		dialer.Timeout = time.Duration(*conf.Timeout)
	}

	if conf.KeepAliveEnabled != nil {
		dialer.KeepAliveConfig.Enable = *conf.KeepAliveEnabled
	}

	if conf.KeepAliveCount != nil {
		dialer.KeepAliveConfig.Count = *conf.KeepAliveCount
	}

	if conf.KeepAliveIdle != nil {
		dialer.KeepAliveConfig.Idle = time.Duration(*conf.KeepAliveIdle)
	}

	if conf.KeepAliveInterval != nil {
		dialer.KeepAliveConfig.Interval = time.Duration(*conf.KeepAliveInterval)
	}

	if conf.FallbackDelay != nil {
		dialer.FallbackDelay = time.Duration(*conf.FallbackDelay)
	}

	return dialer
}
