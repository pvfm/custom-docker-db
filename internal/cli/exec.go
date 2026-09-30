package cli

import "github.com/spf13/cobra"

func newExecCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "exec",
		Short: "Abre um shell (sh) dentro do container",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return errNotImplemented
		},
	}
}
