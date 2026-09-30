package cli

import (
	"errors"
	"os"

	"github.com/spf13/cobra"

	"github.com/pvfm/custom-docker-db/internal/compose"
	"github.com/pvfm/custom-docker-db/internal/engine"
)

func newExecCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "exec",
		Short: "Abre um shell (sh) dentro do container",
		Args:  cobra.NoArgs,
		RunE:  runExec,
	}
}

func runExec(cmd *cobra.Command, args []string) error {
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
	_, exists, err := d.ContainerLabels(paths.Project)
	if err != nil {
		return err
	}
	if exists {
		running, err := d.ContainerRunning(paths.Project)
		if err != nil {
			return err
		}
		exists = running
	}
	if !exists {
		return errors.New("o banco não está rodando; suba com `custom-docker-db up`")
	}
	return d.Exec(paths.Project, engine.Postgres.Shell)
}
