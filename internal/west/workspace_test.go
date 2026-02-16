package west

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectWorkspace_WithWestDir(t *testing.T) {
	// Create a temp workspace with .west/config
	tmp := t.TempDir()
	westDir := filepath.Join(tmp, ".west")
	os.MkdirAll(westDir, 0o755)
	os.WriteFile(filepath.Join(westDir, "config"), []byte("[manifest]\npath = myapp\nfile = west.yml\n"), 0o644)

	// Create the manifest file
	appDir := filepath.Join(tmp, "myapp")
	os.MkdirAll(appDir, 0o755)
	os.WriteFile(filepath.Join(appDir, "west.yml"), []byte("manifest:\n"), 0o644)

	ws := DetectWorkspace(tmp)
	if ws == nil {
		t.Fatal("expected workspace to be found")
	}
	if ws.Root != tmp {
		t.Errorf("expected root=%s, got=%s", tmp, ws.Root)
	}
	if !ws.Initialized {
		t.Error("expected workspace to be initialized")
	}
	expected := filepath.Join(tmp, "myapp", "west.yml")
	if ws.ManifestPath != expected {
		t.Errorf("expected manifest=%s, got=%s", expected, ws.ManifestPath)
	}
}

func TestDetectWorkspace_Subdirectory(t *testing.T) {
	// Create workspace, then detect from a subdirectory
	tmp := t.TempDir()
	westDir := filepath.Join(tmp, ".west")
	os.MkdirAll(westDir, 0o755)
	os.WriteFile(filepath.Join(westDir, "config"), []byte("[manifest]\npath = app\n"), 0o644)

	subdir := filepath.Join(tmp, "app", "src")
	os.MkdirAll(subdir, 0o755)

	ws := DetectWorkspace(subdir)
	if ws == nil {
		t.Fatal("expected workspace to be found from subdirectory")
	}
	if ws.Root != tmp {
		t.Errorf("expected root=%s, got=%s", tmp, ws.Root)
	}
}

func TestDetectWorkspace_NotFound(t *testing.T) {
	tmp := t.TempDir()
	ws := DetectWorkspace(tmp)
	if ws != nil {
		t.Error("expected no workspace in empty directory")
	}
}

func TestDetectWorkspace_RealWorkspace(t *testing.T) {
	// Test against the actual Zephyr workspace if it exists
	wsPath := filepath.Join(os.Getenv("HOME"), "Projects", "zephyr-workspace")
	if _, err := os.Stat(filepath.Join(wsPath, ".west")); err != nil {
		t.Skip("no real Zephyr workspace found, skipping")
	}

	ws := DetectWorkspace(wsPath)
	if ws == nil {
		t.Fatal("expected real workspace to be detected")
	}
	if !ws.Initialized {
		t.Error("expected real workspace to be initialized")
	}
	t.Logf("Root: %s", ws.Root)
	t.Logf("Manifest: %s", ws.ManifestPath)
	t.Logf("Initialized: %v", ws.Initialized)
}
