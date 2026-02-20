package cmd

import (
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
direnv automatically exports the right GH_TOKEN and git identity —
no manual switching needed.

Quick start:
  gh autoprofile setup                           # one-time setup
  gh autoprofile pin <user> --dir <path>         # pin an account
  cd <path> && gh api /user --jq .login          # verify it works`,
		Version:       Version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.AddCommand(
		NewSetupCmd(),
		NewPinCmd(),
		NewUnpinCmd(),
		NewListCmd(),
		NewStatusCmd(),
	)

	return cmd
}
