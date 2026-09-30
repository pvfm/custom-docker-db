package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/pvfm/custom-docker-db/internal/compose"
)

func newDownCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "down",
		Short: "Para e remove o container, mantendo o volume",
		Args:  cobra.NoArgs,
		RunE:  runDown,
	}
}

func runDown(cmd *cobra.Command, args []string) error {
	out := cmd.OutOrStdout()
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	d := newDocker()
	if err := d.Preflight(); err != nil {
		return err
	}
	paths, err := compose.Locate(dir)
	if err != nil {
		return err
	}
	labels, exists, err := d.ContainerLabels(paths.Project)
	if err != nil {
		return err
	}
	if !exists {
		fmt.Fprintln(out, "Nenhum banco encontrado para este projeto; nada a fazer.")
		return nil
	}

	// In ephemeral mode the image's anonymous volume must go with the container.
	ephemeral := labels[compose.EphemeralLabel] == "true"
	if output, err := d.Down(paths.File, paths.Project, ephemeral); err != nil {
		return fmt.Errorf("docker compose down: %w\n%s", err, output)
	}
	if ephemeral {
		fmt.Fprintln(out, "Banco removido; os dados efêmeros foram apagados.")
	} else {
		fmt.Fprintln(out, "Banco removido; o volume foi mantido e volta no próximo up.")
	}
	return nil
}
