package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/mdiloreto/gh-autoprofile/internal/config"
	"github.com/spf13/cobra"
)

// NewListCmd creates the `list` subcommand.
func NewListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List all pinned directories",
		Long:    `Show a table of all directories with pinned GitHub accounts.`,
		RunE:    runList,
	}
}

func runList(cmd *cobra.Command, args []string) error {
	registry, err := config.LoadPins()
	if err != nil {
		return fmt.Errorf("cannot load pin registry: %w", err)
	}

	if len(registry.Pins) == 0 {
		fmt.Println("No pinned directories.")
		fmt.Println("Pin one with: gh autoprofile pin <username> --dir <path>")
		return nil
	}

	// Get current directory for highlighting
	cwd, _ := os.Getwd()

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "DIRECTORY\tACCOUNT\tGIT EMAIL\tGIT NAME\tSSH KEY")
	fmt.Fprintln(w, "---------\t-------\t---------\t--------\t-------")

	for _, pin := range registry.Pins {
		marker := " "
		if pin.Dir == cwd {
			marker = "*"
		}

		email := pin.GitEmail
		if email == "" {
			email = "-"
		}
		name := pin.GitName
		if name == "" {
			name = "-"
		}
		sshKey := pin.SSHKey
		if sshKey == "" {
			sshKey = "-"
		}

		fmt.Fprintf(w, "%s %s\t%s\t%s\t%s\t%s\n", marker, pin.Dir, pin.User, email, name, sshKey)
	}

	w.Flush()
	fmt.Printf("\n%d pin(s) total. (* = current directory)\n", len(registry.Pins))

	return nil
}
