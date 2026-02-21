package direnv

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mdiloreto/gh-autoprofile/internal/config"
)

//go:embed shell/gh-autoprofile.sh
var shellLibContent []byte

//go:embed shell/gh-autoprofile-hook.sh
var shellHookContent []byte

const (
	markerStart = "# gh-autoprofile:start"
	markerEnd   = "# gh-autoprofile:end"

	hookMarkerStart = "# gh-autoprofile-hook:start"
	hookMarkerEnd   = "# gh-autoprofile-hook:end"
)

// IsInstalled checks if direnv is available in PATH.
func IsInstalled() bool {
	_, err := exec.LookPath("direnv")
	return err == nil
}

// GetVersion returns the direnv version string.
func GetVersion() (string, error) {
	cmd := exec.Command("direnv", "version")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// ShellLibDir returns the path to direnv's lib directory where custom
// shell functions are auto-loaded from.
func ShellLibDir() (string, error) {
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		configHome = filepath.Join(home, ".config")
	}
	return filepath.Join(configHome, "direnv", "lib"), nil
}

// ShellLibPath returns the full path to the installed shell library.
func ShellLibPath() (string, error) {
	dir, err := ShellLibDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "gh-autoprofile.sh"), nil
}

// ShellHookPath returns the path to the installed shell hook script.
func ShellHookPath() (string, error) {
	dir, err := config.ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "hook.sh"), nil
}

// InstallShellLib writes the embedded shell library to direnv's lib directory.
func InstallShellLib() error {
	libDir, err := ShellLibDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(libDir, 0755); err != nil {
		return fmt.Errorf("cannot create direnv lib directory %s: %w", libDir, err)
	}

	dest := filepath.Join(libDir, "gh-autoprofile.sh")
	return os.WriteFile(dest, shellLibContent, 0644)
}

// InstallShellHook writes the shell hook script to the config directory
// and injects a source line into the user's shell RC file (~/.zshrc or
// ~/.bashrc). The hook creates gh()/git() wrapper functions when
// GH_AUTOPROFILE_USER is set by direnv.
func InstallShellHook() (hookPath string, err error) {
	// Write hook script to config dir.
	hookPath, err = ShellHookPath()
	if err != nil {
		return "", err
	}
	dir := filepath.Dir(hookPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("cannot create config directory %s: %w", dir, err)
	}
	if err := os.WriteFile(hookPath, shellHookContent, 0644); err != nil {
		return "", fmt.Errorf("cannot write hook script: %w", err)
	}
	return hookPath, nil
}

// InjectHookSource adds a `source <hookPath>` line into the given shell RC
// file, wrapped in markers so it can be updated/removed later.
func InjectHookSource(rcPath, hookPath string) error {
	block := hookMarkerStart + "\n" +
		`source "` + hookPath + `"` + "\n" +
		hookMarkerEnd + "\n"

	existing, err := os.ReadFile(rcPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("cannot read %s: %w", rcPath, err)
	}

	content := string(existing)

	// Already present — replace the existing block.
	if strings.Contains(content, hookMarkerStart) {
		startIdx := strings.Index(content, hookMarkerStart)
		endIdx := strings.Index(content, hookMarkerEnd)
		if endIdx != -1 {
			endIdx += len(hookMarkerEnd)
			if endIdx < len(content) && content[endIdx] == '\n' {
				endIdx++
			}
			content = content[:startIdx] + block + content[endIdx:]
			return os.WriteFile(rcPath, []byte(content), 0644)
		}
	}

	// Append.
	if len(content) > 0 && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	content += "\n" + block
	return os.WriteFile(rcPath, []byte(content), 0644)
}

// CheckShellHookInstalled checks if the gh-autoprofile hook source line
// is present in common shell config files.
func CheckShellHookInstalled() bool {
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	files := []string{
		filepath.Join(home, ".zshrc"),
		filepath.Join(home, ".bashrc"),
		filepath.Join(home, ".bash_profile"),
		filepath.Join(home, ".profile"),
	}
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		if strings.Contains(string(data), hookMarkerStart) {
			return true
		}
	}
	return false
}

// IsShellLibInstalled checks if the shell library file exists.
func IsShellLibInstalled() bool {
	path, err := ShellLibPath()
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}

// CheckDirenvHook looks for the direnv hook in common shell config files.
func CheckDirenvHook() bool {
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	files := []string{
		filepath.Join(home, ".bashrc"),
		filepath.Join(home, ".zshrc"),
		filepath.Join(home, ".config", "fish", "config.fish"),
		filepath.Join(home, ".bash_profile"),
		filepath.Join(home, ".profile"),
	}
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		if strings.Contains(string(data), "direnv hook") || strings.Contains(string(data), "direnv.fish") {
			return true
		}
	}
	return false
}

// WriteEnvrc creates or updates the .envrc file in the pin's directory.
// Uses markers to manage only the gh-autoprofile block, preserving any
// existing user content in the .envrc.
//
// In wrapper mode (default) it writes: use_gh_autoprofile <user> ...
// In export mode it writes: use_gh_autoprofile_export <user> ...
func WriteEnvrc(pin config.Pin) error {
	envrcPath := filepath.Join(pin.Dir, ".envrc")

	// Choose the direnv function based on mode.
	fnName := "use_gh_autoprofile"
	if pin.EffectiveMode() == config.ModeExport {
		fnName = "use_gh_autoprofile_export"
	}

	// Build arguments.
	var args []string
	args = append(args, shellQuote(pin.User))
	if pin.GitEmail != "" {
		args = append(args, shellQuote(pin.GitEmail))
		if pin.GitName != "" {
			args = append(args, shellQuote(pin.GitName))
			if pin.SSHKey != "" {
				args = append(args, shellQuote(pin.SSHKey))
			}
		}
	}

	// Build the managed block.
	var block strings.Builder
	block.WriteString(markerStart + "\n")
	block.WriteString(fnName + " " + strings.Join(args, " ") + "\n")
	block.WriteString(markerEnd + "\n")

	// Read existing .envrc (if any).
	existing, err := os.ReadFile(envrcPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("cannot read .envrc: %w", err)
	}

	var newContent string
	if len(existing) > 0 {
		content := string(existing)
		if strings.Contains(content, markerStart) {
			// Replace existing block.
			startIdx := strings.Index(content, markerStart)
			endIdx := strings.Index(content, markerEnd)
			if endIdx != -1 {
				endIdx += len(markerEnd)
				if endIdx < len(content) && content[endIdx] == '\n' {
					endIdx++
				}
				newContent = content[:startIdx] + block.String() + content[endIdx:]
			} else {
				// Malformed markers — append fresh block.
				newContent = content
				if !strings.HasSuffix(content, "\n") {
					newContent += "\n"
				}
				newContent += block.String()
			}
		} else {
			// Append block to existing content.
			newContent = content
			if !strings.HasSuffix(content, "\n") {
				newContent += "\n"
			}
			newContent += block.String()
		}
	} else {
		newContent = block.String()
	}

	if err := os.WriteFile(envrcPath, []byte(newContent), 0600); err != nil {
		return err
	}
	if err := os.Chmod(envrcPath, 0600); err != nil {
		return fmt.Errorf("cannot set .envrc permissions: %w", err)
	}
	return nil
}

// RemoveEnvrc removes the gh-autoprofile block from .envrc.
// If the file is empty after removal, it deletes the file entirely.
func RemoveEnvrc(dir string) error {
	envrcPath := filepath.Join(dir, ".envrc")

	existing, err := os.ReadFile(envrcPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("cannot read .envrc: %w", err)
	}

	content := string(existing)
	if !strings.Contains(content, markerStart) {
		return nil // No gh-autoprofile block
	}

	startIdx := strings.Index(content, markerStart)
	endIdx := strings.Index(content, markerEnd)
	if endIdx == -1 {
		return nil
	}
	endIdx += len(markerEnd)
	if endIdx < len(content) && content[endIdx] == '\n' {
		endIdx++
	}

	newContent := strings.TrimSpace(content[:startIdx] + content[endIdx:])
	if newContent == "" {
		return os.Remove(envrcPath)
	}

	if err := os.WriteFile(envrcPath, []byte(newContent+"\n"), 0600); err != nil {
		return err
	}
	if err := os.Chmod(envrcPath, 0600); err != nil {
		return fmt.Errorf("cannot set .envrc permissions: %w", err)
	}
	return nil
}

// AllowEnvrc runs `direnv allow` on the .envrc file.
func AllowEnvrc(dir string) error {
	envrcPath := filepath.Join(dir, ".envrc")
	cmd := exec.Command("direnv", "allow", envrcPath)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("direnv allow failed: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}

// shellQuote wraps a string in single quotes for safe shell interpolation.
// Single quotes inside the string are escaped as '\”.
func shellQuote(s string) string {
	// If the string is simple (alphanumeric, dash, dot, underscore, slash,
	// at, plus, colon) it doesn't need quoting.
	safe := true
	for _, c := range s {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') ||
			c == '-' || c == '.' || c == '_' || c == '/' || c == '@' || c == '+' || c == ':') {
			safe = false
			break
		}
	}
	if safe && len(s) > 0 {
		return s
	}

	// Escape using single quotes: replace ' with '\''
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
