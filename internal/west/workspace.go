package west

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Workspace holds information about a detected Zephyr west workspace.
type Workspace struct {
	Root         string // Absolute path to the workspace root (parent of .west/)
	ManifestPath string // Path to the west.yml manifest file
	Initialized  bool   // Whether .west/ directory exists with config
}

// WorkspaceHealth tracks the status of all required workspace components.
type WorkspaceHealth struct {
	BrewDepsOK      bool // Required Homebrew packages installed
	WestInitialized bool // .west/ directory exists
	ModulesUpdated  bool // Zephyr and modules exist
	ZephyrExported  bool // CMake package registry has Zephyr
	PythonDepsOK    bool // Python dependencies installed
	SdkInstalled    bool // Zephyr SDK is available
}

// DetectWorkspace walks up from startDir looking for a .west/ directory,
// which is the standard marker for an initialized west workspace.
// Falls back to looking for west.yml directly.
func DetectWorkspace(startDir string) *Workspace {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return nil
	}

	var manifestCandidate string

	for {
		// Primary check: look for .west/ directory (standard west workspace)
		westDir := filepath.Join(dir, ".west")
		if info, err := os.Stat(westDir); err == nil && info.IsDir() {
			ws := &Workspace{
				Root:        dir,
				Initialized: true,
			}
			ws.ManifestPath = ResolveManifest(dir)
			return ws
		}

		// Record west.yml as fallback candidate, but keep walking
		if manifestCandidate == "" {
			manifest := filepath.Join(dir, "west.yml")
			if _, err := os.Stat(manifest); err == nil {
				manifestCandidate = manifest
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break // reached filesystem root
		}
		dir = parent
	}

	if manifestCandidate != "" {
		return &Workspace{
			Root:         filepath.Dir(manifestCandidate),
			ManifestPath: manifestCandidate,
			Initialized:  false,
		}
	}
	return nil
}

// ResolveManifest reads .west/config to find the manifest path.
func ResolveManifest(root string) string {
	configPath := filepath.Join(root, ".west", "config")
	f, err := os.Open(configPath)
	if err != nil {
		return ""
	}
	defer f.Close()

	var manifestDir, manifestFile string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "path") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				manifestDir = strings.TrimSpace(parts[1])
			}
		}
		if strings.HasPrefix(line, "file") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				manifestFile = strings.TrimSpace(parts[1])
			}
		}
	}

	if manifestDir == "" {
		return ""
	}
	if manifestFile == "" {
		manifestFile = "west.yml"
	}

	return filepath.Join(root, manifestDir, manifestFile)
}

// CheckHealth performs health checks on the workspace to detect which setup steps are complete.
func (w *Workspace) CheckHealth() WorkspaceHealth {
	health := WorkspaceHealth{
		WestInitialized: w.Initialized,
		BrewDepsOK:      checkBrewDeps(),
	}

	if w.Root == "" {
		return health
	}

	// Check if zephyr directory exists (indicates modules were updated)
	zephyrPath := filepath.Join(w.Root, "zephyr")
	if info, err := os.Stat(zephyrPath); err == nil && info.IsDir() {
		health.ModulesUpdated = true
	}

	// Check if CMake package registry has Zephyr (indicates zephyr-export was run)
	// Look for ~/.cmake/packages/Zephyr/
	homeDir, err := os.UserHomeDir()
	if err == nil {
		cmakePackage := filepath.Join(homeDir, ".cmake", "packages", "Zephyr")
		if info, err := os.Stat(cmakePackage); err == nil && info.IsDir() {
			health.ZephyrExported = true
		}
	}

	// Check if Python requirements are installed
	// Look for common Python packages in the venv
	venvPath := filepath.Join(w.Root, ".venv", "lib")
	if info, err := os.Stat(venvPath); err == nil && info.IsDir() {
		// If venv/lib exists and has content, assume deps are installed
		// More sophisticated check could verify specific packages
		health.PythonDepsOK = true
	}

	// Check if Zephyr SDK is installed
	health.SdkInstalled = checkSdkInstalled()

	return health
}

// checkSdkInstalled checks if the Zephyr SDK is available.
// It checks:
// 1. ZEPHYR_SDK_INSTALL_DIR environment variable
// 2. Common installation locations
func checkSdkInstalled() bool {
	// Check environment variable first
	if sdkDir := os.Getenv("ZEPHYR_SDK_INSTALL_DIR"); sdkDir != "" {
		if info, err := os.Stat(sdkDir); err == nil && info.IsDir() {
			return true
		}
	}

	// Check common installation locations
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return false
	}

	locations := []string{
		filepath.Join(homeDir, "zephyr-sdk"),
		filepath.Join("/opt", "zephyr-sdk"),
	}

	// Also check for versioned installations (e.g., ~/zephyr-sdk-0.17.4)
	if entries, err := os.ReadDir(homeDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() && strings.HasPrefix(entry.Name(), "zephyr-sdk-") {
				locations = append(locations, filepath.Join(homeDir, entry.Name()))
			}
		}
	}

	for _, loc := range locations {
		if info, err := os.Stat(loc); err == nil && info.IsDir() {
			// Verify it's a real SDK by checking for cmake/ subdirectory
			cmakeDir := filepath.Join(loc, "cmake")
			if cmakeInfo, err := os.Stat(cmakeDir); err == nil && cmakeInfo.IsDir() {
				return true
			}
		}
	}

	return false
}

// checkBrewDeps checks if all required Homebrew packages are installed.
func checkBrewDeps() bool {
	// Check if brew is available
	brewCmd, err := exec.LookPath("brew")
	if err != nil {
		return false
	}

	// List of required packages (imported from commands.go conceptually, but duplicated here for simplicity)
	required := []string{
		"cmake", "ninja", "gperf", "python3", "python-tk",
		"ccache", "qemu", "dtc", "libmagic", "wget", "openocd",
	}

	// Check each package
	for _, pkg := range required {
		cmd := exec.Command(brewCmd, "list", pkg)
		if err := cmd.Run(); err != nil {
			return false // At least one package is missing
		}
	}

	return true // All packages are installed
}
