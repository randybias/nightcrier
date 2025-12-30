package k8s

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/randybias/nightcrier/internal/storage"
)

func TestGenerateOutputURLs(t *testing.T) {
	ctx := context.Background()

	// Create an in-memory object store for testing
	store, err := storage.NewObjectStore(ctx, "mem://", 1*time.Hour)
	if err != nil {
		t.Fatalf("Failed to create object store: %v", err)
	}
	defer store.Close()

	incidentID := "test-incident-123"
	jobTimeout := 10 * time.Minute

	// Note: mem:// provider doesn't support SignedURL, so we expect an error
	urls, err := GenerateOutputURLs(ctx, store, incidentID, jobTimeout)
	if err == nil {
		t.Fatal("Expected error from mem:// provider not supporting SignedURL")
	}
	if !strings.Contains(err.Error(), "not implemented") {
		t.Errorf("Expected 'not implemented' error, got: %v", err)
	}
	if urls != nil {
		t.Errorf("Expected nil urls on error, got: %+v", urls)
	}
}

func TestGenerateOutputURLs_NilStore(t *testing.T) {
	ctx := context.Background()

	_, err := GenerateOutputURLs(ctx, nil, "test-incident", 10*time.Minute)
	if err == nil {
		t.Error("Expected error for nil store, got nil")
	}
	if !strings.Contains(err.Error(), "object store cannot be nil") {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestGenerateOutputURLs_EmptyIncidentID(t *testing.T) {
	ctx := context.Background()

	store, err := storage.NewObjectStore(ctx, "mem://", 1*time.Hour)
	if err != nil {
		t.Fatalf("Failed to create object store: %v", err)
	}
	defer store.Close()

	_, err = GenerateOutputURLs(ctx, store, "", 10*time.Minute)
	if err == nil {
		t.Error("Expected error for empty incident ID, got nil")
	}
	if !strings.Contains(err.Error(), "incident ID cannot be empty") {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestGenerateOutputURLs_ZeroTimeout(t *testing.T) {
	ctx := context.Background()

	store, err := storage.NewObjectStore(ctx, "mem://", 1*time.Hour)
	if err != nil {
		t.Fatalf("Failed to create object store: %v", err)
	}
	defer store.Close()

	_, err = GenerateOutputURLs(ctx, store, "test-incident", 0)
	if err == nil {
		t.Error("Expected error for zero timeout, got nil")
	}
	if !strings.Contains(err.Error(), "job timeout must be greater than zero") {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestGenerateOutputURLs_CustomTimeout(t *testing.T) {
	ctx := context.Background()

	store, err := storage.NewObjectStore(ctx, "mem://", 1*time.Hour)
	if err != nil {
		t.Fatalf("Failed to create object store: %v", err)
	}
	defer store.Close()

	// Test with a custom timeout
	jobTimeout := 30 * time.Minute

	// Note: mem:// provider doesn't support SignedURL, so we expect an error
	urls, err := GenerateOutputURLs(ctx, store, "test-incident", jobTimeout)
	if err == nil {
		t.Fatal("Expected error from mem:// provider not supporting SignedURL")
	}
	if !strings.Contains(err.Error(), "not implemented") {
		t.Errorf("Expected 'not implemented' error, got: %v", err)
	}
	if urls != nil {
		t.Errorf("Expected nil urls on error, got: %+v", urls)
	}
}

func TestGenerateOutputURLs_URLFormat(t *testing.T) {
	ctx := context.Background()

	store, err := storage.NewObjectStore(ctx, "mem://", 1*time.Hour)
	if err != nil {
		t.Fatalf("Failed to create object store: %v", err)
	}
	defer store.Close()

	incidentID := "incident-with-special-chars-2024"
	jobTimeout := 10 * time.Minute

	// Note: mem:// provider doesn't support SignedURL, so we expect an error
	urls, err := GenerateOutputURLs(ctx, store, incidentID, jobTimeout)
	if err == nil {
		t.Fatal("Expected error from mem:// provider not supporting SignedURL")
	}
	if !strings.Contains(err.Error(), "not implemented") {
		t.Errorf("Expected 'not implemented' error, got: %v", err)
	}
	if urls != nil {
		t.Errorf("Expected nil urls on error, got: %+v", urls)
	}
}

func TestGenerateOutputURLs_AllFilenames(t *testing.T) {
	ctx := context.Background()

	store, err := storage.NewObjectStore(ctx, "mem://", 1*time.Hour)
	if err != nil {
		t.Fatalf("Failed to create object store: %v", err)
	}
	defer store.Close()

	incidentID := "test-incident"
	jobTimeout := 10 * time.Minute

	// Note: mem:// provider doesn't support SignedURL, so we expect an error
	urls, err := GenerateOutputURLs(ctx, store, incidentID, jobTimeout)
	if err == nil {
		t.Fatal("Expected error from mem:// provider not supporting SignedURL")
	}
	if !strings.Contains(err.Error(), "not implemented") {
		t.Errorf("Expected 'not implemented' error, got: %v", err)
	}
	if urls != nil {
		t.Errorf("Expected nil urls on error, got: %+v", urls)
	}
}

func TestOutputURLs_ToPresignedURLs(t *testing.T) {
	now := time.Now()
	outputURLs := &OutputURLs{
		Report:         "https://storage.example.com/report.md?sig=abc",
		ReportExpiry:   now.Add(1 * time.Hour),
		Log:            "https://storage.example.com/agent.log?sig=def",
		LogExpiry:      now.Add(1 * time.Hour),
		Session:        "https://storage.example.com/session.tar.gz?sig=ghi",
		SessionExpiry:  now.Add(1 * time.Hour),
		Result:         "https://storage.example.com/result.json?sig=jkl",
		ResultExpiry:   now.Add(1 * time.Hour),
		Commands:       "https://storage.example.com/commands-executed.log?sig=mno",
		CommandsExpiry: now.Add(1 * time.Hour),
	}

	presignedURLs := outputURLs.ToPresignedURLs()

	// Verify all URLs are copied correctly
	if presignedURLs.Report != outputURLs.Report {
		t.Errorf("Report URL mismatch: expected %q, got %q", outputURLs.Report, presignedURLs.Report)
	}
	if presignedURLs.Log != outputURLs.Log {
		t.Errorf("Log URL mismatch: expected %q, got %q", outputURLs.Log, presignedURLs.Log)
	}
	if presignedURLs.Session != outputURLs.Session {
		t.Errorf("Session URL mismatch: expected %q, got %q", outputURLs.Session, presignedURLs.Session)
	}
	if presignedURLs.Result != outputURLs.Result {
		t.Errorf("Result URL mismatch: expected %q, got %q", outputURLs.Result, presignedURLs.Result)
	}
	if presignedURLs.Commands != outputURLs.Commands {
		t.Errorf("Commands URL mismatch: expected %q, got %q", outputURLs.Commands, presignedURLs.Commands)
	}
}

func TestOutputURLs_ToPresignedURLs_EmptyURLs(t *testing.T) {
	outputURLs := &OutputURLs{}
	presignedURLs := outputURLs.ToPresignedURLs()

	// Verify empty URLs are handled correctly
	if presignedURLs.Report != "" {
		t.Errorf("Expected empty Report URL, got %q", presignedURLs.Report)
	}
	if presignedURLs.Log != "" {
		t.Errorf("Expected empty Log URL, got %q", presignedURLs.Log)
	}
	if presignedURLs.Session != "" {
		t.Errorf("Expected empty Session URL, got %q", presignedURLs.Session)
	}
	if presignedURLs.Result != "" {
		t.Errorf("Expected empty Result URL, got %q", presignedURLs.Result)
	}
	if presignedURLs.Commands != "" {
		t.Errorf("Expected empty Commands URL, got %q", presignedURLs.Commands)
	}
}
