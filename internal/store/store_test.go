package store

import (
	"testing"
	"time"
)

func TestAddAndRetrieveBuilds(t *testing.T) {
	tmp := t.TempDir()
	s := New(tmp)

	record := BuildRecord{
		Board:     "nrf52840dk_nrf52840",
		App:       ".",
		Timestamp: time.Now(),
		Success:   true,
		Duration:  "12.5s",
		Artifacts: []string{"build/zephyr/zephyr.hex"},
	}

	if err := s.AddBuild(record); err != nil {
		t.Fatalf("AddBuild failed: %v", err)
	}

	builds, err := s.Builds()
	if err != nil {
		t.Fatalf("Builds failed: %v", err)
	}
	if len(builds) != 1 {
		t.Fatalf("expected 1 build, got %d", len(builds))
	}
	if builds[0].Board != "nrf52840dk_nrf52840" {
		t.Errorf("expected board=nrf52840dk_nrf52840, got=%s", builds[0].Board)
	}
}

func TestAddMultipleRecords(t *testing.T) {
	tmp := t.TempDir()
	s := New(tmp)

	s.AddBuild(BuildRecord{Board: "board1", Timestamp: time.Now(), Success: true, Duration: "5s"})
	s.AddBuild(BuildRecord{Board: "board2", Timestamp: time.Now(), Success: false, Duration: "3s"})
	s.AddFlash(FlashRecord{Board: "board1", Timestamp: time.Now(), Success: true, Duration: "2s"})

	builds, _ := s.Builds()
	if len(builds) != 2 {
		t.Errorf("expected 2 builds, got %d", len(builds))
	}

	flashes, _ := s.Flashes()
	if len(flashes) != 1 {
		t.Errorf("expected 1 flash, got %d", len(flashes))
	}
}

func TestEmptyStore(t *testing.T) {
	tmp := t.TempDir()
	s := New(tmp)

	builds, err := s.Builds()
	if err != nil {
		t.Fatalf("Builds on empty store failed: %v", err)
	}
	if len(builds) != 0 {
		t.Errorf("expected 0 builds, got %d", len(builds))
	}
}
