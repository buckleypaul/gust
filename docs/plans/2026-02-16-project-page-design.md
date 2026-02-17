# Project Page Design

## Summary

Add a dedicated Project page that centralizes project configuration: project selection, board, shield, Kconfig editing (prj.conf), and board overlay awareness. Replaces the existing ConfigPage. Simplifies the Build page by moving hardware config out.

## Sidebar Order

```
1. Workspace
2. Project    (NEW - replaces Config)
3. Build      (simplified)
4. Flash
5. Monitor
6. Test
7. Artifacts
8. West
9. Settings
```

## Top Bar

Expands to show both project and board:

```
Project: samples/ble-beacon  Board: nrf52840dk/nrf52840  [p] change
```

Board shows `(none)` when unset. `[p] change` hint appears when sidebar is focused.

## Project Page Layout

Sectioned form with three areas, following the Build page's existing form pattern.

```
Project   samples/zephyr/ble-beacon  [p]
Board     [nrf52840dk/nrf52840          ▾]
Shield    [                              ]

── Kconfig (prj.conf) ─── / search ──────
CONFIG_BT=y
CONFIG_BT_PERIPHERAL=y
CONFIG_LOG=y
CONFIG_LOG_DEFAULT_LEVEL=3

── Board Overlay (nrf52840dk) ───────────
boards/nrf52840dk.conf:
  CONFIG_SERIAL=y
  CONFIG_UART_CONSOLE=y
boards/nrf52840dk.overlay: (exists)

4 symbols  |  2 overlay entries
```

### Hardware Section

- **Project**: displays current project path. `p` key opens the picker overlay (reuses existing Picker component and `west.ListProjects` discovery).
- **Board**: type-ahead dropdown with fuzzy search (moved from Build page). Uses existing `west.ListBoards()` for discovery.
- **Shield**: text input (moved from Build page).
- `tab` cycles between fields within this section.

### Kconfig Section

Absorbs the existing ConfigPage functionality:

- Loads and parses `<project>/prj.conf` using existing `parsePrjConf()`.
- `/` activates search/filter (same as current ConfigPage).
- `e` edits the selected entry's value inline.
- `a` adds a new CONFIG_ entry.
- `d` deletes the selected entry.
- Changes written back to `prj.conf` on save.
- Reloads when project selection changes.

### Overlay Section

Read-only discovery based on selected board:

- Scans `<project>/boards/<board_name>.conf` for board-specific Kconfig entries.
- Checks for `<project>/boards/<board_name>.overlay` (devicetree overlay).
- Shows parsed entries from the `.conf` file if it exists.
- Shows existence status of the `.overlay` file.
- Updates automatically when board selection changes.
- Uses standard Zephyr convention (`boards/` directory in project root).

## Build Page Changes

Build page drops project, board, and shield input fields. Receives these values via broadcast messages.

```
Building: samples/ble-beacon
Board: nrf52840dk  Shield: (none)

Pristine  [ ]
CMake     [                              ]

ctrl+b: build  tab: next field
```

Build page retains:

- Pristine toggle
- CMake args text input
- Build execution and output viewport
- Build history recording

Project/board/shield displayed as read-only context labels at the top.

## Message Flow

### New Messages

- `BoardSelectedMsg{Board string}` - broadcast when board changes on Project page.
- `ShieldSelectedMsg{Shield string}` - broadcast when shield changes on Project page.

### Existing Messages (unchanged)

- `ProjectSelectedMsg{Path string}` - broadcast on project selection.
- `ProjectsLoadedMsg` - project discovery results.
- `BoardsLoadedMsg` - board discovery results.

### Broadcast Pattern

All selection messages use the existing non-key message forwarding in `Model.Update()`, which already broadcasts to all pages.

## Config Persistence

Existing fields used:

- `LastProject string` - persists selected project path.
- `DefaultBoard string` - persists selected board (now set from Project page).

New field:

- `LastShield string` - persists selected shield.

All saved to `.gust/config.json` via `config.Save()`.

## Key Bindings (Project Page)

| Key | Context | Action |
|-----|---------|--------|
| `p` | hardware section | open project picker overlay |
| `tab` | any | cycle to next field/section |
| `shift+tab` | any | cycle to previous field/section |
| `down`/`up` | board field | navigate dropdown |
| `enter` | board dropdown | select board |
| `/` | kconfig section | activate search |
| `e` | kconfig section | edit selected entry |
| `a` | kconfig section | add new entry |
| `d` | kconfig section | delete selected entry |
| `esc` | any | close dropdown/search/unfocus |
