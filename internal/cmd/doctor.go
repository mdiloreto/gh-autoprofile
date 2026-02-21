package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mdiloreto/gh-autoprofile/internal/config"
	direnvlib "github.com/mdiloreto/gh-autoprofile/internal/direnv"
	"github.com/spf13/cobra"
)

// NewDoctorCmd creates the `doctor` subcommand.
func NewDoctorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Check setup and migration health",
		Long:  "Validate shell integration, pin schema, and managed .envrc permissions.",
		RunE:  runDoctor,
	}
	cmd.Flags().Bool("fix", false, "Run setup migration automatically")
	return cmd
}

func runDoctor(cmd *cobra.Command, args []string) error {
	fix, err := cmd.Flags().GetBool("fix")
	if err != nil {
		return err
	}
	if fix {
		setupCmd := NewSetupCmd()
		if err := setupCmd.Flags().Set("migrate", "true"); err != nil {
			return err
		}
		return runSetup(setupCmd, args)
	}

	issues := 0
	fmt.Println("gh-autoprofile doctor")
	fmt.Println("=====================")

	if direnvlib.IsShellLibInstalled() {
		fmt.Println("OK   direnv library installed")
	} else {
		fmt.Println("WARN direnv library missing")
		issues++
	}

	if direnvlib.CheckShellHookInstalled() {
		fmt.Println("OK   shell hook source configured")
	} else {
		fmt.Println("WARN shell hook source not detected")
		issues++
	}

	registry, err := config.LoadPins()
	if err != nil {
		return fmt.Errorf("cannot load pins: %w", err)
	}

	missingModes := 0
	envrcPermIssues := 0
	for _, pin := range registry.Pins {
		if pin.Mode == "" {
			missingModes++
		}
		envrcPath := filepath.Join(pin.Dir, ".envrc")
		if fi, err := os.Stat(envrcPath); err == nil {
			if fi.Mode().Perm() != 0600 {
				envrcPermIssues++
			}
		}
	}

	if missingModes == 0 {
		fmt.Println("OK   pin modes normalized")
	} else {
		fmt.Printf("WARN %d pin(s) missing mode (will default to wrapper)\n", missingModes)
		issues++
	}

	if envrcPermIssues == 0 {
		fmt.Println("OK   managed .envrc permissions are 0600")
	} else {
		fmt.Printf("WARN %d managed .envrc file(s) not 0600\n", envrcPermIssues)
		issues++
	}

	if issues == 0 {
		fmt.Println("\nDoctor check passed.")
		return nil
	}

	fmt.Println("\nRun `gh autoprofile setup --migrate` to fix detected issues.")
	return nil
}
