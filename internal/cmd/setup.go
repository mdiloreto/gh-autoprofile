package cmd

import (
	"fmt"
	"strings"

	direnvlib "github.com/mdiloreto/gh-autoprofile/internal/direnv"
	"github.com/mdiloreto/gh-autoprofile/internal/ghauth"
	"github.com/spf13/cobra"
)

// NewSetupCmd creates the `setup` subcommand.
func NewSetupCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "setup",
		Short: "Install direnv shell library and validate prerequisites",
		Long: `Checks that gh CLI (>= 2.40.0) and direnv are installed, then installs
the gh-autoprofile shell library into direnv's lib directory.

Run this once after installing gh-autoprofile.`,
		RunE: runSetup,
	}
}

func runSetup(cmd *cobra.Command, args []string) error {
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
	fmt.Print("  Checking shell hook......... ")
	if direnvlib.CheckShellHook() {
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

	// 5. Install shell library
	fmt.Println()
	fmt.Print("  Installing shell library.... ")
	if err := direnvlib.InstallShellLib(); err != nil {
		fmt.Println("FAILED")
		return fmt.Errorf("cannot install shell library: %w", err)
	}
	libPath, _ := direnvlib.ShellLibPath()
	fmt.Println("OK")
	fmt.Printf("    Installed: %s\n", libPath)

	// Summary
	fmt.Println()
	if allGood {
		fmt.Println("  Setup complete! Pin accounts to directories with:")
		fmt.Println("    gh autoprofile pin <username> --dir <path>")
	} else {
		fmt.Println("  Setup complete with warnings (see above).")
		fmt.Println("  Fix the warnings, then pin accounts with:")
		fmt.Println("    gh autoprofile pin <username> --dir <path>")
	}

	return nil
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
