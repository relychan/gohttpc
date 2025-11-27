package gohttpc

import (
	"errors"
	"net"
	"testing"
)

func TestClassifyDNSError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name: "DNS error with IsNotFound returns host_not_found",
			err: &net.DNSError{
				Err:        "no such host",
				Name:       "example.invalid",
				IsNotFound: true,
			},
			expected: "host_not_found",
		},
		{
			name: "DNS error with IsTimeout returns timeout",
			err: &net.DNSError{
				Err:       "i/o timeout",
				Name:      "example.com",
				IsTimeout: true,
			},
			expected: "timeout",
		},
		{
			name: "DNS error without specific flags returns _OTHER",
			err: &net.DNSError{
				Err:  "temporary failure",
				Name: "example.com",
			},
			expected: "_OTHER",
		},
		{
			name:     "non-DNS error returns _OTHER",
			err:      errors.New("connection refused"),
			expected: "_OTHER",
		},
		{
			name: "wrapped DNS error with IsNotFound returns host_not_found",
			err: errors.Join(
				errors.New("lookup failed"),
				&net.DNSError{
					Err:        "no such host",
					Name:       "example.invalid",
					IsNotFound: true,
				},
			),
			expected: "host_not_found",
		},
		{
			name: "wrapped DNS error with IsTimeout returns timeout",
			err: errors.Join(
				errors.New("lookup failed"),
				&net.DNSError{
					Err:       "i/o timeout",
					Name:      "example.com",
					IsTimeout: true,
				},
			),
			expected: "timeout",
		},
		{
			name: "DNS error with both flags prioritizes IsNotFound",
			err: &net.DNSError{
				Err:        "complex error",
				Name:       "example.com",
				IsNotFound: true,
				IsTimeout:  true,
			},
			expected: "host_not_found",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := classifyDNSError(tc.err)
			if result != tc.expected {
				t.Errorf("classifyDNSError() = %q, want %q", result, tc.expected)
			}
		})
	}
}
