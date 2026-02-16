package west

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestInitEnvDetectsVenv(t *testing.T) {
	// Reset package state
	cmdEnv = nil
	cmdDir = ""
	defer func() { cmdEnv = nil; cmdDir = "" }()

	// Create a temp workspace with a fake venv
	tmp := t.TempDir()
	wsRoot := tmp

	binDir := filepath.Join(wsRoot, ".venv", "bin")
	westBin := "west"
	if runtime.GOOS == "windows" {
		binDir = filepath.Join(wsRoot, ".venv", "Scripts")
		westBin = "west.exe"
	}

	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(binDir, westBin), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	ws := &Workspace{Root: wsRoot, Initialized: true}
	InitEnv(ws, "")

	if cmdDir != wsRoot {
		t.Errorf("cmdDir = %q, want %q", cmdDir, wsRoot)
	}
	if cmdEnv == nil {
		t.Fatal("cmdEnv is nil, expected it to be set")
	}

	// Verify PATH contains the venv bin dir
	for _, e := range cmdEnv {
		if strings.HasPrefix(e, "PATH=") {
			if !strings.HasPrefix(e[5:], binDir) {
				t.Errorf("PATH does not start with venv bin dir\nPATH=%s", e[5:])
			}
			return
		}
	}
	t.Error("PATH not found in cmdEnv")
}

func TestInitEnvOverrideTakesPrecedence(t *testing.T) {
	cmdEnv = nil
	cmdDir = ""
	defer func() { cmdEnv = nil; cmdDir = "" }()

	tmp := t.TempDir()
	wsRoot := tmp

	// Create both default and override venvs
	for _, dir := range []string{".venv", "custom-venv"} {
		binDir := filepath.Join(wsRoot, dir, "bin")
		westBin := "west"
		if runtime.GOOS == "windows" {
			binDir = filepath.Join(wsRoot, dir, "Scripts")
			westBin = "west.exe"
		}
		if err := os.MkdirAll(binDir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(binDir, westBin), []byte("#!/bin/sh\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	ws := &Workspace{Root: wsRoot, Initialized: true}
	overridePath := filepath.Join(wsRoot, "custom-venv")
	InitEnv(ws, overridePath)

	expectedBinDir := filepath.Join(overridePath, "bin")
	if runtime.GOOS == "windows" {
		expectedBinDir = filepath.Join(overridePath, "Scripts")
	}

	for _, e := range cmdEnv {
		if strings.HasPrefix(e, "PATH=") {
			if !strings.HasPrefix(e[5:], expectedBinDir) {
				t.Errorf("PATH should start with override bin dir %q\nPATH=%s", expectedBinDir, e[5:])
			}
			return
		}
	}
	t.Error("PATH not found in cmdEnv")
}

func TestInitEnvFallbackNoVenv(t *testing.T) {
	cmdEnv = nil
	cmdDir = ""
	defer func() { cmdEnv = nil; cmdDir = "" }()

	tmp := t.TempDir()
	ws := &Workspace{Root: tmp, Initialized: true}

	InitEnv(ws, "")

	if cmdEnv != nil {
		t.Error("cmdEnv should be nil when no venv exists")
	}
	if cmdDir != tmp {
		t.Errorf("cmdDir = %q, want %q", cmdDir, tmp)
	}
}

func TestInitEnvNilWorkspace(t *testing.T) {
	cmdEnv = nil
	cmdDir = ""
	defer func() { cmdEnv = nil; cmdDir = "" }()

	InitEnv(nil, "")

	if cmdEnv != nil {
		t.Error("cmdEnv should be nil for nil workspace")
	}
	if cmdDir != "" {
		t.Error("cmdDir should be empty for nil workspace")
	}
}
