# gust

A multi-paned TUI for Zephyr RTOS development with west.

![gust](https://img.shields.io/badge/Go-1.22-00ADD8?logo=go)
![License](https://img.shields.io/badge/license-MIT-blue)

## Features

- **9 Integrated Pages**: Workspace, Build, Flash, Monitor, Test, Artifacts, West, Config, Settings
- **Focus System**: Tab to toggle between sidebar navigation and page content
- **Board Management**: Searchable board list from `west boards` with fuzzy filtering
- **Streaming Output**: Real-time build/flash/test output with scrollable viewports
- **Serial Monitor**: Connect to devices, auto-scroll console, send input
- **History Tracking**: Persistent records of builds, flashes, tests, and serial sessions
- **Config Management**: Edit Kconfig symbols and app settings inline
- **Workspace Detection**: Automatically finds `.west/` directories walking up from CWD

## Installation

### From Source

```bash
git clone https://github.com/buckleypaul/gust.git
cd gust
go build ./cmd/gust
```

### Homebrew (coming soon)

```bash
brew install buckleypaul/tap/gust
```

## Usage

Navigate to a Zephyr workspace and run:

```bash
cd ~/zephyr-workspace
gust
```

### Navigation

**Sidebar (default focus):**
- `↑`/`↓` or `j`/`k` - Navigate pages
- `Enter` - Select page and move focus to content
- Sidebar shows **[FOCUSED]** label when active

**Content (when focused):**
- Page-specific keys (see status bar for hints)
- All interactive features like search, input fields, etc.

**Global Keys:**
- `Tab` - Toggle focus between sidebar and content
- `1-9` - Jump directly to any page
- `q` - Quit
- `?` - Toggle help

### Page-Specific Keys

**Build:**
- `/` - Search boards
- `b` or `Enter` - Build selected board
- `p` - Pristine build
- `c` - Clear output

**Flash:**
- `f` or `Enter` - Flash firmware
- `c` - Clear output

**Monitor:**
- `Enter` - Connect to selected port
- `d` - Disconnect
- `s` - Toggle auto-scroll
- `c` - Clear output
- Type and press `Enter` to send data

**Workspace:**
- `u` - Run west update
- `c` - Clear output

**West:**
- `Enter` - Run selected command
- `c` - Clear output

**Test:**
- `t` or `Enter` - Run tests
- `c` - Clear output

**Config:**
- `/` - Search Kconfig symbols
- `r` - Reload prj.conf

**Settings:**
- `Enter` or `e` - Edit selected setting
- `s` - Save settings to disk

**Artifacts:**
- `h`/`l` or `←`/`→` - Switch tabs (Builds, Flashes, Tests, Serial Logs)

## Configuration

### Global Config
`~/.config/gust/config.json`

### Workspace Config
`.gust/config.json` in your workspace root

### Settings
- `default_board` - Default board for builds
- `build_dir` - Build output directory (default: `build`)
- `serial_port` - Default serial port
- `serial_baud_rate` - Baud rate (default: 115200)
- `flash_runner` - Custom flash runner

Workspace config takes precedence over global config.

## Data Storage

All history and logs are stored in `.gust/` at your workspace root:

```
.gust/
├── config.json           # Workspace-specific config
├── history/
│   ├── builds.json      # Build records
│   ├── flashes.json     # Flash records
│   ├── tests.json       # Test records
│   └── serial_logs.json # Serial session metadata
└── logs/
    └── serial-*.log     # Serial output capture files
```

## Requirements

- Go 1.22 or later
- Zephyr RTOS workspace with west initialized
- For serial monitor: appropriate permissions for serial port access

## Development

### Running Tests

```bash
go test ./...
```

### Building

```bash
go build ./cmd/gust
```

### Project Structure

```
gust/
├── cmd/gust/           # Main entry point
├── internal/
│   ├── app/           # TUI app model, routing, layout
│   ├── config/        # Config system (global + workspace merge)
│   ├── pages/         # All 9 page implementations
│   ├── serial/        # Serial port management
│   ├── store/         # History persistence
│   ├── ui/            # Lipgloss styles and components
│   └── west/          # West command wrappers
└── .goreleaser.yml    # Release configuration
```

## License

MIT

## Contributing

Issues and pull requests welcome at [github.com/buckleypaul/gust](https://github.com/buckleypaul/gust).
