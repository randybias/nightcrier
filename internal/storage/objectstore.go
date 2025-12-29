// Package storage provides cloud object storage abstraction via Go CDK.
package storage

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"gocloud.dev/blob"
	_ "gocloud.dev/blob/azureblob" // Azure Blob Storage driver
	_ "gocloud.dev/blob/memblob"   // In-memory storage for testing
	_ "gocloud.dev/blob/s3blob"    // S3/MinIO/RustFS driver
)

// ObjectStore wraps a Go CDK blob.Bucket and provides domain-specific operations
// for uploading artifacts and generating signed/canonical URLs.
type ObjectStore struct {
	bucket   *blob.Bucket
	provider string // Provider type: "azblob", "s3", "mem"
	expiry   time.Duration

	// Provider-specific fields for URL generation
	bucketName     string // S3 bucket or Azure container name
	endpoint       string // Custom endpoint (S3-compatible services)
	account        string // Azure storage account name
	region         string // AWS region
	usePathStyle   bool   // S3 path-style addressing
	disableHTTPS   bool   // Allow HTTP (development only)
	parsedEndpoint *url.URL
}

// ObjectStoreConfig holds configuration for creating an ObjectStore.
type ObjectStoreConfig struct {
	// URL is the Go CDK storage URL (e.g., "azblob://container", "s3://bucket?region=us-east-1")
	URL string
	// SignedURLExpiry is how long signed URLs remain valid
	SignedURLExpiry time.Duration
}

// NewObjectStore creates a new ObjectStore by opening a Go CDK blob.Bucket.
// The URL scheme determines the provider (azblob://, s3://, mem://).
// Provider-specific metadata is extracted from the URL for canonical URL generation.
func NewObjectStore(ctx context.Context, storageURL string, expiry time.Duration) (*ObjectStore, error) {
	if storageURL == "" {
		return nil, fmt.Errorf("storage URL is required")
	}

	if expiry == 0 {
		expiry = 168 * time.Hour // Default 7 days
	}

	// Parse URL to extract provider and metadata
	parsedURL, err := url.Parse(storageURL)
	if err != nil {
		return nil, fmt.Errorf("invalid storage URL: %w", err)
	}

	provider := parsedURL.Scheme
	if provider == "" {
		return nil, fmt.Errorf("storage URL must have a scheme (azblob://, s3://, mem://)")
	}

	store := &ObjectStore{
		provider: provider,
		expiry:   expiry,
	}

	// Extract provider-specific metadata for URL generation
	switch provider {
	case "azblob":
		// URL format: azblob://container
		store.bucketName = parsedURL.Host
		// Azure account name comes from AZURE_STORAGE_ACCOUNT env var or credentials
		// We'll need to extract it from the resolved URL after opening
	case "s3":
		// URL format: s3://bucket?region=us-east-1&endpoint=http://minio:9000
		store.bucketName = parsedURL.Host
		query := parsedURL.Query()
		store.region = query.Get("region")
		if store.region == "" {
			store.region = "us-east-1" // Default region
		}
		endpointStr := query.Get("endpoint")
		if endpointStr != "" {
			store.endpoint = endpointStr
			parsed, err := url.Parse(endpointStr)
			if err != nil {
				return nil, fmt.Errorf("invalid endpoint URL: %w", err)
			}
			store.parsedEndpoint = parsed
		}
		store.usePathStyle = query.Get("use_path_style") == "true"
		store.disableHTTPS = query.Get("disable_https") == "true"
	case "mem":
		// URL format: mem://
		// No metadata needed for in-memory storage
	default:
		return nil, fmt.Errorf("unsupported storage provider: %s (supported: azblob, s3, mem)", provider)
	}

	// Open the bucket using Go CDK
	bucket, err := blob.OpenBucket(ctx, storageURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open storage bucket: %w", err)
	}

	store.bucket = bucket

	return store, nil
}

// Upload writes data to the object store at the specified key with the given content type.
func (s *ObjectStore) Upload(ctx context.Context, key string, data []byte, contentType string) error {
	if key == "" {
		return fmt.Errorf("key cannot be empty")
	}

	// Set up write options with content type
	opts := &blob.WriterOptions{
		ContentType: contentType,
		// Set content disposition to inline for browser rendering
		ContentDisposition: "inline",
	}

	// Create a writer for the key
	writer, err := s.bucket.NewWriter(ctx, key, opts)
	if err != nil {
		return fmt.Errorf("failed to create writer for key %s: %w", key, err)
	}

	// Write the data
	if _, err := writer.Write(data); err != nil {
		writer.Close()
		return fmt.Errorf("failed to write data for key %s: %w", key, err)
	}

	// Close the writer to finalize the upload
	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to finalize upload for key %s: %w", key, err)
	}

	return nil
}

// SignedURL generates a temporary signed URL for accessing the object.
// Returns the signed URL and its expiration time.
func (s *ObjectStore) SignedURL(ctx context.Context, key string) (string, time.Time, error) {
	if key == "" {
		return "", time.Time{}, fmt.Errorf("key cannot be empty")
	}

	expiryTime := time.Now().Add(s.expiry)

	opts := &blob.SignedURLOptions{
		Expiry: s.expiry,
		Method: "GET",
	}

	signedURL, err := s.bucket.SignedURL(ctx, key, opts)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to generate signed URL for key %s: %w", key, err)
	}

	return signedURL, expiryTime, nil
}

// SignedPutURL generates a temporary signed URL for uploading to the object.
// Returns the signed URL and its expiration time.
// The URL is valid for PUT operations only, providing write-only access.
func (s *ObjectStore) SignedPutURL(ctx context.Context, key string, expiry time.Duration) (string, time.Time, error) {
	if key == "" {
		return "", time.Time{}, fmt.Errorf("key cannot be empty")
	}

	if expiry == 0 {
		expiry = s.expiry
	}

	expiryTime := time.Now().Add(expiry)

	opts := &blob.SignedURLOptions{
		Expiry: expiry,
		Method: "PUT",
	}

	signedURL, err := s.bucket.SignedURL(ctx, key, opts)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to generate signed PUT URL for key %s: %w", key, err)
	}

	return signedURL, expiryTime, nil
}

// CanonicalURL returns the base URL for the object without authentication.
// This URL is stable and can be stored long-term, but may not be directly accessible
// depending on bucket permissions. Use SignCanonicalURL to add temporary authentication.
func (s *ObjectStore) CanonicalURL(key string) string {
	if key == "" {
		return ""
	}

	switch s.provider {
	case "azblob":
		// Azure format: https://{account}.blob.core.windows.net/{container}/{key}
		// Note: account name should be extracted from credentials/config
		if s.account == "" {
			// Fallback if account not set
			return fmt.Sprintf("https://ACCOUNT.blob.core.windows.net/%s/%s", s.bucketName, key)
		}
		return fmt.Sprintf("https://%s.blob.core.windows.net/%s/%s", s.account, s.bucketName, key)

	case "s3":
		if s.endpoint != "" {
			// Custom endpoint (MinIO, RustFS, etc.)
			if s.usePathStyle {
				// Path-style: http://endpoint/bucket/key
				scheme := "https"
				if s.disableHTTPS {
					scheme = "http"
				}
				if s.parsedEndpoint != nil {
					// Use the scheme from the parsed endpoint
					scheme = s.parsedEndpoint.Scheme
					return fmt.Sprintf("%s://%s/%s/%s", scheme, s.parsedEndpoint.Host, s.bucketName, key)
				}
				return fmt.Sprintf("%s/%s/%s", s.endpoint, s.bucketName, key)
			}
			// Virtual-hosted-style: http://bucket.endpoint/key
			if s.parsedEndpoint != nil {
				scheme := s.parsedEndpoint.Scheme
				return fmt.Sprintf("%s://%s.%s/%s", scheme, s.bucketName, s.parsedEndpoint.Host, key)
			}
			return fmt.Sprintf("%s/%s", s.endpoint, key)
		}
		// AWS S3: https://bucket.s3.region.amazonaws.com/key
		return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", s.bucketName, s.region, key)

	case "mem":
		// In-memory storage doesn't have real URLs
		return fmt.Sprintf("mem://%s", key)

	default:
		return key
	}
}

// SignCanonicalURL generates a signed URL from a canonical URL.
// This is useful for re-signing expired URLs when you have the canonical URL stored.
func (s *ObjectStore) SignCanonicalURL(ctx context.Context, canonicalURL string) (string, time.Time, error) {
	if canonicalURL == "" {
		return "", time.Time{}, fmt.Errorf("canonical URL cannot be empty")
	}

	// Extract the key from the canonical URL
	key, err := s.extractKeyFromCanonicalURL(canonicalURL)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to extract key from canonical URL: %w", err)
	}

	// Generate a signed URL for the key
	return s.SignedURL(ctx, key)
}

// extractKeyFromCanonicalURL extracts the object key from a canonical URL.
// This handles provider-specific URL formats.
func (s *ObjectStore) extractKeyFromCanonicalURL(canonicalURL string) (string, error) {
	if canonicalURL == "" {
		return "", fmt.Errorf("canonical URL cannot be empty")
	}

	parsed, err := url.Parse(canonicalURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}

	switch s.provider {
	case "azblob":
		// Azure format: https://{account}.blob.core.windows.net/{container}/{key}
		// Path: /{container}/{key}
		path := strings.TrimPrefix(parsed.Path, "/")
		parts := strings.SplitN(path, "/", 2)
		if len(parts) < 2 {
			return "", fmt.Errorf("invalid Azure blob URL format: %s", canonicalURL)
		}
		// Verify container matches
		if parts[0] != s.bucketName {
			return "", fmt.Errorf("container mismatch: URL has %s, expected %s", parts[0], s.bucketName)
		}
		return parts[1], nil

	case "s3":
		// Path-style: http://endpoint/bucket/key -> /bucket/key
		// Virtual-hosted: http://bucket.endpoint/key -> /key
		path := strings.TrimPrefix(parsed.Path, "/")

		if s.usePathStyle || s.endpoint != "" {
			// Path-style: first segment is bucket
			parts := strings.SplitN(path, "/", 2)
			if len(parts) < 2 {
				return "", fmt.Errorf("invalid S3 path-style URL format: %s", canonicalURL)
			}
			// Verify bucket matches
			if parts[0] != s.bucketName {
				return "", fmt.Errorf("bucket mismatch: URL has %s, expected %s", parts[0], s.bucketName)
			}
			return parts[1], nil
		}

		// Virtual-hosted-style: path is the key
		return path, nil

	case "mem":
		// mem://key
		return strings.TrimPrefix(canonicalURL, "mem://"), nil

	default:
		return "", fmt.Errorf("unsupported provider for key extraction: %s", s.provider)
	}
}

// Download reads data from the object store at the specified key.
// This is used to retrieve artifacts that were uploaded by the agent container.
func (s *ObjectStore) Download(ctx context.Context, key string) ([]byte, error) {
	if key == "" {
		return nil, fmt.Errorf("key cannot be empty")
	}

	reader, err := s.bucket.NewReader(ctx, key, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create reader for key %s: %w", key, err)
	}
	defer reader.Close()

	// Read all data from the reader
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read data for key %s: %w", key, err)
	}

	return data, nil
}

// SetAzureAccount sets the Azure storage account name for canonical URL generation.
// This should be called after construction if the account name isn't available from the URL.
func (s *ObjectStore) SetAzureAccount(account string) {
	s.account = account
}

// Bucket returns the underlying blob.Bucket for advanced operations.
// This is useful for operations not covered by the ObjectStore interface.
func (s *ObjectStore) Bucket() *blob.Bucket {
	return s.bucket
}

// Close closes the underlying bucket and releases any resources.
func (s *ObjectStore) Close() error {
	if s.bucket == nil {
		return nil
	}
	return s.bucket.Close()
}
