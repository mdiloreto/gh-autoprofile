package direnv

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mdiloreto/gh-autoprofile/internal/config"
)

func TestWriteEnvrc_WrapperMode(t *testing.T) {
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

	fi, err := os.Stat(filepath.Join(tmpDir, ".envrc"))
	if err != nil {
		t.Fatalf("cannot stat .envrc: %v", err)
	}
	if got := fi.Mode().Perm(); got != 0600 {
		t.Errorf(".envrc permissions = %o, want 600", got)
	}
}

func TestWriteEnvrc_ExportMode(t *testing.T) {
	tmpDir := t.TempDir()
	pin := config.Pin{User: "alice", Dir: tmpDir, Mode: config.ModeExport}

	if err := WriteEnvrc(pin); err != nil {
		t.Fatalf("WriteEnvrc failed: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(tmpDir, ".envrc"))
	if err != nil {
		t.Fatalf("cannot read .envrc: %v", err)
	}

	expected := "# gh-autoprofile:start\nuse_gh_autoprofile_export alice\n# gh-autoprofile:end\n"
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
	// Default mode should be wrapper
	if !strings.Contains(s, "use_gh_autoprofile ") {
		t.Error("expected .envrc to use use_gh_autoprofile (wrapper mode)")
	}
	if strings.Contains(s, "use_gh_autoprofile_export") {
		t.Error("wrapper mode should not use use_gh_autoprofile_export")
	}
}

func TestWriteEnvrc_ExportWithGitIdentity(t *testing.T) {
	tmpDir := t.TempDir()
	pin := config.Pin{
		User:     "bob",
		Dir:      tmpDir,
		Mode:     config.ModeExport,
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
	if !strings.Contains(s, "use_gh_autoprofile_export") {
		t.Error("expected .envrc to use use_gh_autoprofile_export (export mode)")
	}
	if !strings.Contains(s, "bob@test.com") {
		t.Error("expected .envrc to contain email")
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

	// First pin (wrapper mode)
	pin1 := config.Pin{User: "alice", Dir: tmpDir}
	if err := WriteEnvrc(pin1); err != nil {
		t.Fatalf("first WriteEnvrc failed: %v", err)
	}

	// Update to export mode with different user
	pin2 := config.Pin{User: "bob", Dir: tmpDir, Mode: config.ModeExport, GitEmail: "bob@test.com"}
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
	if !strings.Contains(s, "use_gh_autoprofile_export bob") {
		t.Error("new export mode with user 'bob' not found")
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
	// Verify the embedded shell library contains both wrapper and export functions
	if len(shellLibContent) == 0 {
		t.Fatal("embedded shell library is empty")
	}
	s := string(shellLibContent)
	if !strings.Contains(s, "use_gh_autoprofile()") {
		t.Error("shell library missing use_gh_autoprofile function (wrapper mode)")
	}
	if !strings.Contains(s, "use_gh_autoprofile_export()") {
		t.Error("shell library missing use_gh_autoprofile_export function (export mode)")
	}
	if !strings.Contains(s, "GH_AUTOPROFILE_USER") {
		t.Error("shell library missing GH_AUTOPROFILE_USER marker export")
	}
	if !strings.Contains(s, "GH_TOKEN") {
		t.Error("shell library missing GH_TOKEN export (in export mode)")
	}
	if !strings.Contains(s, "GIT_AUTHOR_EMAIL") {
		t.Error("shell library missing GIT_AUTHOR_EMAIL")
	}
	if !strings.Contains(s, "GIT_SSH_COMMAND") {
		t.Error("shell library missing GIT_SSH_COMMAND")
	}
}

func TestShellHookContent(t *testing.T) {
	// Verify the embedded shell hook is non-empty and contains key elements
	if len(shellHookContent) == 0 {
		t.Fatal("embedded shell hook is empty")
	}
	s := string(shellHookContent)
	if !strings.Contains(s, "_gh_autoprofile_hook") {
		t.Error("shell hook missing _gh_autoprofile_hook function")
	}
	if !strings.Contains(s, "GH_AUTOPROFILE_USER") {
		t.Error("shell hook missing GH_AUTOPROFILE_USER reference")
	}
	if !strings.Contains(s, "gh auth token") {
		t.Error("shell hook missing 'gh auth token' invocation")
	}
	if !strings.Contains(s, "precmd") {
		t.Error("shell hook missing zsh precmd integration")
	}
	if !strings.Contains(s, "PROMPT_COMMAND") {
		t.Error("shell hook missing bash PROMPT_COMMAND integration")
	}
	// Verify export mode detection: hook should check for GH_TOKEN to skip wrappers
	if !strings.Contains(s, `"${GH_TOKEN:-}"`) && !strings.Contains(s, `"${GH_TOKEN:+1}"`) {
		t.Error("shell hook missing GH_TOKEN check for export mode detection")
	}
	// Verify state tracking includes token presence
	if !strings.Contains(s, "_gh_autoprofile_last_has_token") {
		t.Error("shell hook missing _gh_autoprofile_last_has_token state tracking")
	}
}

func TestShellQuote(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"alice", "alice"},
		{"bob-work", "bob-work"},
		{"alice@test.com", "alice@test.com"},
		{"/home/user/.ssh/id_ed25519", "/home/user/.ssh/id_ed25519"},
		{"Bob Smith", "'Bob Smith'"},
		{"it's a test", "'it'\\''s a test'"},
		{"has spaces", "'has spaces'"},
		{"has\ttab", "'has\ttab'"},
		{"has$dollar", "'has$dollar'"},
		{"", "''"},
	}

	for _, tt := range tests {
		got := shellQuote(tt.input)
		if got != tt.expected {
			t.Errorf("shellQuote(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestInjectHookSource(t *testing.T) {
	tmpDir := t.TempDir()
	rcPath := filepath.Join(tmpDir, ".zshrc")
	hookPath := filepath.Join(tmpDir, "hook.sh")

	// First injection — creates the file
	if err := InjectHookSource(rcPath, hookPath); err != nil {
		t.Fatalf("InjectHookSource failed: %v", err)
	}

	content, err := os.ReadFile(rcPath)
	if err != nil {
		t.Fatalf("cannot read RC file: %v", err)
	}
	s := string(content)
	if !strings.Contains(s, hookMarkerStart) {
		t.Error("hook marker start not found")
	}
	if !strings.Contains(s, hookPath) {
		t.Errorf("hook path %q not found in RC file", hookPath)
	}

	// Second injection — should replace, not duplicate
	if err := InjectHookSource(rcPath, hookPath); err != nil {
		t.Fatalf("second InjectHookSource failed: %v", err)
	}

	content, err = os.ReadFile(rcPath)
	if err != nil {
		t.Fatalf("cannot read RC file: %v", err)
	}
	s = string(content)
	if strings.Count(s, hookMarkerStart) != 1 {
		t.Errorf("expected exactly 1 hook marker, got %d", strings.Count(s, hookMarkerStart))
	}
}

func TestInjectHookSource_PreservesExisting(t *testing.T) {
	tmpDir := t.TempDir()
	rcPath := filepath.Join(tmpDir, ".zshrc")
	hookPath := filepath.Join(tmpDir, "hook.sh")

	// Write existing RC content
	existing := "# My zsh config\nexport EDITOR=vim\n"
	if err := os.WriteFile(rcPath, []byte(existing), 0644); err != nil {
		t.Fatalf("cannot write RC file: %v", err)
	}

	if err := InjectHookSource(rcPath, hookPath); err != nil {
		t.Fatalf("InjectHookSource failed: %v", err)
	}

	content, err := os.ReadFile(rcPath)
	if err != nil {
		t.Fatalf("cannot read RC file: %v", err)
	}
	s := string(content)
	if !strings.Contains(s, "export EDITOR=vim") {
		t.Error("existing content was lost")
	}
	if !strings.Contains(s, hookMarkerStart) {
		t.Error("hook marker not added")
	}
}
