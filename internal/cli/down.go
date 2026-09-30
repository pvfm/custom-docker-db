package cli

import "github.com/spf13/cobra"

func newDownCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "down",
		Short: "Para e remove o container, mantendo o volume",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return errNotImplemented
		},
	}
}
