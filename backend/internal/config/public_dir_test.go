package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolvePublicDir(t *testing.T) {
	t.Run("prefers built frontend dist when present", func(t *testing.T) {
		baseDir := t.TempDir()
		frontendDist := filepath.Join(baseDir, "frontend", "dist")
		if err := os.MkdirAll(frontendDist, 0o755); err != nil {
			t.Fatalf("prepare dir: %v", err)
		}

		resolved, err := ResolvePublicDir(baseDir, "./public", "./frontend/dist")
		if err != nil {
			t.Fatalf("ResolvePublicDir returned error: %v", err)
		}
		if resolved != filepath.Join(baseDir, "frontend", "dist") {
			t.Fatalf("expected %q, got %q", filepath.Join(baseDir, "frontend", "dist"), resolved)
		}
	})

	t.Run("returns error when no candidate exists", func(t *testing.T) {
		baseDir := t.TempDir()

		if _, err := ResolvePublicDir(baseDir, "./public", "./frontend/dist"); err == nil {
			t.Fatal("expected error when no public dir exists")
		}
	})
}
