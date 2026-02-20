package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPinRegistry_AddPin(t *testing.T) {
	reg := &PinRegistry{}

	// Add first pin
	reg.AddPin(Pin{User: "alice", Dir: "/tmp/test-a"})
	if len(reg.Pins) != 1 {
		t.Fatalf("expected 1 pin, got %d", len(reg.Pins))
	}
	if reg.Pins[0].User != "alice" {
		t.Errorf("expected user 'alice', got '%s'", reg.Pins[0].User)
	}

	// Add second pin
	reg.AddPin(Pin{User: "bob", Dir: "/tmp/test-b"})
	if len(reg.Pins) != 2 {
		t.Fatalf("expected 2 pins, got %d", len(reg.Pins))
	}

	// Update existing pin (same dir)
	reg.AddPin(Pin{User: "charlie", Dir: "/tmp/test-a"})
	if len(reg.Pins) != 2 {
		t.Fatalf("expected 2 pins after update, got %d", len(reg.Pins))
	}
	if reg.Pins[0].User != "charlie" {
		t.Errorf("expected user 'charlie' after update, got '%s'", reg.Pins[0].User)
	}
}

func TestPinRegistry_FindPin(t *testing.T) {
	reg := &PinRegistry{
		Pins: []Pin{
			{User: "alice", Dir: "/tmp/test-a"},
			{User: "bob", Dir: "/tmp/test-b", GitEmail: "bob@test.com"},
		},
	}

	// Find existing
	pin := reg.FindPin("/tmp/test-b")
	if pin == nil {
		t.Fatal("expected to find pin for /tmp/test-b")
	}
	if pin.User != "bob" {
		t.Errorf("expected user 'bob', got '%s'", pin.User)
	}
	if pin.GitEmail != "bob@test.com" {
		t.Errorf("expected email 'bob@test.com', got '%s'", pin.GitEmail)
	}

	// Find non-existing
	pin = reg.FindPin("/tmp/nonexistent")
	if pin != nil {
		t.Error("expected nil for non-existing directory")
	}
}

func TestPinRegistry_RemovePin(t *testing.T) {
	reg := &PinRegistry{
		Pins: []Pin{
			{User: "alice", Dir: "/tmp/test-a"},
			{User: "bob", Dir: "/tmp/test-b"},
			{User: "charlie", Dir: "/tmp/test-c"},
		},
	}

	// Remove middle pin
	removed := reg.RemovePin("/tmp/test-b")
	if !removed {
		t.Error("expected RemovePin to return true")
	}
	if len(reg.Pins) != 2 {
		t.Fatalf("expected 2 pins after removal, got %d", len(reg.Pins))
	}
	if reg.Pins[0].User != "alice" || reg.Pins[1].User != "charlie" {
		t.Error("wrong pins remaining after removal")
	}

	// Remove non-existing
	removed = reg.RemovePin("/tmp/nonexistent")
	if removed {
		t.Error("expected RemovePin to return false for non-existing")
	}
}

func TestSaveAndLoadPins(t *testing.T) {
	// Use a temp dir for config
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Save
	registry := &PinRegistry{
		Pins: []Pin{
			{User: "alice", Dir: "/tmp/test-a", GitEmail: "alice@test.com", GitName: "Alice"},
			{User: "bob", Dir: "/tmp/test-b", SSHKey: "/home/bob/.ssh/id_ed25519"},
		},
	}
	if err := SavePins(registry); err != nil {
		t.Fatalf("SavePins failed: %v", err)
	}

	// Verify file exists
	pinsFile := filepath.Join(tmpDir, "gh-autoprofile", "pins.yml")
	if _, err := os.Stat(pinsFile); err != nil {
		t.Fatalf("pins file not created: %v", err)
	}

	// Load
	loaded, err := LoadPins()
	if err != nil {
		t.Fatalf("LoadPins failed: %v", err)
	}
	if len(loaded.Pins) != 2 {
		t.Fatalf("expected 2 pins, got %d", len(loaded.Pins))
	}
	if loaded.Pins[0].User != "alice" || loaded.Pins[0].GitEmail != "alice@test.com" {
		t.Errorf("pin 0 mismatch: %+v", loaded.Pins[0])
	}
	if loaded.Pins[1].User != "bob" || loaded.Pins[1].SSHKey != "/home/bob/.ssh/id_ed25519" {
		t.Errorf("pin 1 mismatch: %+v", loaded.Pins[1])
	}
}

func TestLoadPins_EmptyWhenNoFile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	registry, err := LoadPins()
	if err != nil {
		t.Fatalf("LoadPins failed: %v", err)
	}
	if len(registry.Pins) != 0 {
		t.Errorf("expected empty registry, got %d pins", len(registry.Pins))
	}
}
