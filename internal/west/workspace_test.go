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

func TestDetectWorkspace_WestDirInParentWithManifestInChild(t *testing.T) {
	// .west/ in parent, west.yml in child — detect from child should find parent
	tmp := t.TempDir()

	// Parent has .west/config pointing to child
	westDir := filepath.Join(tmp, ".west")
	os.MkdirAll(westDir, 0o755)
	os.WriteFile(filepath.Join(westDir, "config"), []byte("[manifest]\npath = child\n"), 0o644)

	// Child has west.yml
	childDir := filepath.Join(tmp, "child")
	os.MkdirAll(childDir, 0o755)
	os.WriteFile(filepath.Join(childDir, "west.yml"), []byte("manifest:\n"), 0o644)

	ws := DetectWorkspace(childDir)
	if ws == nil {
		t.Fatal("expected workspace to be found")
	}
	if ws.Root != tmp {
		t.Errorf("expected root=%s (parent with .west/), got=%s", tmp, ws.Root)
	}
	if !ws.Initialized {
		t.Error("expected workspace to be initialized (has .west/)")
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

func TestDetectWorkspace_RealFromSubdir(t *testing.T) {
	// Integration test: detect from hubble-device-sdk subdir, init env, run west
	wsPath := filepath.Join(os.Getenv("HOME"), "Projects", "zephyr-workspace")
	subdir := filepath.Join(wsPath, "hubble-device-sdk")
	if _, err := os.Stat(filepath.Join(wsPath, ".west")); err != nil {
		t.Skip("no real Zephyr workspace found, skipping")
	}

	ws := DetectWorkspace(subdir)
	if ws == nil {
		t.Fatal("expected workspace to be found from hubble-device-sdk")
	}
	if ws.Root != wsPath {
		t.Errorf("expected root=%s, got=%s", wsPath, ws.Root)
	}
	if !ws.Initialized {
		t.Error("expected workspace to be initialized")
	}

	// Reset and test InitEnv + RunSimple
	oldEnv, oldDir, oldBin := cmdEnv, cmdDir, westBinPath
	defer func() { cmdEnv = oldEnv; cmdDir = oldDir; westBinPath = oldBin }()
	cmdEnv = nil
	cmdDir = ""
	westBinPath = ""

	if err := InitEnv(ws, ""); err != nil {
		t.Fatalf("InitEnv error: %v", err)
	}
	t.Logf("westBinPath: %s", westBinPath)
	t.Logf("cmdDir: %s", cmdDir)

	out, err := RunSimple("west", "topdir")
	if err != nil {
		t.Fatalf("west topdir failed: %v", err)
	}
	t.Logf("west topdir: %s", out)
}
