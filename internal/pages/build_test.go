package pages

import (
	"strings"
	"testing"
	"time"

	"github.com/buckleypaul/gust/internal/store"
	"github.com/buckleypaul/gust/internal/west"
)

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
