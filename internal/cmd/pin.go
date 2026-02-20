package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mdiloreto/gh-autoprofile/internal/config"
	direnvlib "github.com/mdiloreto/gh-autoprofile/internal/direnv"
	"github.com/mdiloreto/gh-autoprofile/internal/ghauth"
	"github.com/spf13/cobra"
)

// NewPinCmd creates the `pin` subcommand.
func NewPinCmd() *cobra.Command {
	var dir, gitEmail, gitName, sshKey string

	cmd := &cobra.Command{
		Use:   "pin <username>",
		Short: "Pin a GitHub account to a directory",
		Long: `Pin a GitHub account to a directory. When you cd into the directory,
direnv will automatically export the correct GH_TOKEN and git identity.

Examples:
  gh autoprofile pin alice
  gh autoprofile pin bob-work --dir ~/work --git-email bob@company.com
  gh autoprofile pin alice-freelance --dir ~/freelance --git-name "Alice Freelance" --git-email alice@freelance.com`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPin(args[0], dir, gitEmail, gitName, sshKey)
		},
	}

	cmd.Flags().StringVarP(&dir, "dir", "d", ".", "Directory to pin (defaults to current directory)")
	cmd.Flags().StringVar(&gitEmail, "git-email", "", "Git author/committer email for this directory")
	cmd.Flags().StringVar(&gitName, "git-name", "", "Git author/committer name for this directory")
	cmd.Flags().StringVar(&sshKey, "ssh-key", "", "Path to SSH private key for this directory")

	return cmd
}

func runPin(user, dir, gitEmail, gitName, sshKey string) error {
	// Resolve absolute path
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("cannot resolve directory: %w", err)
	}

	// Validate directory exists
	info, err := os.Stat(absDir)
	if err != nil {
		return fmt.Errorf("directory does not exist: %s", absDir)
	}
	if !info.IsDir() {
		return fmt.Errorf("not a directory: %s", absDir)
	}

	// Validate user is logged in to gh
	fmt.Printf("Validating account '%s'... ", user)
	if err := ghauth.ValidateUser(user); err != nil {
		fmt.Println("FAILED")
		return err
	}
	fmt.Println("OK")

	// Validate SSH key exists if specified
	if sshKey != "" {
		absKey, err := filepath.Abs(sshKey)
		if err == nil {
			sshKey = absKey
		}
		if _, err := os.Stat(sshKey); err != nil {
			return fmt.Errorf("SSH key not found: %s", sshKey)
		}
	}

	// Check direnv shell library is installed
	if !direnvlib.IsShellLibInstalled() {
		return fmt.Errorf("direnv shell library not installed. Run first: gh autoprofile setup")
	}

	// Create pin
	pin := config.Pin{
		User:     user,
		Dir:      absDir,
		GitEmail: gitEmail,
		GitName:  gitName,
		SSHKey:   sshKey,
	}

	// Save to registry
	registry, err := config.LoadPins()
	if err != nil {
		return fmt.Errorf("cannot load pin registry: %w", err)
	}
	registry.AddPin(pin)
	if err := config.SavePins(registry); err != nil {
		return fmt.Errorf("cannot save pin registry: %w", err)
	}

	// Write .envrc
	if err := direnvlib.WriteEnvrc(pin); err != nil {
		return fmt.Errorf("cannot write .envrc: %w", err)
	}

	// Auto-allow .envrc
	if direnvlib.IsInstalled() {
		if err := direnvlib.AllowEnvrc(absDir); err != nil {
			fmt.Printf("Warning: could not auto-allow .envrc: %v\n", err)
			fmt.Printf("  Run manually: direnv allow %s/.envrc\n", absDir)
		}
	}

	// Summary
	fmt.Printf("\nPinned '%s' -> %s\n", user, absDir)
	if gitEmail != "" {
		fmt.Printf("  Git email:  %s\n", gitEmail)
	}
	if gitName != "" {
		fmt.Printf("  Git name:   %s\n", gitName)
	}
	if sshKey != "" {
		fmt.Printf("  SSH key:    %s\n", sshKey)
	}
	fmt.Printf("  .envrc:     %s/.envrc\n", absDir)
	fmt.Println("\ncd into the directory to activate the profile.")

	return nil
}
