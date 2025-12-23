package skills

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
)

const (
	K8sSkillRepo = "https://github.com/randybias/k8s4agents"
	K8sSkillName = "k8s4agents"
)

// EnsureSkillsCached ensures required skills are cloned to the cache directory.
// If cacheDir is empty, it defaults to "./agent-home/skills".
// Returns an error if the cache directory cannot be created or git clone fails.
func EnsureSkillsCached(cacheDir string) error {
	if cacheDir == "" {
		cacheDir = "./agent-home/skills"
	}

	// Convert to absolute path for logging clarity
	absPath, err := filepath.Abs(cacheDir)
	if err != nil {
		slog.Warn("failed to get absolute path for cache directory, using relative",
			"path", cacheDir,
			"error", err)
		absPath = cacheDir
	}

	// Create cache directory if it doesn't exist
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create skills cache directory: %w", err)
	}

	// Check if k8s-troubleshooter skill exists
	skillPath := filepath.Join(cacheDir, K8sSkillName)
	// The repository structure has the actual skill in skills/k8s-troubleshooter/
	triageScript := filepath.Join(skillPath, "skills", K8sSkillName, "scripts", "incident_triage.sh")

	if _, err := os.Stat(triageScript); os.IsNotExist(err) {
		slog.Info("k8s skill not found, cloning from GitHub",
			"repo", K8sSkillRepo,
			"target", absPath)

		if err := cloneSkill(K8sSkillRepo, skillPath); err != nil {
			return fmt.Errorf("failed to clone k8s skill: %w", err)
		}

		slog.Info("k8s skill cached successfully", "path", absPath)
	} else if err != nil {
		// Some other error occurred when checking the file
		slog.Warn("error checking for existing skill, will attempt to continue",
			"path", triageScript,
			"error", err)
	} else {
		// Skill already exists
		slog.Debug("k8s skill already cached", "path", absPath)
	}

	return nil
}

func cloneSkill(repoURL, targetPath string) error {
	cmd := exec.Command("git", "clone", "--depth", "1", repoURL, targetPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git clone failed: %w", err)
	}

	return nil
}
