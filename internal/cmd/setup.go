package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mdiloreto/gh-autoprofile/internal/config"
	direnvlib "github.com/mdiloreto/gh-autoprofile/internal/direnv"
	"github.com/mdiloreto/gh-autoprofile/internal/ghauth"
	"github.com/spf13/cobra"
)

// NewSetupCmd creates the `setup` subcommand.
func NewSetupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Install direnv shell library, shell hook, and validate prerequisites",
		Long: `Checks that gh CLI (>= 2.40.0) and direnv are installed, then installs:

  1. The direnv shell library (use_gh_autoprofile / use_gh_autoprofile_export)
  2. The shell hook that creates per-command wrapper functions for secure
     token injection (wrapper mode)

Run this once after installing gh-autoprofile.

Use --migrate after upgrading to refresh generated files,
repair permissions, and update existing pins to the latest defaults.`,
		RunE: runSetup,
	}
	cmd.Flags().Bool("migrate", false, "Migrate existing pins and rewrite managed .envrc files")
	return cmd
}

func runSetup(cmd *cobra.Command, args []string) error {
	migrate, err := cmd.Flags().GetBool("migrate")
	if err != nil {
		return err
	}

	fmt.Println("gh-autoprofile setup")
	fmt.Println("====================")
	fmt.Println()

	allGood := true

	// 1. Check gh CLI
	fmt.Print("  Checking gh CLI............. ")
	ghVersion, err := ghauth.GetGHVersion()
	if err != nil {
		fmt.Println("MISSING")
		fmt.Println("    gh CLI is required. Install from: https://cli.github.com")
		return fmt.Errorf("gh CLI not found")
	}
	if !isVersionAtLeast(ghVersion, "2.40.0") {
		fmt.Printf("v%s (TOO OLD)\n", ghVersion)
		return fmt.Errorf("gh CLI v2.40.0+ required for multi-account support (found v%s)", ghVersion)
	}
	fmt.Printf("v%s\n", ghVersion)

	// 2. Check direnv
	fmt.Print("  Checking direnv............. ")
	if !direnvlib.IsInstalled() {
		fmt.Println("MISSING")
		fmt.Println()
		fmt.Println("  direnv is required. Install it:")
		fmt.Println("    Arch:   sudo pacman -S direnv")
		fmt.Println("    Ubuntu: sudo apt install direnv")
		fmt.Println("    macOS:  brew install direnv")
		fmt.Println()
		fmt.Println("  Then add the shell hook:")
		fmt.Println("    bash:  echo 'eval \"$(direnv hook bash)\"' >> ~/.bashrc")
		fmt.Println("    zsh:   echo 'eval \"$(direnv hook zsh)\"' >> ~/.zshrc")
		fmt.Println("    fish:  echo 'direnv hook fish | source' >> ~/.config/fish/config.fish")
		return fmt.Errorf("direnv not found")
	}
	direnvVersion, _ := direnvlib.GetVersion()
	fmt.Printf("v%s\n", direnvVersion)

	// 3. Check direnv shell hook
	fmt.Print("  Checking direnv hook........ ")
	if direnvlib.CheckDirenvHook() {
		fmt.Println("OK")
	} else {
		fmt.Println("NOT DETECTED")
		fmt.Println("    Add the direnv hook to your shell config:")
		fmt.Println("      bash: eval \"$(direnv hook bash)\"")
		fmt.Println("      zsh:  eval \"$(direnv hook zsh)\"")
		fmt.Println("      fish: direnv hook fish | source")
		allGood = false
	}

	// 4. Check logged-in accounts
	fmt.Print("  Checking gh accounts........ ")
	users, err := ghauth.ListUsers()
	if err != nil {
		fmt.Println("ERROR")
		fmt.Printf("    %v\n", err)
		allGood = false
	} else {
		fmt.Printf("%d account(s)\n", len(users))
		for _, u := range users {
			marker := "  "
			if u.Active {
				marker = "* "
			}
			fmt.Printf("    %s%s (%s)\n", marker, u.User, u.Host)
		}
		if len(users) < 2 {
			fmt.Println("    Tip: Log in to more accounts with: gh auth login")
		}
	}

	// 5. Install direnv shell library
	fmt.Println()
	fmt.Print("  Installing direnv lib....... ")
	if err := direnvlib.InstallShellLib(); err != nil {
		fmt.Println("FAILED")
		return fmt.Errorf("cannot install shell library: %w", err)
	}
	libPath, _ := direnvlib.ShellLibPath()
	fmt.Println("OK")
	fmt.Printf("    Installed: %s\n", libPath)

	// 6. Install shell hook (wrapper mode support)
	fmt.Print("  Installing shell hook....... ")
	hookPath, err := direnvlib.InstallShellHook()
	if err != nil {
		fmt.Println("FAILED")
		return fmt.Errorf("cannot install shell hook: %w", err)
	}
	fmt.Println("OK")
	fmt.Printf("    Installed: %s\n", hookPath)

	// 7. Inject hook source into shell RC file
	fmt.Print("  Configuring shell RC........ ")
	rcPath, err := detectShellRC()
	if err != nil {
		fmt.Println("SKIPPED")
		fmt.Printf("    %v\n", err)
		fmt.Printf("    Add manually to your shell RC:\n")
		fmt.Printf("      source \"%s\"\n", hookPath)
		allGood = false
	} else {
		if direnvlib.CheckShellHookInstalled() {
			fmt.Println("OK (already configured)")
		} else {
			if err := direnvlib.InjectHookSource(rcPath, hookPath); err != nil {
				fmt.Println("FAILED")
				fmt.Printf("    %v\n", err)
				fmt.Printf("    Add manually to %s:\n", rcPath)
				fmt.Printf("      source \"%s\"\n", hookPath)
				allGood = false
			} else {
				fmt.Println("OK")
				fmt.Printf("    Added to: %s\n", rcPath)
			}
		}
	}

	if migrate {
		fmt.Print("  Running migration.......... ")
		report, err := runMigration()
		if err != nil {
			fmt.Println("FAILED")
			fmt.Printf("    %v\n", err)
			allGood = false
		} else {
			fmt.Println("OK")
			fmt.Printf("    Pins updated: %d, .envrc rewritten: %d, direnv allow: %d\n", report.PinsUpdated, report.EnvrcRewritten, report.Allowed)
			if report.Warnings > 0 {
				fmt.Printf("    Warnings: %d (directories missing or not writable)\n", report.Warnings)
				allGood = false
			}
		}
	}

	// Summary
	fmt.Println()
	if allGood {
		fmt.Println("  Setup complete! Pin accounts to directories with:")
		fmt.Println("    gh autoprofile pin <username> --dir <path>")
		fmt.Println()
		fmt.Println("  Token mode:")
		fmt.Println("    wrapper (default)  — token injected per-command, never in env")
		fmt.Println("    export             — GH_TOKEN exported (use --export-token flag)")
		fmt.Println()
		if migrate {
			fmt.Println("  Migration complete.")
		}
		fmt.Println("  Restart your shell or run: source " + rcPath)
	} else {
		fmt.Println("  Setup complete with warnings (see above).")
		fmt.Println("  Fix the warnings, then pin accounts with:")
		fmt.Println("    gh autoprofile pin <username> --dir <path>")
	}

	return nil
}

type migrationReport struct {
	PinsUpdated    int
	EnvrcRewritten int
	Allowed        int
	Warnings       int
}

func runMigration() (migrationReport, error) {
	var report migrationReport

	registry, err := config.LoadPins()
	if err != nil {
		return report, fmt.Errorf("cannot load pin registry: %w", err)
	}

	configDir, err := config.ConfigDir()
	if err == nil {
		_ = os.Chmod(configDir, 0700)
	}
	if pinsPath, err := config.PinsFilePath(); err == nil {
		if _, statErr := os.Stat(pinsPath); statErr == nil {
			_ = os.Chmod(pinsPath, 0600)
		}
	}

	updatedMode := 0
	for i := range registry.Pins {
		if registry.Pins[i].Mode == "" {
			registry.Pins[i].Mode = config.ModeWrapper
			updatedMode++
		}
	}
	if updatedMode > 0 {
		if err := config.SavePins(registry); err != nil {
			return report, fmt.Errorf("cannot save migrated pins: %w", err)
		}
	}
	report.PinsUpdated = updatedMode

	for _, pin := range registry.Pins {
		if _, err := os.Stat(pin.Dir); err != nil {
			report.Warnings++
			continue
		}

		if err := direnvlib.WriteEnvrc(pin); err != nil {
			report.Warnings++
			continue
		}
		report.EnvrcRewritten++

		if err := direnvlib.AllowEnvrc(pin.Dir); err != nil {
			report.Warnings++
			continue
		}
		report.Allowed++
	}

	return report, nil
}

// detectShellRC finds the user's active shell RC file.
func detectShellRC() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}

	// Check SHELL env var first.
	shell := os.Getenv("SHELL")
	if strings.HasSuffix(shell, "/zsh") {
		return filepath.Join(home, ".zshrc"), nil
	}
	if strings.HasSuffix(shell, "/bash") {
		return filepath.Join(home, ".bashrc"), nil
	}

	// Fallback: check which RC files exist.
	for _, name := range []string{".zshrc", ".bashrc", ".bash_profile", ".profile"} {
		p := filepath.Join(home, name)
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	return "", fmt.Errorf("could not detect shell RC file (SHELL=%s)", shell)
}

// isVersionAtLeast compares semver strings (major.minor.patch).
func isVersionAtLeast(current, minimum string) bool {
	cParts := strings.Split(current, ".")
	mParts := strings.Split(minimum, ".")

	for i := 0; i < 3; i++ {
		var c, m int
		if i < len(cParts) {
			fmt.Sscanf(cParts[i], "%d", &c)
		}
		if i < len(mParts) {
			fmt.Sscanf(mParts[i], "%d", &m)
		}
		if c > m {
			return true
		}
		if c < m {
			return false
		}
	}
	return true // equal
}
