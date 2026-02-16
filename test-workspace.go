// +build ignore

// Quick test to verify workspace detection
package main

import (
	"fmt"
	"os"

	"github.com/buckleypaul/gust/internal/west"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run test-workspace.go <path>")
		os.Exit(1)
	}

	path := os.Args[1]
	fmt.Printf("Testing workspace detection from: %s\n\n", path)

	ws := west.DetectWorkspace(path)
	if ws == nil {
		fmt.Println("❌ No workspace detected")
		os.Exit(1)
	}

	fmt.Println("✅ Workspace detected!")
	fmt.Printf("   Root:        %s\n", ws.Root)
	fmt.Printf("   Manifest:    %s\n", ws.ManifestPath)
	fmt.Printf("   Initialized: %v\n", ws.Initialized)

	// Check if manifest file exists
	if _, err := os.Stat(ws.ManifestPath); err != nil {
		fmt.Printf("\n⚠️  Warning: Manifest file not accessible: %v\n", err)
	} else {
		fmt.Println("\n✅ Manifest file exists")
	}
}
