package envconfig

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pvfm/custom-docker-db/internal/prompt"
)

// ErrDeclined means the user chose not to create the default config.
var ErrDeclined = errors.New("config padrão não criada")

var unsafeName = regexp.MustCompile(`[^A-Za-z0-9_-]`)

// DefaultName derives the default database name from a directory path.
func DefaultName(dir string) string {
	abs, err := filepath.Abs(dir)
	if err == nil {
		dir = abs
	}
	name := unsafeName.ReplaceAllString(filepath.Base(dir), "_")
	if name == "" || name == "_" {
		return "postgres"
	}
	return name
}

// DefaultBlock is the config written when the project has none.
func DefaultBlock(dbName string) string {
	return fmt.Sprintf("DB_USER=%s\nDB_PASSWORD=%s\nDB_NAME=%s\nDB_PORT=%d\n",
		DefaultUser, DefaultPassword, dbName, DefaultPort)
}

// OfferDefaults shows the expected config, asks for confirmation and writes it.
// With an explicit --env-file the block is appended to that file; otherwise it
// is appended to ./.env, which is created when missing. It returns the path
// written to, or ErrDeclined.
func OfferDefaults(in io.Reader, out io.Writer, nc *NoConfigError, dir, dbName string) (string, error) {
	target := nc.File
	if target == "" {
		target = filepath.Join(dir, ".env")
	}
	block := DefaultBlock(dbName)

	_, statErr := os.Stat(target)
	exists := statErr == nil

	fmt.Fprintln(out, nc.Error()+".")
	fmt.Fprintln(out, "A CLI espera este padrão:")
	fmt.Fprintln(out)
	for _, l := range strings.Split(strings.TrimRight(block, "\n"), "\n") {
		fmt.Fprintln(out, "  "+l)
	}
	fmt.Fprintln(out)
	question := fmt.Sprintf("Criar %s com esse padrão? [s/N] ", target)
	if exists {
		question = fmt.Sprintf("Acrescentar ao final de %s? [s/N] ", target)
	}
	if !prompt.Confirm(in, out, question) {
		fmt.Fprintln(out)
		return "", ErrDeclined
	}

	if err := writeBlock(target, block, exists); err != nil {
		return "", err
	}
	verb := "Criado"
	if exists {
		verb = "Acrescentado em"
	}
	fmt.Fprintf(out, "%s %s (4 variáveis).\n", verb, target)
	return target, nil
}

func writeBlock(path, block string, exists bool) error {
	if !exists {
		return os.WriteFile(path, []byte(block), 0o600)
	}
	prev, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	defer f.Close()
	prefix := ""
	if len(prev) > 0 && prev[len(prev)-1] != '\n' {
		prefix = "\n"
	}
	if _, err := f.WriteString(prefix + block); err != nil {
		return err
	}
	return f.Close()
}
