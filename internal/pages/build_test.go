package pages

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/buckleypaul/gust/internal/config"
	"github.com/buckleypaul/gust/internal/store"
	"github.com/buckleypaul/gust/internal/west"
)

func TestBuildPageIgnoresForeignCommandResults(t *testing.T) {
	cfg := config.Defaults()
	p := NewBuildPage(nil, &cfg, t.TempDir())
	p.state = buildStateRunning
	p.activeRequestID = "build-1"

	page, _ := p.Update(west.CommandResultMsg{
		RequestID: "build-2",
		Output:    "foreign output",
		ExitCode:  1,
	})
	updated := page.(*BuildPage)

	if updated.state != buildStateRunning {
		t.Fatalf("expected build to remain running, got state %v", updated.state)
	}
	if updated.output.Len() != 0 {
		t.Fatalf("expected no output appended for foreign result, got %q", updated.output.String())
	}
}

func TestBuildPageAcceptsMatchingCommandResult(t *testing.T) {
	cfg := config.Defaults()
	p := NewBuildPage(nil, &cfg, t.TempDir())
	p.state = buildStateRunning
	p.activeRequestID = "build-1"

	page, _ := p.Update(west.CommandResultMsg{
		RequestID: "build-1",
		Output:    "build failed output",
		ExitCode:  1,
	})
	updated := page.(*BuildPage)

	if updated.state != buildStateDone {
		t.Fatalf("expected build to finish, got state %v", updated.state)
	}
	if updated.activeRequestID != "" {
		t.Fatalf("expected active request to be cleared, got %q", updated.activeRequestID)
	}
	if !strings.Contains(updated.output.String(), "build failed output") {
		t.Fatalf("expected output to include command output, got %q", updated.output.String())
	}
}

func TestBuildPageStartBuildPassesBuildDir(t *testing.T) {
	wsRoot := t.TempDir()
	cfg := config.Defaults()
	cfg.DefaultBoard = "nrf52840dk_nrf52840"
	cfg.BuildDir = "build-custom"
	fake := &fakeRunner{
		nextMsg: west.CommandResultMsg{
			Output:   "ok",
			ExitCode: 0,
			Duration: time.Second,
		},
	}

	p := NewBuildPage(nil, &cfg, wsRoot, fake)
	p.selectedBoard = "nrf52840dk_nrf52840"

	cmd := p.startBuild()
	if cmd == nil {
		t.Fatal("expected command")
	}
	_ = cmd()

	if len(fake.runCalls) != 1 {
		t.Fatalf("expected 1 run call, got %d", len(fake.runCalls))
	}
	args := fake.runCalls[0].args
	// Look for -d build-custom in args
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

func TestBuildPageStartBuildOmitsBuildDirWhenEmpty(t *testing.T) {
	wsRoot := t.TempDir()
	cfg := config.Defaults()
	cfg.DefaultBoard = "nrf52840dk_nrf52840"
	cfg.BuildDir = ""
	fake := &fakeRunner{
		nextMsg: west.CommandResultMsg{
			Output:   "ok",
			ExitCode: 0,
			Duration: time.Second,
		},
	}

	p := NewBuildPage(nil, &cfg, wsRoot, fake)
	p.selectedBoard = "nrf52840dk_nrf52840"

	cmd := p.startBuild()
	if cmd == nil {
		t.Fatal("expected command")
	}
	_ = cmd()

	args := fake.runCalls[0].args
	for _, a := range args {
		if a == "-d" {
			t.Fatalf("expected no -d flag when buildDir is empty, got %v", args)
		}
	}
}

func TestBuildPageStartBuildUsesInjectedRunner(t *testing.T) {
	wsRoot := t.TempDir()
	cfg := config.Defaults()
	cfg.DefaultBoard = "nrf52840dk_nrf52840"
	fake := &fakeRunner{
		nextMsg: west.CommandResultMsg{
			Output:    "ok",
			ExitCode:  0,
			Duration:  time.Second,
			RequestID: "ignored",
		},
	}

	p := NewBuildPage(nil, &cfg, wsRoot, fake)
	p.selectedBoard = "nrf52840dk_nrf52840"
	p.selectedProject = filepath.Join("apps", "demo")

	cmd := p.startBuild()
	if cmd == nil {
		t.Fatal("expected command")
	}
	_ = cmd()

	if len(fake.runCalls) != 1 {
		t.Fatalf("expected 1 run call, got %d", len(fake.runCalls))
	}
	call := fake.runCalls[0]
	if call.name != "west" {
		t.Fatalf("expected west command, got %q", call.name)
	}
	if got := call.args[len(call.args)-1]; got != filepath.Join(wsRoot, "apps", "demo") {
		t.Fatalf("expected workspace-relative project path, got %q", got)
	}
	if p.activeRequestID == "" {
		t.Fatal("expected active request ID to be set")
	}
	if p.state != buildStateRunning {
		t.Fatalf("expected running state, got %v", p.state)
	}
}

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
		"nrf52840dk", ".", "", "build-custom",
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
	if builds[0].BuildDir != "build-custom" {
		t.Fatalf("expected BuildDir build-custom, got %q", builds[0].BuildDir)
	}
}
