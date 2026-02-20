package direnv

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mdiloreto/gh-autoprofile/internal/config"
)

func TestWriteEnvrc_NewFile(t *testing.T) {
	tmpDir := t.TempDir()
	pin := config.Pin{User: "alice", Dir: tmpDir}

	if err := WriteEnvrc(pin); err != nil {
		t.Fatalf("WriteEnvrc failed: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(tmpDir, ".envrc"))
	if err != nil {
		t.Fatalf("cannot read .envrc: %v", err)
	}

	expected := "# gh-autoprofile:start\nuse_gh_autoprofile alice\n# gh-autoprofile:end\n"
	if string(content) != expected {
		t.Errorf("unexpected .envrc content:\ngot:  %q\nwant: %q", string(content), expected)
	}
}

func TestWriteEnvrc_WithGitIdentity(t *testing.T) {
	tmpDir := t.TempDir()
	pin := config.Pin{
		User:     "bob",
		Dir:      tmpDir,
		GitEmail: "bob@test.com",
		GitName:  "Bob Test",
	}

	if err := WriteEnvrc(pin); err != nil {
		t.Fatalf("WriteEnvrc failed: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(tmpDir, ".envrc"))
	if err != nil {
		t.Fatalf("cannot read .envrc: %v", err)
	}

	s := string(content)
	if !strings.Contains(s, "bob") {
		t.Error("expected .envrc to contain user 'bob'")
	}
	if !strings.Contains(s, "bob@test.com") {
		t.Error("expected .envrc to contain email")
	}
	if !strings.Contains(s, "Bob Test") {
		t.Error("expected .envrc to contain name")
	}
}

func TestWriteEnvrc_PreservesExistingContent(t *testing.T) {
	tmpDir := t.TempDir()
	envrcPath := filepath.Join(tmpDir, ".envrc")

	// Write pre-existing content
	existing := "export MY_VAR=hello\n"
	if err := os.WriteFile(envrcPath, []byte(existing), 0644); err != nil {
		t.Fatalf("cannot write existing .envrc: %v", err)
	}

	pin := config.Pin{User: "alice", Dir: tmpDir}
	if err := WriteEnvrc(pin); err != nil {
		t.Fatalf("WriteEnvrc failed: %v", err)
	}

	content, err := os.ReadFile(envrcPath)
	if err != nil {
		t.Fatalf("cannot read .envrc: %v", err)
	}

	s := string(content)
	if !strings.Contains(s, "export MY_VAR=hello") {
		t.Error("existing content was lost")
	}
	if !strings.Contains(s, markerStart) {
		t.Error("gh-autoprofile block not added")
	}
	if !strings.Contains(s, "use_gh_autoprofile alice") {
		t.Error("use_gh_autoprofile call not added")
	}
}

func TestWriteEnvrc_UpdatesExistingBlock(t *testing.T) {
	tmpDir := t.TempDir()
	envrcPath := filepath.Join(tmpDir, ".envrc")

	// First pin
	pin1 := config.Pin{User: "alice", Dir: tmpDir}
	if err := WriteEnvrc(pin1); err != nil {
		t.Fatalf("first WriteEnvrc failed: %v", err)
	}

	// Update to different user
	pin2 := config.Pin{User: "bob", Dir: tmpDir, GitEmail: "bob@test.com"}
	if err := WriteEnvrc(pin2); err != nil {
		t.Fatalf("second WriteEnvrc failed: %v", err)
	}

	content, err := os.ReadFile(envrcPath)
	if err != nil {
		t.Fatalf("cannot read .envrc: %v", err)
	}

	s := string(content)
	if strings.Contains(s, "alice") {
		t.Error("old user 'alice' still present after update")
	}
	if !strings.Contains(s, "bob") {
		t.Error("new user 'bob' not found")
	}
	// Should only have one start marker
	if strings.Count(s, markerStart) != 1 {
		t.Errorf("expected exactly 1 start marker, got %d", strings.Count(s, markerStart))
	}
}

func TestRemoveEnvrc_DeletesEmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	envrcPath := filepath.Join(tmpDir, ".envrc")

	// Write a block
	pin := config.Pin{User: "alice", Dir: tmpDir}
	if err := WriteEnvrc(pin); err != nil {
		t.Fatalf("WriteEnvrc failed: %v", err)
	}

	// Remove it
	if err := RemoveEnvrc(tmpDir); err != nil {
		t.Fatalf("RemoveEnvrc failed: %v", err)
	}

	// File should be gone (was only gh-autoprofile content)
	if _, err := os.Stat(envrcPath); !os.IsNotExist(err) {
		t.Error("expected .envrc to be deleted when empty after removal")
	}
}

func TestRemoveEnvrc_PreservesOtherContent(t *testing.T) {
	tmpDir := t.TempDir()
	envrcPath := filepath.Join(tmpDir, ".envrc")

	// Write existing content, then add block
	existing := "export MY_VAR=hello\n"
	if err := os.WriteFile(envrcPath, []byte(existing), 0644); err != nil {
		t.Fatalf("cannot write .envrc: %v", err)
	}
	pin := config.Pin{User: "alice", Dir: tmpDir}
	if err := WriteEnvrc(pin); err != nil {
		t.Fatalf("WriteEnvrc failed: %v", err)
	}

	// Remove block
	if err := RemoveEnvrc(tmpDir); err != nil {
		t.Fatalf("RemoveEnvrc failed: %v", err)
	}

	// File should still exist with original content
	content, err := os.ReadFile(envrcPath)
	if err != nil {
		t.Fatalf("cannot read .envrc: %v", err)
	}

	s := string(content)
	if !strings.Contains(s, "export MY_VAR=hello") {
		t.Error("existing content was lost after removal")
	}
	if strings.Contains(s, markerStart) {
		t.Error("gh-autoprofile markers still present after removal")
	}
}

func TestRemoveEnvrc_NoopWhenNoFile(t *testing.T) {
	tmpDir := t.TempDir()
	// Should not error when no .envrc exists
	if err := RemoveEnvrc(tmpDir); err != nil {
		t.Fatalf("RemoveEnvrc should not error on missing file: %v", err)
	}
}

func TestShellLibContent(t *testing.T) {
	// Verify the embedded shell library is non-empty and contains the function
	if len(shellLibContent) == 0 {
		t.Fatal("embedded shell library is empty")
	}
	s := string(shellLibContent)
	if !strings.Contains(s, "use_gh_autoprofile") {
		t.Error("shell library missing use_gh_autoprofile function")
	}
	if !strings.Contains(s, "GH_TOKEN") {
		t.Error("shell library missing GH_TOKEN export")
	}
	if !strings.Contains(s, "GITHUB_TOKEN") {
		t.Error("shell library missing GITHUB_TOKEN export")
	}
	if !strings.Contains(s, "GIT_AUTHOR_EMAIL") {
		t.Error("shell library missing GIT_AUTHOR_EMAIL")
	}
	if !strings.Contains(s, "GIT_SSH_COMMAND") {
		t.Error("shell library missing GIT_SSH_COMMAND")
	}
}
