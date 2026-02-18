package pages

import (
	"strings"
	"testing"
	"time"

	"github.com/buckleypaul/gust/internal/west"
)

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
