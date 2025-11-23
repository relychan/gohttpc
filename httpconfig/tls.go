package httpconfig

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/hasura/goenvconf"
	"github.com/relychan/goutils"
)

var systemCertPool = x509.SystemCertPool

// We should avoid that users unknowingly use a vulnerable TLS version.
// The defaults should be a safe configuration.
const defaultMinTLSVersion = tls.VersionTLS12

// Uses the default MaxVersion from "crypto/tls" which is the maximum supported version.
const defaultMaxTLSVersion = 0

var tlsVersions = map[string]uint16{
	"1.0": tls.VersionTLS10,
	"1.1": tls.VersionTLS11,
	"1.2": tls.VersionTLS12,
	"1.3": tls.VersionTLS13,
}

var (
	errCertificateRequireEitherFileOrPEM = errors.New(
		"provide either a certificate or the PEM-encoded string, but not both",
	)
	errCertificateInvalidBase64 = errors.New(
		"certificate string must be in base64 format",
	)
	errTLSMinVersionGreaterThanMaxVersion = errors.New(
		"invalid TLS configuration: minVersion cannot be greater than maxVersion",
	)
	errUnsupportedTLSVersion  = errors.New("unsupported TLS version")
	errUnsupportedCipherSuite = errors.New("invalid TLS cipher suite")
	errTLSPEMAndFileEmpty     = errors.New("both PEM and file are empty")
)

// TLSClientCertificate represents a cert and key pair certificate.
type TLSClientCertificate struct {
	// CertFile is the path to the TLS cert to use for TLS required connections.
	CertFile *goenvconf.EnvString `json:"certFile,omitempty" yaml:"certFile,omitempty"`
	// CertPem is alternative to certFile. Provide the certificate contents as a base64-encoded string instead of a filepath.
	CertPem *goenvconf.EnvString `json:"certPem,omitempty" yaml:"certPem,omitempty"`
	// KeyFile is the path to the TLS key to use for TLS required connections.
	KeyFile *goenvconf.EnvString `json:"keyFile,omitempty" yaml:"keyFile,omitempty"`
	// KeyPem is the alternative to keyFile. Provide the key contents as a base64-encoded string instead of a filepath.
	KeyPem *goenvconf.EnvString `json:"keyPem,omitempty" yaml:"keyPem,omitempty"`
}

// IsZero checks if the client certificate is empty.
func (tc TLSClientCertificate) IsZero() bool {
	return ((tc.CertPem == nil || tc.CertPem.IsZero()) &&
		(tc.CertFile == nil || tc.CertFile.IsZero())) &&
		((tc.KeyFile == nil || tc.KeyFile.IsZero()) &&
			(tc.KeyPem == nil || tc.KeyPem.IsZero()))
}

// LoadKeyPair loads the X509 key pair from configurations.
func (tc TLSClientCertificate) LoadKeyPair() (*tls.Certificate, error) {
	certData, err := loadEitherCertPemOrFile(tc.CertPem, tc.CertFile)
	if err != nil {
		return nil, fmt.Errorf("certificate error: %w", err)
	}

	keyData, err := loadEitherCertPemOrFile(tc.KeyPem, tc.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("key error: %w", err)
	}

	certificate, err := tls.X509KeyPair(certData, keyData)
	if err != nil {
		return nil, fmt.Errorf("failed to load TLS cert and key PEMs: %w", err)
	}

	return &certificate, nil
}

// TLSConfig represents the transport layer security (LTS) configuration for the mutualTLS authentication.
type TLSConfig struct {
	// Interval to reload certificates. Only takes effect for file-path certificates.
	// Default value is 24 hours.
	ReloadInterval *goutils.Duration `json:"reloadInterval,omitempty" jsonschema:"oneof_ref=#/$defs/Duration,oneof_type=null" yaml:"reloadInterval"`
	// RootCAFile represents paths to root certificates. For a client this verifies the server certificate. For a server this verifies client certificates.
	// If empty uses system root CA.
	RootCAFile []goenvconf.EnvString `json:"rootCAFile,omitempty" yaml:"rootCAFile,omitempty"`
	// RootCAPem is the alternative to rootCAFile. Provide the CA cert contents as a base64-encoded string instead of a filepath.
	RootCAPem []goenvconf.EnvString `json:"rootCAPem,omitempty" yaml:"rootCAPem,omitempty"`
	// CAFile is the path to the CA cert. For a client this verifies the server certificate. For a server this verifies client certificates.
	// If empty uses system root CA.
	CAFile []goenvconf.EnvString `json:"caFile,omitempty" yaml:"caFile,omitempty"`
	// CAPem is alternative to caFile. Provide the CA cert contents as a base64-encoded string instead of a filepath.
	CAPem []goenvconf.EnvString `json:"caPem,omitempty" yaml:"caPem,omitempty"`
	// Certificates contains the list of client certificates.
	Certificates []TLSClientCertificate `json:"certificates,omitempty" yaml:"certificates,omitempty"`
	// InsecureSkipVerify you can configure TLS to be enabled but skip verifying the server's certificate chain.
	InsecureSkipVerify *goenvconf.EnvBool `json:"insecureSkipVerify,omitempty" yaml:"insecureSkipVerify,omitempty"`
	// IncludeSystemCACertsPool whether to load the system certificate authorities pool alongside the certificate authority.
	IncludeSystemCACertsPool *goenvconf.EnvBool `json:"includeSystemCACertsPool,omitempty" yaml:"includeSystemCACertsPool,omitempty"`
	// Minimum acceptable TLS version.
	MinVersion string `json:"minVersion,omitempty" yaml:"minVersion,omitempty"`
	// Maximum acceptable TLS version.
	MaxVersion string `json:"maxVersion,omitempty" yaml:"maxVersion,omitempty"`
	// Explicit cipher suites can be set. If left blank, a safe default list is used.
	// See https://go.dev/src/crypto/tls/cipher_suites.go for a list of supported cipher suites.
	CipherSuites []string `json:"cipherSuites,omitempty" yaml:"cipherSuites,omitempty"`
	// ServerName requested by client for virtual hosting.
	// This sets the ServerName in the TLSConfig. Please refer to
	// https://godoc.org/crypto/tls#Config for more information. (optional)
	ServerName *goenvconf.EnvString `json:"serverName,omitempty" yaml:"serverName,omitempty"`
}

// Validate if the current instance is valid.
func (tc TLSConfig) Validate() error {
	minTLS, err := tc.GetMinVersion()
	if err != nil {
		return fmt.Errorf("minVersion: %w", err)
	}

	maxTLS, err := tc.GetMaxVersion()
	if err != nil {
		return fmt.Errorf("maxVersion: %w", err)
	}

	if maxTLS < minTLS && maxTLS != defaultMaxTLSVersion {
		return errTLSMinVersionGreaterThanMaxVersion
	}

	err = tc.validateCertificates()
	if err != nil {
		return err
	}

	if tc.IncludeSystemCACertsPool != nil {
		_, err := tc.IncludeSystemCACertsPool.GetOrDefault(false)
		if err != nil {
			return err
		}
	}

	if tc.ServerName != nil {
		_, err := tc.ServerName.GetOrDefault("")
		if err != nil {
			return err
		}
	}

	return nil
}

// GetMinVersion parses the minx TLS version from string.
func (tc TLSConfig) GetMinVersion() (uint16, error) {
	return tc.convertTLSVersion(tc.MinVersion, defaultMinTLSVersion)
}

// GetMaxVersion parses the max TLS version from string.
func (tc TLSConfig) GetMaxVersion() (uint16, error) {
	return tc.convertTLSVersion(tc.MinVersion, defaultMaxTLSVersion)
}

func (tc TLSConfig) validateCertificates() error {
	for i, cert := range tc.Certificates {
		if cert.CertFile != nil && cert.CertPem != nil {
			certFile, err := cert.CertFile.GetOrDefault("")
			if err != nil {
				return fmt.Errorf("certificates[%d].certFile: %w", i, err)
			}

			certPem, err := cert.CertPem.GetOrDefault("")
			if err != nil {
				return fmt.Errorf("certificates[%d].certPem: %w", i, err)
			}

			if certFile != "" && certPem != "" {
				return errCertificateRequireEitherFileOrPEM
			}
		}

		if cert.KeyFile != nil && cert.KeyPem != nil {
			keyFile, err := cert.KeyFile.GetOrDefault("")
			if err != nil {
				return fmt.Errorf("certificates[%d].keyFile: %w", i, err)
			}

			keyPem, err := cert.KeyPem.GetOrDefault("")
			if err != nil {
				return fmt.Errorf("certificates[%d].keyPem: %w", i, err)
			}

			if keyFile != "" && keyPem != "" {
				return errCertificateRequireEitherFileOrPEM
			}
		}
	}

	return nil
}

func (tc TLSConfig) convertTLSVersion(v string, defaultVersion uint16) (uint16, error) {
	// Use a default that is explicitly defined
	if v == "" {
		return defaultVersion, nil
	}

	val, ok := tlsVersions[v]
	if !ok {
		return 0, fmt.Errorf("%w: %q", errUnsupportedTLSVersion, v)
	}

	return val, nil
}

// loadTLSConfig loads TLS certificates and returns a tls.Config.
// This will set the RootCAs and Certificates of a tls.Config.
func loadTLSConfig(tlsConfig *TLSConfig) (*tls.Config, error) {
	var (
		insecureSkipVerify bool
		err                error
	)

	if tlsConfig.InsecureSkipVerify != nil {
		insecureSkipVerify, err = tlsConfig.InsecureSkipVerify.GetOrDefault(false)
		if err != nil {
			return nil, fmt.Errorf("failed to parse insecureSkipVerify: %w", err)
		}
	}

	certPool, err := loadSystemCACertPool(tlsConfig)
	if err != nil {
		return nil, err
	}

	minTLS, err := tlsConfig.GetMinVersion()
	if err != nil {
		return nil, fmt.Errorf("minVersion: %w", err)
	}

	maxTLS, err := tlsConfig.GetMaxVersion()
	if err != nil {
		return nil, fmt.Errorf("maxVersion: %w", err)
	}

	cipherSuites, err := convertCipherSuites(tlsConfig.CipherSuites)
	if err != nil {
		return nil, err
	}

	var serverName string

	if tlsConfig.ServerName != nil {
		serverName, err = tlsConfig.ServerName.GetOrDefault("")
		if err != nil {
			return nil, fmt.Errorf("failed to get TLS server name: %w", err)
		}
	}

	result := &tls.Config{
		RootCAs:            certPool,
		MinVersion:         minTLS,
		MaxVersion:         maxTLS,
		CipherSuites:       cipherSuites,
		ServerName:         serverName,
		InsecureSkipVerify: insecureSkipVerify, //nolint:gosec
	}

	return result, addTLSCertificates(result, tlsConfig)
}

func loadSystemCACertPool(tlsConfig *TLSConfig) (*x509.CertPool, error) {
	// There is no need to load the System Certs for RootCAs because
	// if the value is nil, it will default to checking against th System Certs.
	var err error

	var certPool *x509.CertPool

	var includeSystemCACertsPool bool

	if tlsConfig.IncludeSystemCACertsPool != nil {
		includeSystemCACertsPool, err = tlsConfig.IncludeSystemCACertsPool.GetOrDefault(false)
		if err != nil {
			return nil, fmt.Errorf("invalid includeSystemCACertsPool config: %w", err)
		}
	}

	if includeSystemCACertsPool {
		scp, err := systemCertPool()
		if err != nil {
			return nil, err
		}

		if scp != nil {
			certPool = scp
		}
	}

	if certPool == nil {
		certPool = x509.NewCertPool()
	}

	return certPool, nil
}

func convertCipherSuites(cipherSuites []string) ([]uint16, error) {
	var result []uint16

	var errs []error

	for _, suite := range cipherSuites {
		found := false

		for _, supported := range tls.CipherSuites() {
			if suite == supported.Name {
				result = append(result, supported.ID)
				found = true

				break
			}
		}

		if !found {
			errs = append(errs, fmt.Errorf("%w: %q", errUnsupportedCipherSuite, suite))
		}
	}

	return result, errors.Join(errs...)
}

func addTLSCertificates(tlsc *tls.Config, tlsConf *TLSConfig) error {
	err := addTLSCertPoolCAs(tlsc.RootCAs, tlsConf.RootCAPem, tlsConf.RootCAFile)
	if err != nil {
		return fmt.Errorf("RootCAs: %w", err)
	}

	err = addTLSCertPoolCAs(tlsc.ClientCAs, tlsConf.CAPem, tlsConf.CAFile)
	if err != nil {
		return fmt.Errorf("ClientCAs: %w", err)
	}

	return addTLSClientCertificates(tlsc, tlsConf.Certificates)
}

func addTLSCertPoolCAs(
	certPool *x509.CertPool,
	caPEMs []goenvconf.EnvString,
	caFiles []goenvconf.EnvString,
) error {
	for i, certStrEnv := range caPEMs {
		certBytes, err := loadCertificateString(certStrEnv)
		if err != nil {
			return fmt.Errorf("failed to load root certificate string at %d: %w", i, err)
		}

		if len(certBytes) == 0 {
			slog.Warn(fmt.Sprintf("the root certificate string at %d is empty", i))

			continue
		}

		certPool.AppendCertsFromPEM(certBytes)
	}

	for i, certEnv := range caFiles {
		certFile, err := certEnv.GetOrDefault("")
		if err != nil {
			return fmt.Errorf("failed to load root certificate file at %d: %w", i, err)
		}

		if certFile == "" {
			slog.Warn(fmt.Sprintf("the root certificate file path at %d is empty", i))

			continue
		}

		certData, err := os.ReadFile(filepath.Clean(certFile))
		if err != nil {
			return fmt.Errorf("failed to read certificate file at %d: %w", i, err)
		}

		certPool.AppendCertsFromPEM(certData)
	}

	return nil
}

func addTLSClientCertificates(tlsc *tls.Config, certs []TLSClientCertificate) error {
	for i, cert := range certs {
		c, err := cert.LoadKeyPair()
		if err != nil {
			return fmt.Errorf("failed to load client certificate at %d: %w", i, err)
		}

		tlsc.Certificates = append(tlsc.Certificates, *c)
	}

	return nil
}

func loadCertificateString(certEnv goenvconf.EnvString) ([]byte, error) {
	certBase64, err := certEnv.GetOrDefault("")
	if err != nil {
		return nil, err
	}

	if certBase64 == "" {
		return nil, nil
	}

	certStr, err := base64.StdEncoding.DecodeString(certBase64)
	if err != nil {
		return nil, errCertificateInvalidBase64
	}

	return certStr, nil
}

func loadEitherCertPemOrFile(certPemEnv, certFileEnv *goenvconf.EnvString) ([]byte, error) {
	var certData []byte

	var err error

	if certPemEnv != nil {
		certData, err = loadCertificateString(*certPemEnv)
		if err != nil {
			return nil, fmt.Errorf("failed to load PEM: %w", err)
		}
	}

	if len(certData) == 0 && certFileEnv != nil {
		certFile, err := certFileEnv.GetOrDefault("")
		if err != nil {
			return nil, fmt.Errorf("failed to load file: %w", err)
		}

		if certFile != "" {
			certData, err = os.ReadFile(filepath.Clean(certFile))
			if err != nil {
				return nil, fmt.Errorf("failed to read certificate file: %w", err)
			}
		}
	}

	if len(certData) == 0 {
		return nil, errTLSPEMAndFileEmpty
	}

	return certData, nil
}
