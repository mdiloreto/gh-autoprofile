package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// PinMode controls how tokens are injected into the shell.
type PinMode string

const (
	// ModeWrapper (default) — direnv exports only the GH_AUTOPROFILE_USER
	// marker; a shell hook creates gh()/git() wrapper functions that inject
	// the token per-invocation (~30 ms). The token never sits in the
	// shell environment.
	ModeWrapper PinMode = "wrapper"

	// ModeExport — direnv exports GH_TOKEN and GITHUB_TOKEN directly into
	// the shell environment (legacy behaviour). Use for directories where
	// third-party tools (Terraform, act, etc.) need the env var.
	ModeExport PinMode = "export"
)

// Pin represents a directory-to-account mapping.
type Pin struct {
	User     string  `yaml:"user"`
	Dir      string  `yaml:"dir"`
	Mode     PinMode `yaml:"mode,omitempty"`
	GitEmail string  `yaml:"git_email,omitempty"`
	GitName  string  `yaml:"git_name,omitempty"`
	SSHKey   string  `yaml:"ssh_key,omitempty"`
}

// EffectiveMode returns the pin's mode, defaulting to ModeWrapper.
func (p *Pin) EffectiveMode() PinMode {
	if p.Mode == "" {
		return ModeWrapper
	}
	return p.Mode
}

// PinRegistry holds all directory pins.
type PinRegistry struct {
	Pins []Pin `yaml:"pins"`
}

// ConfigDir returns the gh-autoprofile config directory path.
// Respects XDG_CONFIG_HOME, defaults to ~/.config/gh-autoprofile.
func ConfigDir() (string, error) {
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("cannot determine home directory: %w", err)
		}
		configHome = filepath.Join(home, ".config")
	}
	return filepath.Join(configHome, "gh-autoprofile"), nil
}

// PinsFilePath returns the full path to the pins.yml file.
func PinsFilePath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "pins.yml"), nil
}

// LoadPins reads the pin registry from disk.
// Returns an empty registry if the file doesn't exist.
func LoadPins() (*PinRegistry, error) {
	path, err := PinsFilePath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &PinRegistry{}, nil
		}
		return nil, fmt.Errorf("cannot read pins file: %w", err)
	}

	var registry PinRegistry
	if err := yaml.Unmarshal(data, &registry); err != nil {
		return nil, fmt.Errorf("cannot parse pins file %s: %w", path, err)
	}
	return &registry, nil
}

// SavePins writes the pin registry to disk, creating directories as needed.
func SavePins(registry *PinRegistry) error {
	path, err := PinsFilePath()
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("cannot create config directory %s: %w", dir, err)
	}

	data, err := yaml.Marshal(registry)
	if err != nil {
		return fmt.Errorf("cannot marshal pins: %w", err)
	}

	return os.WriteFile(path, data, 0600)
}

// FindPin returns the pin for a given directory, or nil if not found.
func (r *PinRegistry) FindPin(dir string) *Pin {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil
	}
	for i := range r.Pins {
		if r.Pins[i].Dir == absDir {
			return &r.Pins[i]
		}
	}
	return nil
}

// AddPin adds or updates a pin in the registry.
func (r *PinRegistry) AddPin(pin Pin) {
	absDir, err := filepath.Abs(pin.Dir)
	if err == nil {
		pin.Dir = absDir
	}

	// Update existing pin for same directory
	for i := range r.Pins {
		if r.Pins[i].Dir == pin.Dir {
			r.Pins[i] = pin
			return
		}
	}
	r.Pins = append(r.Pins, pin)
}

// RemovePin removes a pin by directory path. Returns true if found and removed.
func (r *PinRegistry) RemovePin(dir string) bool {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return false
	}
	for i := range r.Pins {
		if r.Pins[i].Dir == absDir {
			r.Pins = append(r.Pins[:i], r.Pins[i+1:]...)
			return true
		}
	}
	return false
}
