package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pvfm/custom-docker-db/internal/envconfig"
)

func TestHelpListsCommandsAndFlags(t *testing.T) {
	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"up", "down", "exec", "--env-file", "--ephemeral"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("help não menciona %q:\n%s", want, out.String())
		}
	}
}

func TestSubcommandsReturnNotImplemented(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("DB_USER=alice\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)

	for _, args := range [][]string{{}, {"up"}, {"down"}, {"exec"}} {
		cmd := NewRootCmd()
		cmd.SetOut(&bytes.Buffer{})
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetArgs(args)
		if err := cmd.Execute(); err != errNotImplemented {
			t.Errorf("args %v: erro = %v, esperado errNotImplemented", args, err)
		}
	}
}

func TestUpOffersDefaultInEmptyDir(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetIn(strings.NewReader("s\n"))
	cmd.SetArgs([]string{"up"})
	if err := cmd.Execute(); err != errNotImplemented {
		t.Fatalf("erro = %v\n%s", err, out.String())
	}
	got, err := os.ReadFile(filepath.Join(dir, ".env"))
	if err != nil || !strings.Contains(string(got), "DB_PORT=5432") {
		t.Fatalf(".env = %q, err = %v", got, err)
	}
	if !strings.Contains(out.String(), "Config de") {
		t.Errorf("saída sem a config detectada:\n%s", out.String())
	}
}

func TestUpDeclinedDefaultStops(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetIn(strings.NewReader("n\n"))
	cmd.SetArgs([]string{"up"})
	if err := cmd.Execute(); !errors.Is(err, envconfig.ErrDeclined) {
		t.Fatalf("erro = %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".env")); err == nil {
		t.Error(".env não deveria existir")
	}
}
