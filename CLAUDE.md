# gust - Zephyr RTOS TUI Development Guide

A multi-paned TUI for Zephyr RTOS development with west, built with Go and Charm libraries.

## Quick Start

```bash
# Build
make build

# Run tests
make test

# Run locally (requires Zephyr workspace)
cd ~/zephyr-workspace && ./gust

# Release (requires goreleaser)
make release
```

## Architecture

**Pattern**: Bubble Tea (Elm architecture) - `Update(msg) -> (Model, Cmd)`

```
internal/
├── app/       # Main TUI model, routing, focus management
├── config/    # Config system (merges global ~/.config/gust + workspace .gust/)
├── pages/     # 9 page implementations (Workspace, Build, Flash, Monitor, etc.)
├── serial/    # Serial port I/O (go.bug.st/serial)
├── store/     # History persistence (.gust/history/*.json)
├── ui/        # Lipgloss styles, reusable components
└── west/      # West CLI wrappers, workspace detection
```

## Key Files

- `internal/app/model.go` - Root app model, focus system (sidebar ↔ content)
- `internal/app/page.go` - Page interface definition
- `internal/pages/*.go` - Each page implements `Page` interface
- `internal/west/workspace.go` - Workspace detection (walks up for `.west/`)
- `.goreleaser.yml` - Release config (Darwin/Linux, amd64/arm64)
- `Makefile` - Build, test, install, release targets (handles CGO)

## Development Gotchas

### CGO Required
The serial library (`go.bug.st/serial`) requires CGO:
```bash
CGO_ENABLED=1 go build ./cmd/gust
```
This is why goreleaser sets `CGO_ENABLED=1` in builds.

### West Dependency
- Full functionality requires a Zephyr workspace with `west` installed
- Workspace detection: walks up from CWD looking for `.west/` directory
- For development without Zephyr: most features will error gracefully

### Focus System
Tab toggles between sidebar and content panes.
- Sidebar shows `[FOCUSED]` label when active
- Number keys (1-9) always jump to pages
- Page-specific keys only work when content is focused

## Testing

```bash
# Run all tests
go test ./...

# Test specific package
go test ./internal/west
go test ./internal/config
go test ./internal/store
```

**Test coverage:**
- `internal/west/workspace_test.go` - Workspace detection logic
- `internal/west/env_test.go` - West environment/SDK detection
- `internal/config/config_test.go` - Config merge behavior
- `internal/store/store_test.go` - History persistence

## Dependencies

**Core UI:**
- `github.com/charmbracelet/bubbletea` - TUI framework (Elm architecture)
- `github.com/charmbracelet/lipgloss` - Terminal styling
- `github.com/charmbracelet/bubbles` - Reusable UI components

**Serial:**
- `go.bug.st/serial` - Serial port I/O (requires CGO)

## Configuration

**Global**: `~/.config/gust/config.json`
**Workspace**: `.gust/config.json` (workspace takes precedence)

Workspace config overrides global settings. Both are gitignored.

## Data Storage

All workspace data stored in `.gust/` (gitignored):
```
.gust/
├── config.json              # Workspace settings
├── history/
│   ├── builds.json         # Build records
│   ├── flashes.json        # Flash records
│   ├── tests.json          # Test records
│   └── serial_logs.json    # Serial session metadata
└── logs/
    └── serial-*.log        # Serial output captures
```
