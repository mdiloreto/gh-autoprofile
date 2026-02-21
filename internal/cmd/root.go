package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mdiloreto/gh-autoprofile/internal/config"
	direnvlib "github.com/mdiloreto/gh-autoprofile/internal/direnv"
	"github.com/spf13/cobra"
)

// Version is set at build time via ldflags.
var Version = "dev"

// NewRootCmd creates the top-level cobra command for gh-autoprofile.
func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gh-autoprofile",
		Short: "Automatic GitHub profile switching per directory",
		Long: `gh-autoprofile pins a GitHub account to a directory. When you cd in,
direnv activates the right profile context and git identity —
no manual switching needed.

Quick start:
  gh autoprofile setup                           # one-time setup
  gh autoprofile pin <user> --dir <path>         # pin an account
  cd <path> && gh api /user --jq .login          # verify it works`,
		Version:       Version,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			subcmd := ""
			if len(os.Args) > 1 {
				subcmd = os.Args[1]
			}
			if subcmd == "setup" || subcmd == "doctor" || subcmd == "help" || subcmd == "completion" {
				return nil
			}
			warnUpgradeDrift(cmd)
			return nil
		},
	}

	cmd.AddCommand(
		NewSetupCmd(),
		NewPinCmd(),
		NewUnpinCmd(),
		NewListCmd(),
		NewStatusCmd(),
		NewDoctorCmd(),
	)

	return cmd
}

func warnUpgradeDrift(cmd *cobra.Command) {
	registry, err := config.LoadPins()
	if err != nil {
		return
	}

	needsSetup := !direnvlib.IsShellLibInstalled() || !direnvlib.CheckShellHookInstalled()
	needsModeMigration := false
	needsEnvrcPerms := false

	for _, pin := range registry.Pins {
		if pin.Mode == "" {
			needsModeMigration = true
		}
		envrcPath := filepath.Join(pin.Dir, ".envrc")
		if fi, err := os.Stat(envrcPath); err == nil {
			if fi.Mode().Perm() != 0600 {
				needsEnvrcPerms = true
			}
		}
	}

	if !(needsSetup || needsModeMigration || needsEnvrcPerms) {
		return
	}

	fmt.Fprintln(cmd.ErrOrStderr(), "gh-autoprofile: upgrade tasks detected. Run `gh autoprofile setup --migrate` to apply security migrations.")
}
