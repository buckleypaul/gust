package west

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// cmdEnv holds the modified environment for west commands.
// When nil, commands inherit the parent process environment.
var cmdEnv []string

// cmdDir holds the workspace root directory for west commands.
var cmdDir string

// westBinPath holds the resolved absolute path to the west binary.
// When set, applyEnv rewrites "west" commands to use this path,
// since exec.Command resolves via LookPath before Env is applied.
var westBinPath string

// InitEnv detects the Python virtual environment and prepends its bin/
// directory to PATH for all subsequent west command executions.
// Detection order: venvOverride → <workspace>/.venv/ → system west → auto-setup.
// Returns an error if auto-setup was attempted and failed.
func InitEnv(ws *Workspace, venvOverride string) error {
	if ws == nil {
		return nil
	}

	cmdDir = ws.Root

	candidates := []string{}
	if venvOverride != "" {
		candidates = append(candidates, venvOverride)
	}
	candidates = append(candidates, filepath.Join(ws.Root, ".venv"))

	for _, venv := range candidates {
		binDir := venvBinDir(venv)
		westBin := filepath.Join(binDir, westExeName())
		if _, err := os.Stat(westBin); err == nil {
			westBinPath = westBin
			cmdEnv = buildEnvWithPath(binDir)
			return nil
		}
	}

	// Check if west is available on the system PATH
	if _, err := exec.LookPath(westExeName()); err == nil {
		return nil
	}

	// Auto-setup: create venv and install west
	return setupVenv(ws.Root)
}

// setupVenv creates a Python virtual environment at <wsRoot>/.venv and installs west.
func setupVenv(wsRoot string) error {
	venvPath := filepath.Join(wsRoot, ".venv")

	fmt.Fprintf(os.Stderr, "west not found, creating venv at %s...\n", venvPath)
	if out, err := exec.Command("python3", "-m", "venv", venvPath).CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create venv: %w\n%s", err, out)
	}

	pipBin := filepath.Join(venvBinDir(venvPath), "pip")
	fmt.Fprintf(os.Stderr, "Installing west...\n")
	if out, err := exec.Command(pipBin, "install", "west").CombinedOutput(); err != nil {
		return fmt.Errorf("failed to install west: %w\n%s", err, out)
	}

	binDir := venvBinDir(venvPath)
	westBinPath = filepath.Join(binDir, westExeName())
	cmdEnv = buildEnvWithPath(binDir)
	fmt.Fprintf(os.Stderr, "west installed successfully.\n")
	return nil
}

// venvBinDir returns the bin (or Scripts on Windows) directory for a venv.
func venvBinDir(venvPath string) string {
	if runtime.GOOS == "windows" {
		return filepath.Join(venvPath, "Scripts")
	}
	return filepath.Join(venvPath, "bin")
}

// westExeName returns the west executable name for the current OS.
func westExeName() string {
	if runtime.GOOS == "windows" {
		return "west.exe"
	}
	return "west"
}

// buildEnvWithPath creates a copy of the current environment with binDir
// prepended to PATH.
func buildEnvWithPath(binDir string) []string {
	env := os.Environ()
	result := make([]string, 0, len(env))
	pathSet := false

	for _, e := range env {
		if strings.HasPrefix(e, "PATH=") {
			result = append(result, "PATH="+binDir+string(os.PathListSeparator)+e[5:])
			pathSet = true
		} else {
			result = append(result, e)
		}
	}

	if !pathSet {
		result = append(result, "PATH="+binDir)
	}

	return result
}

// applyEnv sets the environment, working directory, and resolved west binary path on an exec.Cmd.
func applyEnv(cmd *exec.Cmd) {
	if westBinPath != "" && filepath.Base(cmd.Path) == westExeName() {
		cmd.Path = westBinPath
		cmd.Err = nil
	}
	if cmdEnv != nil {
		cmd.Env = cmdEnv
	}
	if cmdDir != "" {
		cmd.Dir = cmdDir
	}
}
