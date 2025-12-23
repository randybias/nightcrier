package skills

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureSkillsCached(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, "skills-cache")

	// Test 1: First call should clone the repository
	t.Run("clone on first call", func(t *testing.T) {
		err := EnsureSkillsCached(cacheDir)
		if err != nil {
			t.Skipf("skipping test: git clone failed (this is expected if network is unavailable): %v", err)
		}

		// Verify the triage script exists (note the nested structure: skills/k8s-troubleshooter/)
		triageScript := filepath.Join(cacheDir, K8sSkillName, "skills", K8sSkillName, "scripts", "incident_triage.sh")
		if _, err := os.Stat(triageScript); os.IsNotExist(err) {
			t.Errorf("expected triage script to exist at %s", triageScript)
		}
	})

	// Test 2: Second call should detect existing repository
	t.Run("skip clone on second call", func(t *testing.T) {
		// This should not clone again
		err := EnsureSkillsCached(cacheDir)
		if err != nil {
			t.Errorf("expected no error on second call, got: %v", err)
		}
	})
}

func TestEnsureSkillsCached_DefaultDir(t *testing.T) {
	// Test with empty cacheDir (should use default)
	t.Run("use default directory", func(t *testing.T) {
		// We can't actually test the cloning without side effects,
		// but we can verify the function handles an empty cacheDir
		// In a real scenario, this would create ./agent-home/skills
		// For this test, we just verify it doesn't panic
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("function panicked with empty cacheDir: %v", r)
			}
		}()

		// Call with empty cacheDir - this may fail due to permissions or network
		// but should not panic
		_ = EnsureSkillsCached("")
	})
}

func TestEnsureSkillsCached_CreatesCacheDir(t *testing.T) {
	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, "nested", "cache", "dir")

	// Verify the nested directory doesn't exist
	if _, err := os.Stat(cacheDir); !os.IsNotExist(err) {
		t.Fatalf("expected cache dir to not exist initially")
	}

	// Call EnsureSkillsCached - it should create the directory structure
	// Even if git clone fails, the directory should be created
	_ = EnsureSkillsCached(cacheDir)

	// Verify the directory was created
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		t.Errorf("expected cache directory to be created at %s", cacheDir)
	}
}
