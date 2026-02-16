package west

import (
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

// InitEnv detects the Python virtual environment and prepends its bin/
// directory to PATH for all subsequent west command executions.
// Detection order: venvOverride → <workspace>/.venv/ → system PATH (no modification).
func InitEnv(ws *Workspace, venvOverride string) {
	if ws == nil {
		return
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
			cmdEnv = buildEnvWithPath(binDir)
			return
		}
	}
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

// applyEnv sets the environment and working directory on an exec.Cmd.
func applyEnv(cmd *exec.Cmd) {
	if cmdEnv != nil {
		cmd.Env = cmdEnv
	}
	if cmdDir != "" {
		cmd.Dir = cmdDir
	}
}
