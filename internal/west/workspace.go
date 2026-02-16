package west

import (
	"os"
	"path/filepath"
)

// Workspace holds information about a detected Zephyr west workspace.
type Workspace struct {
	Root         string // Absolute path to the workspace root (directory containing west.yml)
	ManifestPath string // Path to west.yml
	Initialized  bool   // Whether .west/ directory exists
}

// DetectWorkspace walks up from startDir looking for west.yml.
// Returns a Workspace if found, or nil if no west.yml is found.
func DetectWorkspace(startDir string) *Workspace {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return nil
	}

	for {
		manifest := filepath.Join(dir, "west.yml")
		if _, err := os.Stat(manifest); err == nil {
			ws := &Workspace{
				Root:         dir,
				ManifestPath: manifest,
			}
			// Check if .west/ directory exists (workspace is initialized)
			westDir := filepath.Join(dir, ".west")
			if info, err := os.Stat(westDir); err == nil && info.IsDir() {
				ws.Initialized = true
			}
			return ws
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break // reached filesystem root
		}
		dir = parent
	}
	return nil
}
