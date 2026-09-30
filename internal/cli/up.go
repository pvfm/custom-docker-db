package cli

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/pvfm/custom-docker-db/internal/compose"
	"github.com/pvfm/custom-docker-db/internal/docker"
	"github.com/pvfm/custom-docker-db/internal/engine"
	"github.com/pvfm/custom-docker-db/internal/prompt"
)

// newDocker builds the docker client; tests replace it with a fake.
var newDocker = docker.New

const healthTimeout = 60 * time.Second

func newUpCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "up",
		Short: "Detecta a config do .env e sobe o Postgres (padrão)",
		Args:  cobra.NoArgs,
		RunE:  runUp,
	}
	cmd.Flags().String("env-file", "", "usa apenas este arquivo .env, sem descoberta automática")
	cmd.Flags().Bool("ephemeral", false, "sobe sem volume: os dados somem no down")
	return cmd
}

// change says what to do with an existing container whose config differs.
type change int

const (
	changeNone     change = iota // same config: nothing to do
	changeRecreate               // recreate, keep the named volume
	changeWipe                   // recreate and drop disposable data
	changeConfirm                // persistent data would be lost: ask first
)

// planChange compares the labels of the existing container with the config
// about to be started.
func planChange(labels map[string]string, hash string, ephemeral bool) change {
	oldEphemeral := labels[compose.EphemeralLabel] == "true"
	if labels[compose.ConfigLabel] == hash && oldEphemeral == ephemeral {
		return changeNone
	}
	switch {
	case oldEphemeral:
		// The image keeps an anonymous volume across recreations, so the old
		// credentials would survive unless it is dropped. The data is disposable.
		return changeWipe
	case ephemeral:
		// The named volume stays on disk, untouched.
		return changeRecreate
	default:
		return changeConfirm
	}
}

func runUp(cmd *cobra.Command, args []string) error {
	out := cmd.OutOrStdout()
	in := bufio.NewReader(cmd.InOrStdin())

	cfg, err := resolveConfig(cmd, in)
	if err != nil {
		return err
	}
	printConfig(cmd, cfg)

	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	ephemeral, _ := cmd.Flags().GetBool("ephemeral")
	eng := engine.Postgres
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
	running := false
	ownsPort := false
	if exists {
		if running, err = d.ContainerRunning(paths.Project); err != nil {
			return err
		}
		hash := compose.ConfigHash(eng, cfg)
		portChanged := labels[compose.PortLabel] != fmt.Sprint(cfg.Port)
		switch planChange(labels, hash, ephemeral) {
		case changeConfirm:
			q := fmt.Sprintf("\nA config de banco mudou desde a última execução. O %s só aplica usuário, senha e banco\n"+
				"quando cria o volume, então é preciso apagar o volume (e os dados) para recriar o banco.\n"+
				"Apagar o volume e recriar? [s/N] ", strings.Title(eng.Name))
			if !prompt.Confirm(in, out, q) {
				return errors.New("config alterada; nada foi feito (restaure o valor anterior no .env ou confirme para recriar)")
			}
			if err := recreate(d, paths, true); err != nil {
				return err
			}
			running = false
		case changeWipe:
			fmt.Fprintln(out, "Config alterada: recriando o container e descartando os dados efêmeros.")
			if err := recreate(d, paths, true); err != nil {
				return err
			}
			running = false
		case changeRecreate:
			fmt.Fprintln(out, "Config alterada: recriando o container e mantendo o volume.")
			if err := recreate(d, paths, false); err != nil {
				return err
			}
			running = false
		default:
			if running && portChanged {
				// compose up recreates the container on its own and keeps the volume.
				fmt.Fprintln(out, "Porta alterada: o container será recriado mantendo o volume.")
			}
		}
		ownsPort = running && !portChanged
	}

	// Our own running container already holds its port; check every other case.
	if !ownsPort && !d.PortFree(cfg.Port) {
		return fmt.Errorf("a porta %d já está em uso; libere a porta ou mude DB_PORT em %s (nada foi feito)", cfg.Port, cfg.Source)
	}

	if err := compose.Write(paths, compose.Spec{Engine: eng, Config: cfg, Project: paths.Project, Ephemeral: ephemeral}); err != nil {
		return fmt.Errorf("escrevendo o compose: %w", err)
	}

	fmt.Fprintf(out, "Subindo o %s (aguardando o healthcheck, até %d s)...\n", strings.Title(eng.Name), int(healthTimeout.Seconds()))
	if output, err := d.Up(paths.File, paths.Project, healthTimeout); err != nil {
		fmt.Fprintln(cmd.ErrOrStderr(), output)
		if logs := d.Logs(paths.File, paths.Project, 20); logs != "" {
			fmt.Fprintln(cmd.ErrOrStderr(), "\nÚltimas linhas dos logs:\n"+logs)
		}
		return fmt.Errorf("o banco não ficou saudável: %w", err)
	}

	fmt.Fprintf(out, "Banco pronto: %s\n", eng.URL(cfg, true))
	if ephemeral {
		fmt.Fprintln(out, "Modo efêmero: os dados somem no down.")
	}
	fmt.Fprintf(out, "Compose: %s\n", paths.File)
	return nil
}

func recreate(d docker.Client, paths compose.Paths, wipe bool) error {
	if output, err := d.Down(paths.File, paths.Project, wipe); err != nil {
		return fmt.Errorf("removendo o container antigo: %w\n%s", err, output)
	}
	return nil
}
