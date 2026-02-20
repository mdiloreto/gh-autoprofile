package cmd

import (
	"fmt"
	"os"

	"github.com/mdiloreto/gh-autoprofile/internal/config"
	direnvlib "github.com/mdiloreto/gh-autoprofile/internal/direnv"
	"github.com/mdiloreto/gh-autoprofile/internal/ghauth"
	"github.com/spf13/cobra"
)

// NewStatusCmd creates the `status` subcommand.
func NewStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show the active profile context for the current directory",
		Long: `Display the current directory's pinned account, active GH_TOKEN,
git identity, and any mismatches between expected and actual state.`,
		RunE: runStatus,
	}
}

func runStatus(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("cannot get current directory: %w", err)
	}

	fmt.Printf("Directory: %s\n", cwd)
	fmt.Println()

	// Check pin registry
	registry, err := config.LoadPins()
	if err != nil {
		return fmt.Errorf("cannot load pin registry: %w", err)
	}

	pin := registry.FindPin(cwd)

	// Pinned account
	if pin != nil {
		fmt.Printf("  Pinned account:   %s\n", pin.User)
		if pin.GitEmail != "" {
			fmt.Printf("  Pinned email:     %s\n", pin.GitEmail)
		}
		if pin.GitName != "" {
			fmt.Printf("  Pinned name:      %s\n", pin.GitName)
		}
		if pin.SSHKey != "" {
			fmt.Printf("  Pinned SSH key:   %s\n", pin.SSHKey)
		}
	} else {
		fmt.Println("  Pinned account:   (none)")
	}
	fmt.Println()

	// Active environment
	ghToken := os.Getenv("GH_TOKEN")
	githubToken := os.Getenv("GITHUB_TOKEN")
	gitEmail := os.Getenv("GIT_AUTHOR_EMAIL")
	gitName := os.Getenv("GIT_AUTHOR_NAME")
	gitSSH := os.Getenv("GIT_SSH_COMMAND")

	fmt.Println("  Environment:")
	if ghToken != "" {
		// Mask the token — show first 4 and last 4 chars
		masked := maskToken(ghToken)
		fmt.Printf("    GH_TOKEN:             %s\n", masked)
	} else {
		fmt.Println("    GH_TOKEN:             (not set)")
	}
	if githubToken != "" {
		masked := maskToken(githubToken)
		fmt.Printf("    GITHUB_TOKEN:         %s\n", masked)
	} else {
		fmt.Println("    GITHUB_TOKEN:         (not set)")
	}
	if gitEmail != "" {
		fmt.Printf("    GIT_AUTHOR_EMAIL:     %s\n", gitEmail)
	} else {
		fmt.Println("    GIT_AUTHOR_EMAIL:     (not set)")
	}
	if gitName != "" {
		fmt.Printf("    GIT_AUTHOR_NAME:      %s\n", gitName)
	} else {
		fmt.Println("    GIT_AUTHOR_NAME:      (not set)")
	}
	if gitSSH != "" {
		fmt.Printf("    GIT_SSH_COMMAND:      %s\n", gitSSH)
	}
	fmt.Println()

	// Active gh user (from gh auth status, not env)
	fmt.Print("  Active gh user:   ")
	users, err := ghauth.ListUsers()
	if err != nil {
		fmt.Printf("(error: %v)\n", err)
	} else {
		found := false
		for _, u := range users {
			if u.Active {
				fmt.Printf("%s (%s)\n", u.User, u.Host)
				found = true
				break
			}
		}
		if !found {
			fmt.Println("(none active)")
		}
	}

	// Diagnostics
	fmt.Println()
	if pin != nil && ghToken == "" {
		fmt.Println("  WARNING: Directory is pinned but GH_TOKEN is not set.")
		fmt.Println("           Is direnv loaded? Try: cd . (to re-trigger direnv)")
		if !direnvlib.IsInstalled() {
			fmt.Println("           direnv is not installed!")
		} else if !direnvlib.IsShellLibInstalled() {
			fmt.Println("           Shell library not installed. Run: gh autoprofile setup")
		} else if !direnvlib.CheckShellHook() {
			fmt.Println("           direnv shell hook not detected in your shell config.")
		}
	} else if pin != nil && ghToken != "" {
		fmt.Println("  Profile is active and GH_TOKEN is set.")
	} else if pin == nil && ghToken != "" {
		fmt.Println("  NOTE: GH_TOKEN is set but this directory has no pin.")
		fmt.Println("        The token may come from another source (parent .envrc, export, etc.)")
	} else {
		fmt.Println("  No pin and no GH_TOKEN. Using default gh account.")
	}

	return nil
}

// maskToken shows the first 4 and last 4 characters of a token.
func maskToken(token string) string {
	if len(token) <= 10 {
		return "****"
	}
	return token[:4] + "****" + token[len(token)-4:]
}
