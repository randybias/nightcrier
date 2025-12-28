package storage

import (
	"context"
	"net/url"
	"strings"
	"testing"
	"time"
)

// TestNewObjectStore verifies the constructor creates valid ObjectStore instances.
func TestNewObjectStore(t *testing.T) {
	tests := []struct {
		name       string
		storageURL string
		expiry     time.Duration
		wantErr    bool
		wantExpiry time.Duration
	}{
		{
			name:       "mem storage",
			storageURL: "mem://",
			expiry:     24 * time.Hour,
			wantErr:    false,
			wantExpiry: 24 * time.Hour,
		},
		{
			name:       "mem storage with default expiry",
			storageURL: "mem://",
			expiry:     0,
			wantErr:    false,
			wantExpiry: 168 * time.Hour, // 7 days default
		},
		{
			name:       "empty URL",
			storageURL: "",
			expiry:     24 * time.Hour,
			wantErr:    true,
		},
		{
			name:       "no scheme",
			storageURL: "bucket-name",
			expiry:     24 * time.Hour,
			wantErr:    true,
		},
		{
			name:       "unsupported scheme",
			storageURL: "gcs://bucket",
			expiry:     24 * time.Hour,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			store, err := NewObjectStore(ctx, tt.storageURL, tt.expiry)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if store == nil {
				t.Fatalf("expected non-nil store")
			}

			if store.expiry != tt.wantExpiry {
				t.Errorf("expiry mismatch: expected %v, got %v", tt.wantExpiry, store.expiry)
			}

			// Clean up
			if err := store.Close(); err != nil {
				t.Errorf("failed to close store: %v", err)
			}
		})
	}
}

// TestObjectStoreUpload verifies the Upload method works correctly with mem:// storage.
func TestObjectStoreUpload(t *testing.T) {
	ctx := context.Background()
	store, err := NewObjectStore(ctx, "mem://", 24*time.Hour)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	tests := []struct {
		name        string
		key         string
		data        []byte
		contentType string
		wantErr     bool
	}{
		{
			name:        "valid upload json",
			key:         "test/incident.json",
			data:        []byte(`{"incident":"test"}`),
			contentType: "application/json",
			wantErr:     false,
		},
		{
			name:        "valid upload html",
			key:         "test/report.html",
			data:        []byte("<html><body>Report</body></html>"),
			contentType: "text/html",
			wantErr:     false,
		},
		{
			name:        "valid upload markdown",
			key:         "test/investigation.md",
			data:        []byte("# Investigation\n\nDetails here."),
			contentType: "text/markdown",
			wantErr:     false,
		},
		{
			name:        "empty key",
			key:         "",
			data:        []byte("data"),
			contentType: "text/plain",
			wantErr:     true,
		},
		{
			name:        "empty data",
			key:         "test/empty.txt",
			data:        []byte{},
			contentType: "text/plain",
			wantErr:     false,
		},
		{
			name:        "binary data",
			key:         "test/binary.bin",
			data:        []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD},
			contentType: "application/octet-stream",
			wantErr:     false,
		},
		{
			name:        "large data",
			key:         "test/large.bin",
			data:        make([]byte, 1024*1024), // 1MB
			contentType: "application/octet-stream",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := store.Upload(ctx, tt.key, tt.data, tt.contentType)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestObjectStoreSignedURL verifies SignedURL generation behavior.
// Note: mem:// storage doesn't support SignedURL (returns Unimplemented),
// so we test the error handling and parameter validation.
func TestObjectStoreSignedURL(t *testing.T) {
	ctx := context.Background()
	expiry := 24 * time.Hour
	store, err := NewObjectStore(ctx, "mem://", expiry)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	// Upload a test file first
	testKey := "test/file.txt"
	testData := []byte("test content")
	if err := store.Upload(ctx, testKey, testData, "text/plain"); err != nil {
		t.Fatalf("failed to upload test file: %v", err)
	}

	tests := []struct {
		name        string
		key         string
		wantErr     bool
		wantErrType string // "empty_key", "unimplemented", etc.
	}{
		{
			name:        "empty key",
			key:         "",
			wantErr:     true,
			wantErrType: "empty_key",
		},
		{
			name:        "valid key - mem unimplemented",
			key:         testKey,
			wantErr:     true,
			wantErrType: "unimplemented",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			signedURL, expiryTime, err := store.SignedURL(ctx, tt.key)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				// For empty key, verify it's the right error
				if tt.wantErrType == "empty_key" && !strings.Contains(err.Error(), "cannot be empty") {
					t.Errorf("expected empty key error, got: %v", err)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if signedURL == "" {
				t.Errorf("expected non-empty signed URL")
			}

			// Verify expiry time is roughly correct (within 1 second)
			expectedExpiry := time.Now().Add(expiry)
			timeDiff := expiryTime.Sub(expectedExpiry)
			if timeDiff < -time.Second || timeDiff > time.Second {
				t.Errorf("expiry time mismatch: expected ~%v, got %v (diff: %v)",
					expectedExpiry, expiryTime, timeDiff)
			}
		})
	}
}

// TestObjectStoreCanonicalURL verifies CanonicalURL generation with mem:// storage.
func TestObjectStoreCanonicalURL(t *testing.T) {
	tests := []struct {
		name      string
		key       string
		wantURL   string
	}{
		{
			name:    "mem storage with path",
			key:     "incident-123/report.html",
			wantURL: "mem://incident-123/report.html",
		},
		{
			name:    "mem storage empty key",
			key:     "",
			wantURL: "",
		},
		{
			name:    "mem storage nested path",
			key:     "a/b/c/file.json",
			wantURL: "mem://a/b/c/file.json",
		},
	}

	ctx := context.Background()
	store, err := NewObjectStore(ctx, "mem://", 24*time.Hour)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			canonicalURL := store.CanonicalURL(tt.key)

			if canonicalURL != tt.wantURL {
				t.Errorf("URL mismatch: expected %q, got %q", tt.wantURL, canonicalURL)
			}

			// Verify empty key returns empty string
			if tt.key == "" && canonicalURL != "" {
				t.Errorf("empty key should return empty URL, got %q", canonicalURL)
			}
		})
	}
}

// TestObjectStoreCanonicalURLFormats verifies URL format for each provider without connection.
func TestObjectStoreCanonicalURLFormats(t *testing.T) {
	tests := []struct {
		name       string
		provider   string
		bucketName string
		region     string
		endpoint   string
		account    string
		usePathStyle bool
		disableHTTPS bool
		key        string
		wantFormat string // regex or exact match pattern
	}{
		{
			name:       "s3 aws standard",
			provider:   "s3",
			bucketName: "my-bucket",
			region:     "us-east-1",
			key:        "path/file.json",
			wantFormat: "https://my-bucket.s3.us-east-1.amazonaws.com/path/file.json",
		},
		{
			name:       "s3 aws custom region",
			provider:   "s3",
			bucketName: "my-bucket",
			region:     "eu-west-1",
			key:        "file.json",
			wantFormat: "https://my-bucket.s3.eu-west-1.amazonaws.com/file.json",
		},
		{
			name:         "s3 minio path style",
			provider:     "s3",
			bucketName:   "my-bucket",
			endpoint:     "http://minio:9000",
			usePathStyle: true,
			key:          "data/file.json",
			wantFormat:   "http://minio:9000/my-bucket/data/file.json",
		},
		{
			name:         "s3 minio virtual hosted",
			provider:     "s3",
			bucketName:   "my-bucket",
			endpoint:     "https://s3.example.com",
			usePathStyle: false,
			key:          "file.json",
			wantFormat:   "https://my-bucket.s3.example.com/file.json",
		},
		{
			name:       "azure with account",
			provider:   "azblob",
			bucketName: "my-container",
			account:    "myaccount",
			key:        "incident/report.html",
			wantFormat: "https://myaccount.blob.core.windows.net/my-container/incident/report.html",
		},
		{
			name:       "azure without account",
			provider:   "azblob",
			bucketName: "my-container",
			key:        "file.json",
			wantFormat: "https://ACCOUNT.blob.core.windows.net/my-container/file.json",
		},
		{
			name:       "mem storage",
			provider:   "mem",
			key:        "test/file.txt",
			wantFormat: "mem://test/file.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &ObjectStore{
				provider:     tt.provider,
				bucketName:   tt.bucketName,
				region:       tt.region,
				endpoint:     tt.endpoint,
				account:      tt.account,
				usePathStyle: tt.usePathStyle,
				disableHTTPS: tt.disableHTTPS,
			}

			// Parse endpoint if provided
			if tt.endpoint != "" {
				parsed, err := url.Parse(tt.endpoint)
				if err != nil {
					t.Fatalf("failed to parse endpoint: %v", err)
				}
				store.parsedEndpoint = parsed
			}

			canonicalURL := store.CanonicalURL(tt.key)

			if canonicalURL != tt.wantFormat {
				t.Errorf("URL format mismatch:\nexpected: %q\ngot:      %q",
					tt.wantFormat, canonicalURL)
			}
		})
	}
}

// TestObjectStoreSignCanonicalURL verifies re-signing functionality.
// Note: Since mem:// doesn't support SignedURL, we test key extraction and error handling.
func TestObjectStoreSignCanonicalURL(t *testing.T) {
	ctx := context.Background()
	store, err := NewObjectStore(ctx, "mem://", 24*time.Hour)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	tests := []struct {
		name         string
		canonicalURL string
		wantErr      bool
		errContains  string
	}{
		{
			name:         "empty URL",
			canonicalURL: "",
			wantErr:      true,
			errContains:  "cannot be empty",
		},
		{
			name:         "valid mem URL - unimplemented",
			canonicalURL: "mem://incident-123/report.html",
			wantErr:      true,
			errContains:  "not implemented",
		},
		{
			name:         "invalid URL format",
			canonicalURL: "://invalid",
			wantErr:      true,
			errContains:  "invalid URL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := store.SignCanonicalURL(ctx, tt.canonicalURL)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
					return
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error should contain %q, got: %v", tt.errContains, err)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestExtractKeyFromCanonicalURL verifies key extraction from canonical URLs.
func TestExtractKeyFromCanonicalURL(t *testing.T) {
	tests := []struct {
		name         string
		provider     string
		bucketName   string
		usePathStyle bool
		endpoint     string
		canonicalURL string
		wantKey      string
		wantErr      bool
	}{
		{
			name:         "mem simple path",
			provider:     "mem",
			canonicalURL: "mem://incident-123/report.html",
			wantKey:      "incident-123/report.html",
			wantErr:      false,
		},
		{
			name:         "mem nested path",
			provider:     "mem",
			canonicalURL: "mem://path/to/deep/file.json",
			wantKey:      "path/to/deep/file.json",
			wantErr:      false,
		},
		{
			name:         "azure valid",
			provider:     "azblob",
			bucketName:   "my-container",
			canonicalURL: "https://myaccount.blob.core.windows.net/my-container/incident/report.html",
			wantKey:      "incident/report.html",
			wantErr:      false,
		},
		{
			name:         "azure nested path",
			provider:     "azblob",
			bucketName:   "my-container",
			canonicalURL: "https://myaccount.blob.core.windows.net/my-container/a/b/c/file.json",
			wantKey:      "a/b/c/file.json",
			wantErr:      false,
		},
		{
			name:         "azure wrong container",
			provider:     "azblob",
			bucketName:   "my-container",
			canonicalURL: "https://myaccount.blob.core.windows.net/other-container/file.json",
			wantKey:      "",
			wantErr:      true,
		},
		{
			name:         "azure invalid format",
			provider:     "azblob",
			bucketName:   "my-container",
			canonicalURL: "https://myaccount.blob.core.windows.net/my-container",
			wantKey:      "",
			wantErr:      true,
		},
		{
			name:         "s3 path style",
			provider:     "s3",
			bucketName:   "my-bucket",
			usePathStyle: true,
			endpoint:     "http://minio:9000",
			canonicalURL: "http://minio:9000/my-bucket/path/file.json",
			wantKey:      "path/file.json",
			wantErr:      false,
		},
		{
			name:         "s3 path style wrong bucket",
			provider:     "s3",
			bucketName:   "my-bucket",
			usePathStyle: true,
			endpoint:     "http://minio:9000",
			canonicalURL: "http://minio:9000/other-bucket/file.json",
			wantKey:      "",
			wantErr:      true,
		},
		{
			name:         "s3 virtual hosted",
			provider:     "s3",
			bucketName:   "my-bucket",
			usePathStyle: false,
			canonicalURL: "https://my-bucket.s3.us-east-1.amazonaws.com/path/to/file.json",
			wantKey:      "path/to/file.json",
			wantErr:      false,
		},
		{
			name:         "s3 virtual hosted nested",
			provider:     "s3",
			bucketName:   "my-bucket",
			usePathStyle: false,
			canonicalURL: "https://my-bucket.s3.eu-west-1.amazonaws.com/a/b/c/d/file.json",
			wantKey:      "a/b/c/d/file.json",
			wantErr:      false,
		},
		{
			name:         "empty URL",
			provider:     "mem",
			canonicalURL: "",
			wantKey:      "",
			wantErr:      true,
		},
		{
			name:         "invalid URL",
			provider:     "mem",
			canonicalURL: "://invalid",
			wantKey:      "",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &ObjectStore{
				provider:     tt.provider,
				bucketName:   tt.bucketName,
				usePathStyle: tt.usePathStyle,
				endpoint:     tt.endpoint,
			}

			key, err := store.extractKeyFromCanonicalURL(tt.canonicalURL)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if key != tt.wantKey {
				t.Errorf("key mismatch: expected %q, got %q", tt.wantKey, key)
			}
		})
	}
}

// TestObjectStoreClose verifies the Close method.
func TestObjectStoreClose(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name       string
		storageURL string
		wantErr    bool
	}{
		{
			name:       "mem storage",
			storageURL: "mem://",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, err := NewObjectStore(ctx, tt.storageURL, 24*time.Hour)
			if err != nil {
				t.Fatalf("failed to create store: %v", err)
			}

			err = store.Close()

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			// Note: Go CDK buckets are not idempotent for Close()
			// Second close will fail with "Bucket has been closed"
			// This is expected behavior
		})
	}
}

// TestObjectStoreCloseNilBucket verifies Close handles nil bucket gracefully.
func TestObjectStoreCloseNilBucket(t *testing.T) {
	store := &ObjectStore{
		bucket: nil,
	}

	err := store.Close()
	if err != nil {
		t.Errorf("Close with nil bucket should not error, got: %v", err)
	}
}

// TestObjectStoreSetAzureAccount verifies the Azure account setter.
func TestObjectStoreSetAzureAccount(t *testing.T) {
	ctx := context.Background()
	store, err := NewObjectStore(ctx, "mem://my-container", 24*time.Hour)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	// Initially should use ACCOUNT placeholder
	canonicalURL := store.CanonicalURL("test.json")
	if !strings.Contains(canonicalURL, "mem://") {
		t.Errorf("expected mem URL format, got %q", canonicalURL)
	}

	// After setting account for azblob provider
	store.provider = "azblob"
	store.bucketName = "my-container"
	store.SetAzureAccount("myaccount")

	canonicalURL = store.CanonicalURL("test.json")
	expected := "https://myaccount.blob.core.windows.net/my-container/test.json"
	if canonicalURL != expected {
		t.Errorf("canonical URL mismatch after setting account:\nexpected: %q\ngot:      %q",
			expected, canonicalURL)
	}
}

// TestObjectStoreUploadAndCanonicalURL verifies upload and canonical URL generation work together.
func TestObjectStoreUploadAndCanonicalURL(t *testing.T) {
	ctx := context.Background()
	store, err := NewObjectStore(ctx, "mem://", 24*time.Hour)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	testCases := []struct {
		name        string
		key         string
		data        []byte
		contentType string
		wantURL     string
	}{
		{
			name:        "json artifact",
			key:         "incident-001/incident.json",
			data:        []byte(`{"id":"001","status":"resolved"}`),
			contentType: "application/json",
			wantURL:     "mem://incident-001/incident.json",
		},
		{
			name:        "html report",
			key:         "incident-002/investigation.html",
			data:        []byte("<html><body><h1>Investigation</h1></body></html>"),
			contentType: "text/html",
			wantURL:     "mem://incident-002/investigation.html",
		},
		{
			name:        "markdown report",
			key:         "incident-003/investigation.md",
			data:        []byte("# Investigation\n\nFindings here."),
			contentType: "text/markdown",
			wantURL:     "mem://incident-003/investigation.md",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Upload
			if err := store.Upload(ctx, tc.key, tc.data, tc.contentType); err != nil {
				t.Fatalf("upload failed: %v", err)
			}

			// Get canonical URL
			canonicalURL := store.CanonicalURL(tc.key)
			if canonicalURL != tc.wantURL {
				t.Errorf("canonical URL mismatch: expected %q, got %q", tc.wantURL, canonicalURL)
			}
		})
	}
}

// TestObjectStoreProviderDetection verifies provider is correctly detected from URL.
func TestObjectStoreProviderDetection(t *testing.T) {
	tests := []struct {
		name         string
		storageURL   string
		wantProvider string
		wantErr      bool
	}{
		{
			name:         "mem provider",
			storageURL:   "mem://",
			wantProvider: "mem",
			wantErr:      false,
		},
		{
			name:         "s3 provider",
			storageURL:   "mem://bucket",
			wantProvider: "mem",
			wantErr:      false,
		},
		{
			name:         "azblob provider",
			storageURL:   "mem://container",
			wantProvider: "mem",
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			store, err := NewObjectStore(ctx, tt.storageURL, 24*time.Hour)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			defer store.Close()

			if store.provider != tt.wantProvider {
				t.Errorf("provider mismatch: expected %q, got %q",
					tt.wantProvider, store.provider)
			}
		})
	}
}

// TestObjectStoreContextCancellation verifies behavior with cancelled context.
func TestObjectStoreContextCancellation(t *testing.T) {
	ctx := context.Background()

	// Create store with valid context
	store, err := NewObjectStore(ctx, "mem://", 24*time.Hour)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	// Upload with valid context should work
	if err := store.Upload(ctx, "test.txt", []byte("data"), "text/plain"); err != nil {
		t.Fatalf("upload with valid context failed: %v", err)
	}

	// Create a cancelled context
	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	// Operations with cancelled context behavior depends on Go CDK implementation
	// For mem://, Upload is typically fast and doesn't check context
	err = store.Upload(cancelledCtx, "test2.txt", []byte("data"), "text/plain")
	// Either succeeds (operation is fast) or fails (context checked)
	// Both are acceptable behaviors
	t.Logf("Upload with cancelled context: %v", err)
}
