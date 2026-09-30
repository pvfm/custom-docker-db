package cli

import (
	"errors"

	"github.com/spf13/cobra"
)

var errNotImplemented = errors.New("ainda não implementado")

func newUpCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "up",
		Short: "Detecta a config do .env e sobe o Postgres (padrão)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := resolveConfig(cmd)
			if err != nil {
				return err
			}
			printConfig(cmd, cfg)
			return errNotImplemented // compose e subida: M4
		},
	}
	cmd.Flags().String("env-file", "", "usa apenas este arquivo .env, sem descoberta automática")
	cmd.Flags().Bool("ephemeral", false, "sobe sem volume: os dados somem no down")
	return cmd
}
