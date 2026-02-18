# Design: Combined Project/Build/Flash Page

**Date:** 2026-02-18
**Status:** Approved

## Problem

The Project, Build, and Flash pages are three separate sidebar entries but represent a single linear workflow: configure → build → flash. Users must navigate between pages to complete a basic dev cycle. Configuration set on Project is silently consumed by Build, which is silently consumed by Flash — the coupling is real but the UI pretends otherwise.

## Goal

Merge Project, Build, and Flash into a single "Project" page (sidebar slot 2) that handles the full configure/build/flash workflow in one place, while keeping the build and flash logic in separate source files so they can grow independently.

## Design

### File structure

```
internal/pages/
├── project.go   — ProjectPage struct + orchestration (field routing, view composition, shared output viewport)
├── build.go     — buildSection struct: build state, startBuild(), viewBuildSection(), key handling methods
├── flash.go     — flashSection struct: flash state, startFlash(), viewFlashSection(), key handling methods
├── config.go    — existing kconfig helpers (unchanged)
```

`build.go` and `flash.go` stop being standalone `Page` implementations. They define unexported sub-component structs (`buildSection`, `flashSection`) that `ProjectPage` embeds.

### ProjectPage struct

```go
type ProjectPage struct {
    // --- existing config fields ---
    cfg          *config.Config
    wsRoot       string
    manifestPath string
    store        *store.Store   // NEW: needed for recording builds/flashes
    runner       west.Runner    // NEW: shared runner for build and flash

    projectInput  textinput.Model
    boardInput    textinput.Model
    shieldInput   textinput.Model
    buildDirInput textinput.Model
    runnerInput   textinput.Model
    // ... project/board/kconfig/overlay fields (unchanged) ...

    // --- embedded sub-components ---
    build buildSection
    flash flashSection

    // --- shared output (replaces per-page viewports) ---
    output          strings.Builder
    viewport        viewport.Model
    activeOp        string  // "Build" or "Flash" — labels the output panel
    activeRequestID string
    requestSeq      int

    width, height int
    message       string
    loading       bool
}
```

### buildSection (build.go)

Holds state previously in `BuildPage`:
- `cmakeInput textinput.Model`
- `pristine bool`
- `state buildState` (idle/running/done)
- `buildStart time.Time`
- `gitBranch, gitCommit string`, `gitDirty bool`

Exposes:
- `viewBuildSection(width int, focusedField projField) string`
- `handleKey(msg tea.KeyMsg, ...) tea.Cmd`
- `startBuild(wsRoot, project, board, shield, buildDir string, runner west.Runner) tea.Cmd`

### flashSection (flash.go)

Holds state previously in `FlashPage`:
- `flashing bool`
- `flashStart time.Time`
- `lastBuild *store.BuildRecord`

Exposes:
- `viewFlashSection(width int) string`
- `startFlash(buildDir, flashRunner string, runner west.Runner) tea.Cmd`
- `refreshLastBuild(s *store.Store)`

### Layout

```
┌──────────────────────────────────────┐
│  Project  app/my-app                 │  ↑
│  Board    nrf52840dk                 │  │ config section
│  Shield / Build Dir / Runner         │  │
│  ── Kconfig ──────────────────────── │  │
│  CONFIG_FOO = y  ...                 │  │
│  ── Board Overlay ─────────────────  │  │
│                                      │  │
│  ── Build ─────────────────────────  │  │ buildSection.viewBuildSection()
│  Pristine [ ]  CMake: ───────────    │  │
│                                      │  │
│  ── Flash ─────────────────────────  │  │ flashSection.viewFlashSection()
│  Last build: OK  runner: jlink       │  ↓
├──────────────────────────────────────┤
│ Output [Build / Flash]               │  ↑ shared viewport, labeled by activeOp
│ $ west build -b nrf52840dk ...       │  │ only rendered when output.Len() > 0
│ [273/273] Linking... succeeded       │  ↓
└──────────────────────────────────────┘
```

The output panel is hidden when empty (full height goes to config). When a build or flash runs, the output panel appears at a fixed height and the config sections scroll.

### Key handling

Keys are routed in `ProjectPage.Update()`:
1. Active text inputs get priority (existing logic, unchanged)
2. `ctrl+b` → `p.build.startBuild(...)` from any field position
3. `f` (no input captured) → `p.flash.startFlash(...)`
4. `west.CommandResultMsg` → checked against `activeRequestID`, then routed to build or flash post-processing based on `activeOp`
5. Viewport scroll keys forwarded to `p.viewport` when output is visible

### Page routing changes

`internal/app/page.go`:

```go
// Before
WorkspacePage, ProjectPage, BuildPage, FlashPage, MonitorPage, TestPage, ArtifactsPage, WestPage, SettingsPage

// After
WorkspacePage, ProjectPage, MonitorPage, TestPage, ArtifactsPage, WestPage, SettingsPage
// BuildPage and FlashPage PageIDs removed
```

`cmd/gust/main.go`: Remove construction of `BuildPage` and `FlashPage`. `NewProjectPage` gains `store` and `runner` parameters.

### Testing

Existing test files are updated rather than replaced:
- `build_test.go` — tests `buildSection` methods directly
- `flash_test.go` — tests `flashSection` methods directly
- `project_test.go` — integration tests for `ProjectPage`: config fields, kconfig, that build/flash results appear in the shared viewport

`runner_fake_test.go` is unchanged — the `west.Runner` interface is the same.

## Out of scope

- Flashing from the Artifacts pane (future work)
- Any changes to Monitor, Test, West, Settings, or Workspace pages
