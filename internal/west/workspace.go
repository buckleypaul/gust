package west

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// Workspace holds information about a detected Zephyr west workspace.
type Workspace struct {
	Root         string // Absolute path to the workspace root (parent of .west/)
	ManifestPath string // Path to the west.yml manifest file
	Initialized  bool   // Whether .west/ directory exists with config
}

// DetectWorkspace walks up from startDir looking for a .west/ directory,
// which is the standard marker for an initialized west workspace.
// Falls back to looking for west.yml directly.
func DetectWorkspace(startDir string) *Workspace {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return nil
	}

	for {
		// Primary check: look for .west/ directory (standard west workspace)
		westDir := filepath.Join(dir, ".west")
		if info, err := os.Stat(westDir); err == nil && info.IsDir() {
			ws := &Workspace{
				Root:        dir,
				Initialized: true,
			}
			// Parse .west/config to find the manifest path
			ws.ManifestPath = resolveManifest(dir)
			return ws
		}

		// Fallback: look for west.yml directly (uninitialised workspace)
		manifest := filepath.Join(dir, "west.yml")
		if _, err := os.Stat(manifest); err == nil {
			return &Workspace{
				Root:         dir,
				ManifestPath: manifest,
				Initialized:  false,
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break // reached filesystem root
		}
		dir = parent
	}
	return nil
}

// resolveManifest reads .west/config to find the manifest path.
func resolveManifest(root string) string {
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
