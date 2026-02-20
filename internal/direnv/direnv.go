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

const (
	markerStart = "# gh-autoprofile:start"
	markerEnd   = "# gh-autoprofile:end"
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

// IsShellLibInstalled checks if the shell library file exists.
func IsShellLibInstalled() bool {
	path, err := ShellLibPath()
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}

// CheckShellHook looks for the direnv hook in common shell config files.
func CheckShellHook() bool {
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
func WriteEnvrc(pin config.Pin) error {
	envrcPath := filepath.Join(pin.Dir, ".envrc")

	// Build the use_gh_autoprofile call with all arguments
	var args []string
	args = append(args, quote(pin.User))
	if pin.GitEmail != "" {
		args = append(args, quote(pin.GitEmail))
		if pin.GitName != "" {
			args = append(args, quote(pin.GitName))
			if pin.SSHKey != "" {
				args = append(args, quote(pin.SSHKey))
			}
		}
	}

	// Build the managed block
	var block strings.Builder
	block.WriteString(markerStart + "\n")
	block.WriteString("use_gh_autoprofile " + strings.Join(args, " ") + "\n")
	block.WriteString(markerEnd + "\n")

	// Read existing .envrc (if any)
	existing, err := os.ReadFile(envrcPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("cannot read .envrc: %w", err)
	}

	var newContent string
	if len(existing) > 0 {
		content := string(existing)
		if strings.Contains(content, markerStart) {
			// Replace existing block
			startIdx := strings.Index(content, markerStart)
			endIdx := strings.Index(content, markerEnd)
			if endIdx != -1 {
				endIdx += len(markerEnd)
				if endIdx < len(content) && content[endIdx] == '\n' {
					endIdx++
				}
				newContent = content[:startIdx] + block.String() + content[endIdx:]
			} else {
				// Malformed markers — append fresh block
				newContent = content
				if !strings.HasSuffix(content, "\n") {
					newContent += "\n"
				}
				newContent += block.String()
			}
		} else {
			// Append block to existing content
			newContent = content
			if !strings.HasSuffix(content, "\n") {
				newContent += "\n"
			}
			newContent += block.String()
		}
	} else {
		newContent = block.String()
	}

	return os.WriteFile(envrcPath, []byte(newContent), 0644)
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

	return os.WriteFile(envrcPath, []byte(newContent+"\n"), 0644)
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

// quote wraps a string in shell-safe double quotes if it contains spaces.
func quote(s string) string {
	if strings.ContainsAny(s, " \t\"'\\$") {
		return fmt.Sprintf("%q", s)
	}
	return s
}
