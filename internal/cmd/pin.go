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
	var exportToken bool

	cmd := &cobra.Command{
		Use:   "pin <username>",
		Short: "Pin a GitHub account to a directory",
		Long: `Pin a GitHub account to a directory. When you cd into the directory,
the correct credentials and git identity are automatically activated.

By default, tokens are injected per-command via shell wrapper functions
(wrapper mode). The token never sits in your shell environment.

Use --export-token for directories where third-party tools (Terraform,
act, etc.) need GH_TOKEN / GITHUB_TOKEN as environment variables.

Examples:
  gh autoprofile pin alice
  gh autoprofile pin bob-work --dir ~/work --git-email bob@company.com
  gh autoprofile pin alice-freelance --dir ~/freelance --export-token`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPin(args[0], dir, gitEmail, gitName, sshKey, exportToken)
		},
	}

	cmd.Flags().StringVarP(&dir, "dir", "d", ".", "Directory to pin (defaults to current directory)")
	cmd.Flags().StringVar(&gitEmail, "git-email", "", "Git author/committer email for this directory")
	cmd.Flags().StringVar(&gitName, "git-name", "", "Git author/committer name for this directory")
	cmd.Flags().StringVar(&sshKey, "ssh-key", "", "Path to SSH private key for this directory")
	cmd.Flags().BoolVar(&exportToken, "export-token", false, "Export GH_TOKEN/GITHUB_TOKEN into the shell environment (less secure)")

	return cmd
}

func runPin(user, dir, gitEmail, gitName, sshKey string, exportToken bool) error {
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

	// Determine mode
	mode := config.ModeWrapper
	if exportToken {
		mode = config.ModeExport
	}

	// Create pin
	pin := config.Pin{
		User:     user,
		Dir:      absDir,
		Mode:     mode,
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
	modeLabel := "wrapper"
	if mode == config.ModeExport {
		modeLabel = "export"
	}
	fmt.Printf("\nPinned '%s' -> %s\n", user, absDir)
	fmt.Printf("  Mode:       %s\n", modeLabel)
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

	if mode == config.ModeWrapper {
		fmt.Println("\n  Token is injected per-command (never in shell environment).")
	} else {
		fmt.Println("\n  WARNING: Token is exported into shell environment.")
		fmt.Println("  All child processes can read GH_TOKEN/GITHUB_TOKEN.")
	}
	fmt.Println("\ncd into the directory to activate the profile.")

	return nil
}
