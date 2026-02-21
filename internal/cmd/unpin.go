package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/mdiloreto/gh-autoprofile/internal/config"
	direnvlib "github.com/mdiloreto/gh-autoprofile/internal/direnv"
	"github.com/spf13/cobra"
)

// NewUnpinCmd creates the `unpin` subcommand.
func NewUnpinCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "unpin [directory]",
		Short: "Remove a pinned account from a directory",
		Long: `Remove the gh-autoprofile block from the directory's .envrc and
delete the pin from the registry. If the .envrc has no other content,
it will be deleted entirely.

Examples:
  gh autoprofile unpin              # unpin current directory
  gh autoprofile unpin ~/carto      # unpin specific directory`,
		Args: cobra.MaximumNArgs(1),
		RunE: runUnpin,
	}
}

func runUnpin(cmd *cobra.Command, args []string) error {
	dir := "."
	if len(args) > 0 {
		dir = args[0]
	}

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("cannot resolve directory: %w", err)
	}

	// Load registry
	registry, err := config.LoadPins()
	if err != nil {
		return fmt.Errorf("cannot load pin registry: %w", err)
	}

	// Check if directory is pinned
	pin := registry.FindPin(absDir)
	if pin == nil {
		return fmt.Errorf("no pin found for directory: %s", absDir)
	}

	user := pin.User

	// Remove from registry
	registry.RemovePin(absDir)
	if err := config.SavePins(registry); err != nil {
		return fmt.Errorf("cannot save pin registry: %w", err)
	}

	// Remove .envrc block
	if err := direnvlib.RemoveEnvrc(absDir); err != nil {
		return fmt.Errorf("cannot clean .envrc: %w", err)
	}

	fmt.Printf("Unpinned '%s' from %s\n", user, absDir)
	return nil
}
