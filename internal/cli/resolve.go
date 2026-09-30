package cli

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/pvfm/custom-docker-db/internal/envconfig"
)

// resolveConfig loads the database config for the current directory. When the
// project has none, it offers to create the default and loads it again.
func resolveConfig(cmd *cobra.Command, in io.Reader) (envconfig.Config, error) {
	dir, err := os.Getwd()
	if err != nil {
		return envconfig.Config{}, err
	}
	envFile, _ := cmd.Flags().GetString("env-file")
	name := envconfig.DefaultName(dir)

	cfg, err := envconfig.Load(dir, envFile, name)
	var nc *envconfig.NoConfigError
	if !errors.As(err, &nc) {
		return cfg, err
	}
	if _, err := envconfig.OfferDefaults(in, cmd.OutOrStdout(), nc, dir, name); err != nil {
		return envconfig.Config{}, err
	}
	return envconfig.Load(dir, envFile, name)
}

func printConfig(cmd *cobra.Command, cfg envconfig.Config) {
	fmt.Fprintf(cmd.OutOrStdout(), "Config de %s: usuário=%s banco=%s porta=%d\n",
		cfg.Source, cfg.User, cfg.Name, cfg.Port)
}
