package west

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// Project represents a buildable Zephyr project.
type Project struct {
	Name   string // Last path segment (display name)
	Path   string // Relative path from workspace root
	Source string // "manifest" or "local"
}

// ProjectsLoadedMsg is sent when the project list has been loaded.
type ProjectsLoadedMsg struct {
	Projects []Project
	Err      error
}

// skipDirs lists directories to skip during recursive scan.
var skipDirs = map[string]bool{
	".git":        true,
	"build":       true,
	"twister-out": true,
	".west":       true,
	"node_modules": true,
	"__pycache__": true,
}

// ListProjects discovers buildable Zephyr projects by scanning for CMakeLists.txt
// files that contain find_package(Zephyr. It scans the manifest project directory
// (the user's repo) to avoid finding hundreds of samples in the zephyr core.
func ListProjects(wsRoot string, manifestPath string) tea.Cmd {
	return func() tea.Msg {
		// Determine the scan root: the manifest project directory
		scanRoot := wsRoot
		if manifestPath != "" {
			scanRoot = filepath.Dir(manifestPath)
		}

		projects := discoverFromScan(scanRoot, wsRoot)

		sort.Slice(projects, func(i, j int) bool {
			return projects[i].Path < projects[j].Path
		})

		return ProjectsLoadedMsg{Projects: projects}
	}
}

func discoverFromScan(scanRoot string, wsRoot string) []Project {
	var projects []Project

	filepath.WalkDir(scanRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if d.IsDir() {
			if skipDirs[d.Name()] {
				return fs.SkipDir
			}
			return nil
		}

		if d.Name() != "CMakeLists.txt" {
			return nil
		}

		if containsZephyrPackage(path) {
			dir := filepath.Dir(path)
			relPath, err := filepath.Rel(wsRoot, dir)
			if err != nil {
				return nil
			}
			projects = append(projects, Project{
				Name:   filepath.Base(dir),
				Path:   relPath,
				Source: "local",
			})
		}

		return nil
	})

	return projects
}

func containsZephyrPackage(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	// Read first ~512 bytes to check for find_package(Zephyr
	buf := make([]byte, 512)
	n, _ := f.Read(buf)
	return strings.Contains(string(buf[:n]), "find_package(Zephyr")
}
