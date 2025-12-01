package httpconfig

import (
	"crypto/tls"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/hasura/goenvconf"
)

func TestTLSClientCertificate_IsZero(t *testing.T) {
	t.Run("returns true when all fields are nil", func(t *testing.T) {
		cert := TLSClientCertificate{}

		if !cert.IsZero() {
			t.Error("expected IsZero to return true")
		}
	})

	t.Run("returns false when CertFile is set", func(t *testing.T) {
		certFile := goenvconf.NewEnvStringValue("cert.pem")
		cert := TLSClientCertificate{
			CertFile: &certFile,
		}

		if cert.IsZero() {
			t.Error("expected IsZero to return false")
		}
	})

	t.Run("returns false when CertPem is set", func(t *testing.T) {
		certPem := goenvconf.NewEnvStringValue("base64cert")
		cert := TLSClientCertificate{
			CertPem: &certPem,
		}

		if cert.IsZero() {
			t.Error("expected IsZero to return false")
		}
	})
}

func TestTLSConfig_GetMinVersion(t *testing.T) {
	t.Run("returns default min version when empty", func(t *testing.T) {
		config := TLSConfig{}

		version, err := config.GetMinVersion()

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if version != defaultMinTLSVersion {
			t.Errorf("expected %d, got %d", defaultMinTLSVersion, version)
		}
	})

	t.Run("parses valid TLS versions", func(t *testing.T) {
		testCases := []struct {
			Version  string
			Expected uint16
		}{
			{"1.0", tls.VersionTLS10},
			{"1.1", tls.VersionTLS11},
			{"1.2", tls.VersionTLS12},
			{"1.3", tls.VersionTLS13},
		}

		for _, tc := range testCases {
			t.Run("TLS "+tc.Version, func(t *testing.T) {
				config := TLSConfig{
					MinVersion: tc.Version,
				}

				version, err := config.GetMinVersion()

				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}

				if version != tc.Expected {
					t.Errorf("expected %d, got %d", tc.Expected, version)
				}
			})
		}
	})

	t.Run("returns error for unsupported version", func(t *testing.T) {
		config := TLSConfig{
			MinVersion: "1.4",
		}

		_, err := config.GetMinVersion()

		if err == nil {
			t.Error("expected error for unsupported version")
		}

		if !errors.Is(err, errUnsupportedTLSVersion) {
			t.Errorf("expected errUnsupportedTLSVersion, got %v", err)
		}
	})
}

func TestTLSConfig_GetMaxVersion(t *testing.T) {
	t.Run("returns default max version when empty", func(t *testing.T) {
		config := TLSConfig{}

		version, err := config.GetMaxVersion()

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if version != defaultMaxTLSVersion {
			t.Errorf("expected %d, got %d", defaultMaxTLSVersion, version)
		}
	})

	t.Run("parses valid TLS versions", func(t *testing.T) {
		config := TLSConfig{
			MaxVersion: "1.3",
		}

		version, err := config.GetMaxVersion()

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if version != tls.VersionTLS13 {
			t.Errorf("expected %d, got %d", tls.VersionTLS13, version)
		}
	})
}

func TestTLSConfig_Validate(t *testing.T) {
	t.Run("validates successfully with empty config", func(t *testing.T) {
		config := TLSConfig{}

		err := config.Validate()

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("returns error when minVersion is invalid", func(t *testing.T) {
		config := TLSConfig{
			MinVersion: "invalid",
		}

		err := config.Validate()

		if err == nil {
			t.Error("expected error for invalid minVersion")
		}
	})

	t.Run("returns error when maxVersion is invalid", func(t *testing.T) {
		config := TLSConfig{
			MaxVersion: "invalid",
		}

		err := config.Validate()

		if err == nil {
			t.Error("expected error for invalid maxVersion")
		}
	})

	t.Run("returns error when minVersion > maxVersion", func(t *testing.T) {
		config := TLSConfig{
			MinVersion: "1.3",
			MaxVersion: "1.2",
		}

		err := config.Validate()

		if err == nil {
			t.Error("expected error when minVersion > maxVersion")
		}

		if !errors.Is(err, errTLSMinVersionGreaterThanMaxVersion) {
			t.Errorf("expected errTLSMinVersionGreaterThanMaxVersion, got %v", err)
		}
	})

	t.Run("validates successfully when minVersion <= maxVersion", func(t *testing.T) {
		config := TLSConfig{
			MinVersion: "1.2",
			MaxVersion: "1.3",
		}

		err := config.Validate()

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("returns error when certificate has both file and pem", func(t *testing.T) {
		certFile := goenvconf.NewEnvStringValue("cert.pem")
		certPem := goenvconf.NewEnvStringValue("base64cert")

		config := TLSConfig{
			Certificates: []TLSClientCertificate{
				{
					CertFile: &certFile,
					CertPem:  &certPem,
				},
			},
		}

		err := config.Validate()

		if err == nil {
			t.Error("expected error when certificate has both file and pem")
		}

		if !errors.Is(err, errCertificateRequireEitherFileOrPEM) {
			t.Errorf("expected errCertificateRequireEitherFileOrPEM, got %v", err)
		}
	})

	t.Run("returns error when key has both file and pem", func(t *testing.T) {
		keyFile := goenvconf.NewEnvStringValue("key.pem")
		keyPem := goenvconf.NewEnvStringValue("base64key")

		config := TLSConfig{
			Certificates: []TLSClientCertificate{
				{
					KeyFile: &keyFile,
					KeyPem:  &keyPem,
				},
			},
		}

		err := config.Validate()

		if err == nil {
			t.Error("expected error when key has both file and pem")
		}

		if !errors.Is(err, errCertificateRequireEitherFileOrPEM) {
			t.Errorf("expected errCertificateRequireEitherFileOrPEM, got %v", err)
		}
	})
}

func TestLoadCertificateString(t *testing.T) {
	t.Run("returns nil when env string is empty", func(t *testing.T) {
		certEnv := goenvconf.NewEnvStringValue("")

		data, err := loadCertificateString(certEnv)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if data != nil {
			t.Error("expected data to be nil")
		}
	})

	t.Run("decodes valid base64 string", func(t *testing.T) {
		original := "test certificate data"
		encoded := base64.StdEncoding.EncodeToString([]byte(original))
		certEnv := goenvconf.NewEnvStringValue(encoded)

		data, err := loadCertificateString(certEnv)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if string(data) != original {
			t.Errorf("expected %s, got %s", original, string(data))
		}
	})

	t.Run("returns error for invalid base64", func(t *testing.T) {
		certEnv := goenvconf.NewEnvStringValue("not-valid-base64!!!")

		_, err := loadCertificateString(certEnv)

		if err == nil {
			t.Error("expected error for invalid base64")
		}

		if !errors.Is(err, errCertificateInvalidBase64) {
			t.Errorf("expected errCertificateInvalidBase64, got %v", err)
		}
	})
}

func TestConvertCipherSuites(t *testing.T) {
	t.Run("returns empty slice for empty input", func(t *testing.T) {
		result, err := convertCipherSuites([]string{})

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if len(result) != 0 {
			t.Errorf("expected empty slice, got %d items", len(result))
		}
	})

	t.Run("converts valid cipher suites", func(t *testing.T) {
		// Get a valid cipher suite name from the supported list
		supportedSuites := tls.CipherSuites()
		if len(supportedSuites) == 0 {
			t.Skip("no supported cipher suites available")
		}

		suiteName := supportedSuites[0].Name

		result, err := convertCipherSuites([]string{suiteName})

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if len(result) != 1 {
			t.Errorf("expected 1 cipher suite, got %d", len(result))
		}

		if result[0] != supportedSuites[0].ID {
			t.Errorf("expected ID %d, got %d", supportedSuites[0].ID, result[0])
		}
	})

	t.Run("returns error for unsupported cipher suite", func(t *testing.T) {
		result, err := convertCipherSuites([]string{"INVALID_CIPHER_SUITE"})

		if err == nil {
			t.Error("expected error for unsupported cipher suite")
		}

		if !errors.Is(err, errUnsupportedCipherSuite) {
			t.Errorf("expected errUnsupportedCipherSuite, got %v", err)
		}

		if len(result) != 0 {
			t.Errorf("expected empty result, got %d items", len(result))
		}
	})
}

func TestLoadEitherCertPemOrFile(t *testing.T) {
	t.Run("returns error when both are nil", func(t *testing.T) {
		_, err := loadEitherCertPemOrFile(nil, nil)

		if err == nil {
			t.Error("expected error when both are nil")
		}

		if !errors.Is(err, errTLSPEMAndFileEmpty) {
			t.Errorf("expected errTLSPEMAndFileEmpty, got %v", err)
		}
	})

	t.Run("loads from PEM when provided", func(t *testing.T) {
		original := "test certificate"
		encoded := base64.StdEncoding.EncodeToString([]byte(original))
		certPem := goenvconf.NewEnvStringValue(encoded)

		data, err := loadEitherCertPemOrFile(&certPem, nil)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if string(data) != original {
			t.Errorf("expected %s, got %s", original, string(data))
		}
	})

	t.Run("loads from file when PEM is empty", func(t *testing.T) {
		// Create a temporary test file
		tmpDir := t.TempDir()
		certFile := filepath.Join(tmpDir, "test.crt")
		testData := []byte("test certificate data")

		err := os.WriteFile(certFile, testData, 0600)
		if err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		certFileEnv := goenvconf.NewEnvStringValue(certFile)
		emptyPem := goenvconf.NewEnvStringValue("")

		data, err := loadEitherCertPemOrFile(&emptyPem, &certFileEnv)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if string(data) != string(testData) {
			t.Errorf("expected %s, got %s", string(testData), string(data))
		}
	})

	t.Run("returns error when file does not exist", func(t *testing.T) {
		certFileEnv := goenvconf.NewEnvStringValue("/nonexistent/file.crt")

		_, err := loadEitherCertPemOrFile(nil, &certFileEnv)

		if err == nil {
			t.Error("expected error for nonexistent file")
		}
	})
}
