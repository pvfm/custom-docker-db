// Package cli defines the custom-docker-db command tree.
package cli

import "github.com/spf13/cobra"

// Version is set at build time via -ldflags.
var Version = "dev"

// NewRootCmd builds the root command. Running it without a subcommand is
// equivalent to `up`.
func NewRootCmd() *cobra.Command {
	up := newUpCmd()

	root := &cobra.Command{
		Use:           "custom-docker-db",
		Short:         "Sobe um Postgres em Docker a partir da config do seu .env",
		Version:       Version,
		SilenceUsage:  true,
		SilenceErrors: false,
		RunE:          up.RunE,
	}
	// Flags of `up` are also accepted on the root command.
	root.Flags().AddFlagSet(up.Flags())

	root.AddCommand(up, newDownCmd(), newExecCmd())
	return root
}
