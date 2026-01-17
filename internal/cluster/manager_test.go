package cluster

import (
	"errors"
	"testing"
)

func TestIsPermanentAuthError(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		permanent bool
	}{
		{
			name:      "nil error",
			err:       nil,
			permanent: false,
		},
		{
			name:      "unauthorized error",
			err:       errors.New("kubectl auth can-i --list failed: exit status 1 (output: error: You must be logged in to the server (Unauthorized)\n)"),
			permanent: true,
		},
		{
			name:      "forbidden error",
			err:       errors.New("kubectl auth can-i --list failed: exit status 1 (output: error: Forbidden\n)"),
			permanent: true,
		},
		{
			name:      "certificate error",
			err:       errors.New("x509: certificate signed by unknown authority"),
			permanent: true,
		},
		{
			name:      "tls error",
			err:       errors.New("tls: failed to verify certificate"),
			permanent: true,
		},
		{
			name:      "invalid token",
			err:       errors.New("invalid token"),
			permanent: true,
		},
		{
			name:      "token expired",
			err:       errors.New("token expired"),
			permanent: true,
		},
		{
			name:      "credentials error",
			err:       errors.New("invalid credentials"),
			permanent: true,
		},
		{
			name:      "connection refused (transient)",
			err:       errors.New("dial tcp: connection refused"),
			permanent: false,
		},
		{
			name:      "timeout (transient)",
			err:       errors.New("context deadline exceeded"),
			permanent: false,
		},
		{
			name:      "server unavailable (transient)",
			err:       errors.New("the server is currently unable to handle the request"),
			permanent: false,
		},
		{
			name:      "network unreachable (transient)",
			err:       errors.New("network is unreachable"),
			permanent: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isPermanentAuthError(tt.err)
			if got != tt.permanent {
				t.Errorf("isPermanentAuthError(%v) = %v, want %v", tt.err, got, tt.permanent)
			}
		})
	}
}
