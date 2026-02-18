# Combined Project/Build/Flash Page Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Merge the Project, Build, and Flash pages into a single "Project" page with vertical sections and a shared output viewport, while keeping build and flash logic in separate source files.

**Architecture:** `buildSection` and `flashSection` are unexported structs in `build.go`/`flash.go` that hold per-operation state and expose `start()`/`complete()`/`viewSection()` methods. `ProjectPage` in `project.go` embeds both, adds a shared `output strings.Builder` + `viewport`, and orchestrates routing of key events and `west.CommandResultMsg`. `app/page.go` drops `BuildPage`/`FlashPage` PageIDs (7 pages, renumbered). `cmd/gust/main.go` drops the two old page constructions.

**Tech Stack:** Go, Bubble Tea (Elm architecture), Lipgloss, Charmbracelet Bubbles (textinput, viewport), go.bug.st/serial, internal store/west/config packages.

---

### Task 1: Add buildSection struct to build.go

**Files:**
- Modify: `internal/pages/build.go`

Keep `BuildPage` untouched for now. Add `buildSection` below it.

**Step 1: Write failing tests for buildSection**

Add to `internal/pages/build_test.go` (keep existing tests — they still compile against `BuildPage`):

```go
func TestBuildSectionStartPassesBuildDir(t *testing.T) {
	var out strings.Builder
	wsRoot := t.TempDir()
	fake := &fakeRunner{nextMsg: west.CommandResultMsg{Output: "ok", ExitCode: 0, Duration: time.Second}}

	b := newBuildSection()
	requestID, cmd := b.start(wsRoot, ".", "nrf52840dk", "", "build-custom", fake, &out)
	if requestID == "" {
		t.Fatal("expected non-empty requestID")
	}
	if cmd == nil {
		t.Fatal("expected non-nil command")
	}
	_ = cmd()

	if len(fake.runCalls) != 1 {
		t.Fatalf("expected 1 run call, got %d", len(fake.runCalls))
	}
	args := fake.runCalls[0].args
	found := false
	for i, a := range args {
		if a == "-d" && i+1 < len(args) && args[i+1] == "build-custom" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected -d build-custom in args, got %v", args)
	}
}

func TestBuildSectionStartOmitsBuildDirWhenEmpty(t *testing.T) {
	var out strings.Builder
	fake := &fakeRunner{nextMsg: west.CommandResultMsg{Output: "ok", ExitCode: 0, Duration: time.Second}}

	b := newBuildSection()
	_, cmd := b.start(t.TempDir(), ".", "nrf52840dk", "", "", fake, &out)
	_ = cmd()

	for _, a := range fake.runCalls[0].args {
		if a == "-d" {
			t.Fatalf("expected no -d flag when buildDir empty, got %v", fake.runCalls[0].args)
		}
	}
}

func TestBuildSectionCompleteRecordsToStore(t *testing.T) {
	wsRoot := t.TempDir()
	st := store.New(t.TempDir())

	b := newBuildSection()
	b.buildStart = time.Now()
	var out strings.Builder
	b.complete(
		west.CommandResultMsg{Output: "ok", ExitCode: 0, Duration: time.Second},
		"nrf52840dk", ".", "", "",
		st, wsRoot, &out,
	)

	builds, err := st.Builds()
	if err != nil {
		t.Fatalf("Builds() error: %v", err)
	}
	if len(builds) != 1 {
		t.Fatalf("expected 1 build record, got %d", len(builds))
	}
	if !builds[0].Success {
		t.Fatal("expected success build record")
	}
}
```

You'll also need to add `"github.com/buckleypaul/gust/internal/store"` to build_test.go imports.

**Step 2: Run tests to verify they fail**

```bash
go test ./internal/pages/... -run TestBuildSection -v
```

Expected: FAIL — `newBuildSection undefined`, `buildSection undefined`.

**Step 3: Add buildSection to build.go**

Add this below the existing `BuildPage` code (do not touch `BuildPage` yet):

```go
// buildSection holds per-build state for the combined Project page.
// It is not a Page; ProjectPage orchestrates it.
type buildSection struct {
	cmakeInput textinput.Model
	pristine   bool
	state      buildState
	buildStart time.Time
	gitBranch  string
	gitCommit  string
	gitDirty   bool
	message    string
	seq        int
}

func newBuildSection() buildSection {
	cmake := textinput.New()
	cmake.Placeholder = "e.g. -DOVERLAY_CONFIG=overlay.conf"
	cmake.CharLimit = 512
	cmake.Prompt = ""
	return buildSection{cmakeInput: cmake}
}

func (b *buildSection) nextRequestID() string {
	b.seq++
	return fmt.Sprintf("build-%d", b.seq)
}

// viewSection renders the Build section header and controls.
func (b *buildSection) viewSection(width int, focusedCMake bool) string {
	var sb strings.Builder
	sectionLabel := lipgloss.NewStyle().Foreground(ui.Subtle).Bold(true)
	separator := strings.Repeat("─", max(width-9, 10))
	sb.WriteString("  " + sectionLabel.Render("── Build "+separator) + "\n")

	focusedLabel := lipgloss.NewStyle().Foreground(ui.Primary).Bold(true)
	normalLabel := lipgloss.NewStyle().Foreground(ui.Text)

	check := "[ ]"
	if b.pristine {
		check = "[x]"
	}
	sb.WriteString("  " + normalLabel.Render(fmt.Sprintf("%-9s", "Pristine")) + " " + check + "\n")

	inputWidth := width - labelWidth - 4
	if inputWidth < 10 {
		inputWidth = 10
	}
	b.cmakeInput.Width = inputWidth
	lbl := normalLabel.Render(fmt.Sprintf("%-9s", "CMake"))
	if focusedCMake {
		lbl = focusedLabel.Render(fmt.Sprintf("%-9s", "CMake"))
	}
	sb.WriteString("  " + lbl + " " + b.cmakeInput.View() + "\n")

	if b.message != "" {
		sb.WriteString("  " + b.message + "\n")
	}
	if b.state == buildStateRunning {
		sb.WriteString("  " + ui.DimStyle.Render("Building...") + "\n")
	}
	return sb.String()
}

// start launches west build, writes the command header to out, and returns
// the request ID and the tea.Cmd to execute.
func (b *buildSection) start(wsRoot, project, board, shield, buildDir string, runner west.Runner, out *strings.Builder) (requestID string, cmd tea.Cmd) {
	b.state = buildStateRunning
	b.buildStart = time.Now()
	b.message = ""
	requestID = b.nextRequestID()

	if !filepath.IsAbs(project) {
		project = filepath.Join(wsRoot, project)
	}

	b.gitBranch, b.gitCommit, b.gitDirty = "", "", false
	gitDir := project
	if gitDir == "" {
		gitDir = wsRoot
	}
	if o, err := gitCmd(gitDir, "branch", "--show-current"); err == nil {
		b.gitBranch = strings.TrimSpace(o)
	}
	if o, err := gitCmd(gitDir, "rev-parse", "--short=8", "HEAD"); err == nil {
		b.gitCommit = strings.TrimSpace(o)
	}
	if o, err := gitCmd(gitDir, "status", "--porcelain"); err == nil {
		b.gitDirty = strings.TrimSpace(o) != ""
	}

	args := []string{"build", "-b", board}
	if buildDir != "" {
		args = append(args, "-d", buildDir)
	}
	if b.pristine {
		args = append(args, "-p", "always")
	}
	if shield != "" {
		args = append(args, "--shield", shield)
	}
	if cmake := b.cmakeInput.Value(); cmake != "" {
		args = append(args, "--")
		args = append(args, strings.Fields(cmake)...)
	}
	args = append(args, project)

	out.WriteString("$ west " + strings.Join(args, " ") + "\n\n")
	return requestID, west.WithRequestID(requestID, runner.Run("west", args...))
}

// complete finalises build state and records to store.
func (b *buildSection) complete(result west.CommandResultMsg, board, app, shield, buildDir string, s *store.Store, wsRoot string, out *strings.Builder) {
	b.state = buildStateDone
	success := result.ExitCode == 0
	out.WriteString(result.Output)
	status := "success"
	if !success {
		status = fmt.Sprintf("failed (exit code: %d)", result.ExitCode)
	}
	out.WriteString(fmt.Sprintf("\nBuild %s in %s\n", status, result.Duration))

	var binarySize int64
	if success {
		dir := buildDir
		if dir == "" {
			dir = "build"
		}
		if fi, err := os.Stat(filepath.Join(wsRoot, dir, "zephyr", "zephyr.bin")); err == nil {
			binarySize = fi.Size()
		}
	}
	if s != nil {
		_ = s.AddBuild(store.BuildRecord{
			Board:      board,
			App:        app,
			Timestamp:  b.buildStart,
			Success:    success,
			Duration:   result.Duration.String(),
			Shield:     shield,
			Pristine:   b.pristine,
			CMakeArgs:  b.cmakeInput.Value(),
			GitBranch:  b.gitBranch,
			GitCommit:  b.gitCommit,
			GitDirty:   b.gitDirty,
			BuildDir:   buildDir,
			BinarySize: binarySize,
		})
	}
}
```

Add `"github.com/buckleypaul/gust/internal/store"` to build.go imports.

**Step 4: Run tests to verify they pass**

```bash
go test ./internal/pages/... -run TestBuildSection -v
```

Expected: PASS for all three new tests.

**Step 5: Commit**

```bash
git add internal/pages/build.go internal/pages/build_test.go
git commit -m "feat: add buildSection sub-component to build.go"
```

---

### Task 2: Add flashSection struct to flash.go

**Files:**
- Modify: `internal/pages/flash.go`
- Modify: `internal/pages/flash_test.go`

**Step 1: Write failing tests for flashSection**

Add to `flash_test.go` (keep existing tests):

```go
func TestFlashSectionStartPassesFlags(t *testing.T) {
	var out strings.Builder
	fake := &fakeRunner{nextMsg: west.CommandResultMsg{Output: "ok", ExitCode: 0, Duration: time.Second}}

	f := flashSection{}
	requestID, cmd := f.start("build-custom", "jlink", fake, &out)
	if requestID == "" {
		t.Fatal("expected non-empty requestID")
	}
	_ = cmd()

	args := fake.runCalls[0].args
	argStr := strings.Join(args, " ")
	if !strings.Contains(argStr, "-d build-custom") {
		t.Fatalf("expected -d build-custom in args, got %v", args)
	}
	if !strings.Contains(argStr, "--runner jlink") {
		t.Fatalf("expected --runner jlink in args, got %v", args)
	}
}

func TestFlashSectionStartOmitsEmptyFlags(t *testing.T) {
	var out strings.Builder
	fake := &fakeRunner{nextMsg: west.CommandResultMsg{Output: "ok", ExitCode: 0, Duration: time.Second}}

	f := flashSection{}
	_, cmd := f.start("", "", fake, &out)
	_ = cmd()

	args := fake.runCalls[0].args
	if len(args) != 1 || args[0] != "flash" {
		t.Fatalf("expected bare [flash] args, got %v", args)
	}
}
```

**Step 2: Run tests to verify they fail**

```bash
go test ./internal/pages/... -run TestFlashSection -v
```

Expected: FAIL — `flashSection undefined`.

**Step 3: Add flashSection to flash.go**

Add below the existing `FlashPage` code:

```go
// flashSection holds per-flash state for the combined Project page.
type flashSection struct {
	flashing   bool
	flashStart time.Time
	lastBuild  *store.BuildRecord
	message    string
	seq        int
}

func (f *flashSection) nextRequestID() string {
	f.seq++
	return fmt.Sprintf("flash-%d", f.seq)
}

func (f *flashSection) refreshLastBuild(s *store.Store) {
	if s == nil {
		f.lastBuild = nil
		return
	}
	builds, err := s.Builds()
	if err != nil || len(builds) == 0 {
		f.lastBuild = nil
		return
	}
	last := builds[len(builds)-1]
	f.lastBuild = &last
}

// viewSection renders the Flash section header and status.
func (f *flashSection) viewSection(width int) string {
	var sb strings.Builder
	sectionLabel := lipgloss.NewStyle().Foreground(ui.Subtle).Bold(true)
	separator := strings.Repeat("─", max(width-9, 10))
	sb.WriteString("  " + sectionLabel.Render("── Flash "+separator) + "\n")

	if f.lastBuild != nil {
		ts := f.lastBuild.Timestamp.Format("Jan 02 15:04")
		if f.lastBuild.Success {
			sb.WriteString("  Last build: " + ui.SuccessBadge("OK") + fmt.Sprintf("  (%s)\n", ts))
		} else {
			sb.WriteString("  Last build: " + ui.ErrorBadge("FAILED") + fmt.Sprintf("  (%s)\n", ts))
		}
	} else {
		sb.WriteString("  " + ui.DimStyle.Render("No recent builds. Run a build first.") + "\n")
	}
	if f.message != "" {
		sb.WriteString("  " + f.message + "\n")
	}
	if f.flashing {
		sb.WriteString("  " + ui.DimStyle.Render("Flashing...") + "\n")
	}
	return sb.String()
}

// start launches west flash, writes the command header to out.
func (f *flashSection) start(buildDir, flashRunner string, runner west.Runner, out *strings.Builder) (requestID string, cmd tea.Cmd) {
	f.flashing = true
	f.flashStart = time.Now()
	f.message = ""
	requestID = f.nextRequestID()

	args := []string{"flash"}
	if buildDir != "" {
		args = append(args, "-d", buildDir)
	}
	if flashRunner != "" {
		args = append(args, "--runner", flashRunner)
	}
	out.WriteString("$ west " + strings.Join(args, " ") + "\n\n")
	return requestID, west.WithRequestID(requestID, runner.Run("west", args...))
}

// complete finalises flash state and records to store.
func (f *flashSection) complete(result west.CommandResultMsg, board string, s *store.Store, out *strings.Builder) {
	f.flashing = false
	success := result.ExitCode == 0
	out.WriteString(result.Output)
	status := "success"
	if !success {
		status = fmt.Sprintf("failed (exit code: %d)", result.ExitCode)
	}
	out.WriteString(fmt.Sprintf("\nFlash %s in %s\n", status, result.Duration))
	if s != nil {
		_ = s.AddFlash(store.FlashRecord{
			Board:     board,
			Timestamp: f.flashStart,
			Success:   success,
			Duration:  result.Duration.String(),
		})
	}
}
```

flash.go already imports `store` and `strings` — confirm those are present; add `lipgloss` and `internal/ui` if not already there.

**Step 4: Run tests to verify they pass**

```bash
go test ./internal/pages/... -run TestFlashSection -v
```

Expected: PASS.

**Step 5: Run all existing tests to confirm no regressions**

```bash
go test ./internal/pages/...
```

Expected: all pass.

**Step 6: Commit**

```bash
git add internal/pages/flash.go internal/pages/flash_test.go
git commit -m "feat: add flashSection sub-component to flash.go"
```

---

### Task 3: Expand ProjectPage struct and update constructor

**Files:**
- Modify: `internal/pages/project.go`

**Step 1: Add new fields to ProjectPage struct**

In `project.go`, find the `ProjectPage` struct and add these fields after the existing `message` and `loading` fields:

```go
// Build/flash sub-components
build  buildSection
flash  flashSection
store  *store.Store
runner west.Runner

// Shared output panel
output          strings.Builder
viewport        viewport.Model
activeOp        string // "Build" or "Flash"
activeRequestID string
```

**Step 2: Add projFieldCMake to the field enum**

Find the `projField` const block and add `projFieldCMake` before `projFieldCount`:

```go
const (
	projFieldProject projField = iota
	projFieldBoard
	projFieldShield
	projFieldBuildDir
	projFieldRunner
	projFieldKconfig
	projFieldCMake  // cmake args input in build section
	projFieldCount
)
```

**Step 3: Update NewProjectPage signature**

Change:
```go
func NewProjectPage(cfg *config.Config, wsRoot string, manifestPath string) *ProjectPage {
```
To:
```go
func NewProjectPage(s *store.Store, cfg *config.Config, wsRoot string, manifestPath string, runners ...west.Runner) *ProjectPage {
```

Inside the constructor, before the `p := &ProjectPage{...}` line, add:
```go
runner := west.RealRunner()
if len(runners) > 0 && runners[0] != nil {
	runner = runners[0]
}
```

Add to the `&ProjectPage{...}` literal:
```go
store:    s,
runner:   runner,
build:    newBuildSection(),
viewport: viewport.New(0, 0),
```

Add `"github.com/buckleypaul/gust/internal/store"` and the reflow imports to project.go:
```go
"github.com/muesli/reflow/ansi"
"github.com/muesli/reflow/truncate"
"github.com/muesli/reflow/wrap"
"github.com/charmbracelet/bubbles/viewport"
```

**Step 4: Update blurCurrent and focusCurrent**

In `blurCurrent()`, add:
```go
case projFieldCMake:
	p.build.cmakeInput.Blur()
```

In `focusCurrent()`, add:
```go
case projFieldCMake:
	p.build.cmakeInput.Focus()
```

In `blurAll()`, add:
```go
p.build.cmakeInput.Blur()
```

**Step 5: Update InputCaptured**

Add `|| p.build.cmakeInput.Focused()` to the return expression.

**Step 6: Fix project_test.go to use new constructor signature**

All existing `NewProjectPage(&cfg, wsRoot, "")` calls become `NewProjectPage(nil, &cfg, wsRoot, "")`.

There are 7 occurrences across the existing tests — update all of them.

**Step 7: Confirm tests still pass**

```bash
go test ./internal/pages/...
```

Expected: all pass (no new behaviour yet, just signature changed).

**Step 8: Commit**

```bash
git add internal/pages/project.go internal/pages/project_test.go
git commit -m "feat: expand ProjectPage struct with build/flash sub-components"
```

---

### Task 4: Add build/flash handling to ProjectPage.Update()

**Files:**
- Modify: `internal/pages/project.go`

**Step 1: Write failing integration tests**

Add to `project_test.go`:

```go
func TestProjectPageCtrlBTriggersBuild(t *testing.T) {
	wsRoot := t.TempDir()
	cfg := config.Defaults()
	cfg.DefaultBoard = "nrf52840dk"
	cfg.BuildDir = "build-custom"
	fake := &fakeRunner{nextMsg: west.CommandResultMsg{Output: "ok", ExitCode: 0, Duration: time.Second}}

	p := NewProjectPage(nil, &cfg, wsRoot, "", fake)
	p.boardInput.SetValue("nrf52840dk")
	p.buildDirInput.SetValue("build-custom")

	page, cmd := p.Update(tea.KeyMsg{Type: tea.KeyCtrlB})
	_ = page
	if cmd == nil {
		t.Fatal("expected command from ctrl+b")
	}
	_ = cmd()

	if len(fake.runCalls) != 1 {
		t.Fatalf("expected 1 run call, got %d", len(fake.runCalls))
	}
	args := fake.runCalls[0].args
	found := false
	for i, a := range args {
		if a == "-d" && i+1 < len(args) && args[i+1] == "build-custom" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected -d build-custom in west build args, got %v", args)
	}
}

func TestProjectPageFTriggerFlash(t *testing.T) {
	cfg := config.Defaults()
	cfg.BuildDir = "build-custom"
	cfg.FlashRunner = "jlink"
	fake := &fakeRunner{nextMsg: west.CommandResultMsg{Output: "ok", ExitCode: 0, Duration: time.Second}}

	p := NewProjectPage(nil, &cfg, t.TempDir(), "", fake)
	p.buildDirInput.SetValue("build-custom")
	p.runnerInput.SetValue("jlink")

	page, cmd := p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	_ = page
	if cmd == nil {
		t.Fatal("expected command from f")
	}
	_ = cmd()

	args := fake.runCalls[0].args
	argStr := strings.Join(args, " ")
	if !strings.Contains(argStr, "-d build-custom") {
		t.Fatalf("expected -d build-custom, got %v", args)
	}
	if !strings.Contains(argStr, "--runner jlink") {
		t.Fatalf("expected --runner jlink, got %v", args)
	}
}

func TestProjectPageIgnoresForeignCommandResult(t *testing.T) {
	cfg := config.Defaults()
	p := NewProjectPage(nil, &cfg, t.TempDir(), "")
	p.activeRequestID = "build-1"
	p.activeOp = "Build"
	p.build.state = buildStateRunning

	page, _ := p.Update(west.CommandResultMsg{
		RequestID: "build-2",
		Output:    "foreign",
		ExitCode:  1,
	})
	updated := page.(*ProjectPage)

	if updated.build.state != buildStateRunning {
		t.Fatalf("expected build to remain running, got %v", updated.build.state)
	}
	if updated.output.Len() != 0 {
		t.Fatal("expected no output for foreign result")
	}
}
```

**Step 2: Run tests to verify they fail**

```bash
go test ./internal/pages/... -run "TestProjectPageCtrlB|TestProjectPageFTrigger|TestProjectPageIgnores" -v
```

Expected: FAIL — `ctrl+b` and `f` not yet wired.

**Step 3: Add helper methods to project.go**

Add these methods to `ProjectPage`:

```go
func (p *ProjectPage) projectValue() string {
	if p.projectPath == "" {
		return "."
	}
	return p.projectPath
}

func (p *ProjectPage) triggerBuild() tea.Cmd {
	board := p.boardInput.Value()
	if board == "" {
		p.message = "Board is required. Set a board above."
		return nil
	}
	p.output.Reset()
	p.activeOp = "Build"
	requestID, cmd := p.build.start(
		p.wsRoot, p.projectValue(), board,
		p.shieldInput.Value(), p.buildDirInput.Value(),
		p.runner, &p.output,
	)
	p.activeRequestID = requestID
	p.updateViewportContent()
	return cmd
}

func (p *ProjectPage) triggerFlash() tea.Cmd {
	p.flash.refreshLastBuild(p.store)
	p.output.Reset()
	p.activeOp = "Flash"
	requestID, cmd := p.flash.start(
		p.buildDirInput.Value(), p.runnerInput.Value(),
		p.runner, &p.output,
	)
	p.activeRequestID = requestID
	p.updateViewportContent()
	return cmd
}

func (p *ProjectPage) updateViewportContent() {
	if p.viewport.Width > 0 {
		content := p.output.String()
		wrapped := wrap.String(content, p.viewport.Width)
		lines := strings.Split(wrapped, "\n")
		for i, line := range lines {
			if ansi.PrintableRuneWidth(line) > p.viewport.Width {
				lines[i] = truncate.String(line, uint(p.viewport.Width))
			}
		}
		p.viewport.SetContent(strings.Join(lines, "\n"))
	} else {
		p.viewport.SetContent(p.output.String())
	}
}
```

**Step 4: Add west.CommandResultMsg handling to Update()**

In `ProjectPage.Update()`, add a new case in the `switch msg := msg.(type)` block:

```go
case west.CommandResultMsg:
	if p.activeRequestID == "" || msg.RequestID != p.activeRequestID {
		return p, nil
	}
	p.activeRequestID = ""
	board := p.boardInput.Value()
	switch p.activeOp {
	case "Build":
		p.build.complete(msg, board, p.projectValue(), p.shieldInput.Value(), p.buildDirInput.Value(), p.store, p.wsRoot, &p.output)
		// Persist board to config on success
		if msg.ExitCode == 0 {
			p.cfg.DefaultBoard = board
			_ = config.Save(*p.cfg, p.wsRoot, false)
		}
	case "Flash":
		p.flash.complete(msg, board, p.store, &p.output)
	}
	p.updateViewportContent()
	p.viewport.GotoBottom()
	return p, nil
```

**Step 5: Wire ctrl+b and f in handleKey()**

In `handleKey()`, find the global `switch keyStr {` block (where `"esc"` is handled) and add:

```go
case "ctrl+b":
	return p, p.triggerBuild()
case "f":
	if !p.InputCaptured() {
		return p, p.triggerFlash()
	}
```

Also update the `"esc"` case to clear output when present:

```go
case "esc":
	if p.output.Len() > 0 && !p.adding && !p.editing && !p.searchInput.Focused() {
		p.output.Reset()
		p.viewport.SetContent("")
		p.activeOp = ""
		p.activeRequestID = ""
		return p, nil
	}
	p.projectListOpen = false
	p.boardListOpen = false
	p.blurAll()
	return p, nil
```

Also handle `projFieldCMake` in the field-specific switch:

```go
case projFieldCMake:
	switch keyStr {
	case "enter":
		return p, p.triggerBuild()
	case "up":
		p.advanceField(-1)
		return p, nil
	case "down":
		p.advanceField(1)
		return p, nil
	}
	var cmd tea.Cmd
	p.build.cmakeInput, cmd = p.build.cmakeInput.Update(msg)
	return p, cmd
```

Also forward unhandled keys to the viewport when output is visible. At the very end of `handleKey()`, before the final `return p, nil`:

```go
if p.output.Len() > 0 {
	var cmd tea.Cmd
	p.viewport, cmd = p.viewport.Update(msg)
	return p, cmd
}
```

**Step 6: Run tests to verify they pass**

```bash
go test ./internal/pages/... -run "TestProjectPageCtrlB|TestProjectPageFTrigger|TestProjectPageIgnores" -v
```

Expected: PASS.

**Step 7: Run all tests**

```bash
go test ./internal/pages/...
```

Expected: all pass.

**Step 8: Commit**

```bash
git add internal/pages/project.go internal/pages/project_test.go
git commit -m "feat: wire build/flash triggers and CommandResultMsg into ProjectPage"
```

---

### Task 5: Update ProjectPage.View() for combined layout

**Files:**
- Modify: `internal/pages/project.go`

**Step 1: Rename current View() to viewConfig()**

Rename `func (p *ProjectPage) View() string` to `func (p *ProjectPage) viewConfig(width, height int) string`.

Update the signature to accept width/height instead of using `p.width`/`p.height` directly (the method already mostly uses `p.width`/`p.height` as local vars — just ensure the method uses the passed values for the outer sizing).

Inside `viewConfig`, before building the string, call `p.flash.refreshLastBuild(p.store)`.

Also append the build and flash sections to the output at the end (before the help bar line):

```go
// Build section
b.WriteString(p.build.viewSection(width, p.focusedField == projFieldCMake))
b.WriteString("\n")

// Flash section
b.WriteString(p.flash.viewSection(width))
b.WriteString("\n")
```

Update the help bar line to include the new key bindings:
```go
b.WriteString(ui.DimStyle.Render("  ↑/↓: navigate  /: search  e: edit  a: add  d: delete  ctrl+b: build  f: flash"))
```

**Step 2: Add the new View() that splits vertically**

```go
func (p *ProjectPage) View() string {
	if p.output.Len() > 0 {
		outputHeight := p.height / 2
		if outputHeight < 5 {
			outputHeight = 5
		}
		if outputHeight > 20 {
			outputHeight = 20
		}
		configHeight := p.height - outputHeight

		p.viewport.Width = p.width - 4
		p.viewport.Height = outputHeight - 2
		if p.viewport.Height < 3 {
			p.viewport.Height = 3
		}

		label := "Output"
		if p.activeOp != "" {
			label = p.activeOp + " Output"
		}
		outputPanel := ui.Panel(label, p.viewport.View(), p.width, outputHeight, false)
		return lipgloss.JoinVertical(lipgloss.Left,
			p.viewConfig(p.width, configHeight),
			outputPanel,
		)
	}
	return p.viewConfig(p.width, p.height)
}
```

**Step 3: Update ShortHelp()**

Add `ctrl+b` and `f` bindings to the default (non-editing) return:

```go
key.NewBinding(key.WithKeys("ctrl+b"), key.WithHelp("ctrl+b", "build")),
key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "flash")),
```

**Step 4: Build and run all tests**

```bash
go build ./... && go test ./internal/pages/...
```

Expected: compiles, all tests pass.

**Step 5: Commit**

```bash
git add internal/pages/project.go
git commit -m "feat: combined vertical layout with shared output panel in ProjectPage"
```

---

### Task 6: Cut over — update page.go, main.go, remove BuildPage/FlashPage

Do all steps in this task before committing, since they must be atomic.

**Files:**
- Modify: `internal/app/page.go`
- Modify: `cmd/gust/main.go`
- Modify: `internal/pages/build.go`
- Modify: `internal/pages/flash.go`

**Step 1: Update app/page.go**

Replace the PageID block:

```go
const (
	WorkspacePage PageID = iota
	ProjectPage
	MonitorPage
	TestPage
	ArtifactsPage
	WestPage
	SettingsPage
)

var PageOrder = []PageID{
	WorkspacePage,
	ProjectPage,
	MonitorPage,
	TestPage,
	ArtifactsPage,
	WestPage,
	SettingsPage,
}
```

(Remove `BuildPage` and `FlashPage` entirely.)

**Step 2: Update cmd/gust/main.go**

Change `NewProjectPage` call:
```go
app.ProjectPage: pages.NewProjectPage(st, &cfg, ws.Root, ws.ManifestPath, runner),
```

Remove these two lines:
```go
app.BuildPage:  pages.NewBuildPage(st, &cfg, ws.Root, runner),
app.FlashPage:  pages.NewFlashPage(st, &cfg, ws.Root, runner),
```

**Step 3: Remove BuildPage from build.go**

Delete everything from the top of build.go down through the end of `BuildPage` (the struct, `NewBuildPage`, `Init`, `Update`, `handleKey`, `advanceField`, `blurAll`, `blurCurrent`, `focusCurrent`, `View`, `viewForm`, `viewOutput`, `Name`, `ShortHelp`, `InputCaptured`, `SetSize`, `projectValue`, `updateViewportContent`, `startBuild`, `copyToClipboard`, `nextRequestID`).

Keep only: the `import` block (trimmed to what `buildSection` needs), the `formField`/`buildState` consts, the `labelWidth`/`minLeftWidth`/`maxLeftWidth`/`maxDropdownItems` consts, `buildSection`, `newBuildSection`, and all `buildSection` methods.

Trim imports to: `fmt`, `os`, `os/exec`, `path/filepath`, `strings`, `time`, `textinput`, `tea`, `lipgloss`, `store`, `ui`, `west`.

**Step 4: Remove FlashPage from flash.go**

Delete the `FlashPage` struct, `NewFlashPage`, `Init`, `Update`, `View`, `Name`, `ShortHelp`, `SetSize`, `refreshLastBuild`, `nextRequestID` methods that belong to `*FlashPage`.

Keep only: trimmed imports, `flashSection` struct, and all `flashSection` methods.

Trim imports to: `fmt`, `strings`, `time`, `tea`, `lipgloss`, `store`, `ui`, `west`.

**Step 5: Build to confirm it compiles**

```bash
go build ./...
```

Fix any import or reference errors before proceeding.

**Step 6: Run all tests**

```bash
go test ./...
```

Some old `BuildPage`/`FlashPage` tests will now fail (they reference removed types). That's expected — we'll fix them in the next task.

**Step 7: Commit**

```bash
git add internal/app/page.go cmd/gust/main.go internal/pages/build.go internal/pages/flash.go
git commit -m "feat: remove BuildPage/FlashPage page IDs and constructors, route through ProjectPage"
```

---

### Task 7: Update test files to remove old BuildPage/FlashPage tests

**Files:**
- Modify: `internal/pages/build_test.go`
- Modify: `internal/pages/flash_test.go`

**Step 1: Replace build_test.go**

Remove all test functions that reference `BuildPage` (the four old tests). Keep the three `TestBuildSection*` tests added in Task 1. Also remove any imports no longer needed (`path/filepath` — actually keep it, it's still used).

The file should end up containing only:
- `TestBuildSectionStartPassesBuildDir`
- `TestBuildSectionStartOmitsBuildDirWhenEmpty`
- `TestBuildSectionCompleteRecordsToStore`

**Step 2: Replace flash_test.go**

Remove the three old `TestFlashPage*` functions. Keep the two `TestFlashSection*` tests added in Task 2. Remove the `app` import if it's no longer referenced.

**Step 3: Run all tests**

```bash
go test ./...
```

Expected: all pass.

**Step 4: Commit**

```bash
git add internal/pages/build_test.go internal/pages/flash_test.go
git commit -m "test: remove old BuildPage/FlashPage tests, keep buildSection/flashSection tests"
```

---

### Task 8: Add project_test.go integration tests for broadcast message handling

The old `TestFlashPageHandlesBroadcastMessages` tested that `FlashPage` responded to `BoardSelectedMsg` etc. Now `ProjectPage` owns that. Add equivalent coverage.

**Files:**
- Modify: `internal/pages/project_test.go`

**Step 1: Add broadcast message tests**

```go
func TestProjectPageBroadcastUpdatesFlashState(t *testing.T) {
	cfg := config.Defaults()
	p := NewProjectPage(nil, &cfg, t.TempDir(), "")

	page, _ := p.Update(app.BuildDirChangedMsg{Dir: "build-x"})
	p = page.(*ProjectPage)
	if p.buildDirInput.Value() != "build-x" {
		t.Fatalf("expected buildDir build-x, got %s", p.buildDirInput.Value())
	}

	page, _ = p.Update(app.FlashRunnerChangedMsg{Runner: "openocd"})
	p = page.(*ProjectPage)
	if p.runnerInput.Value() != "openocd" {
		t.Fatalf("expected runner openocd, got %s", p.runnerInput.Value())
	}
}
```

**Step 2: Run all tests**

```bash
go test ./...
```

Expected: all pass.

**Step 3: Commit**

```bash
git add internal/pages/project_test.go
git commit -m "test: add ProjectPage integration tests for broadcast message handling"
```

---

### Task 9: Final verification

**Step 1: Full test run**

```bash
go test ./...
```

Expected: all pass, zero failures.

**Step 2: Build**

```bash
make build
```

Expected: clean build.

**Step 3: Verify sidebar page count**

In `internal/app/page.go`, confirm `PageOrder` has exactly 7 entries: Workspace, Project, Monitor, Test, Artifacts, West, Settings.

**Step 4: Commit if any last fixes were needed**

```bash
git add -p
git commit -m "chore: final cleanup after Project/Build/Flash merge"
```
