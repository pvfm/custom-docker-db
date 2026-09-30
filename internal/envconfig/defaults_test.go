package envconfig

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultName(t *testing.T) {
	cases := map[string]string{
		"/x/meu-app":   "meu-app",
		"/x/meu app":   "meu_app",
		"/x/api.v2":    "api_v2",
		"/x/loja_2024": "loja_2024",
	}
	for in, want := range cases {
		if got := DefaultName(in); got != want {
			t.Errorf("DefaultName(%q) = %q, esperado %q", in, got, want)
		}
	}
}

func offer(t *testing.T, answer string, nc *NoConfigError, dir string) (string, string, error) {
	t.Helper()
	var out bytes.Buffer
	path, err := OfferDefaults(strings.NewReader(answer), &out, nc, dir, "shop")
	return path, out.String(), err
}

func TestOfferCreatesEnvAndLoadWorks(t *testing.T) {
	d := t.TempDir()
	_, err := Load(d, "", "shop")
	var nc *NoConfigError
	if !errors.As(err, &nc) {
		t.Fatalf("erro = %v", err)
	}
	path, out, err := offer(t, "s\n", nc, d)
	if err != nil {
		t.Fatal(err)
	}
	if path != filepath.Join(d, ".env") || !strings.Contains(out, "Criar") || !strings.Contains(out, "DB_NAME=shop") || !strings.Contains(out, "Criado") {
		t.Fatalf("path = %s\nsaída:\n%s", path, out)
	}
	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0o600 {
		t.Errorf("permissão = %v", info.Mode().Perm())
	}
	cfg, err := Load(d, "", "shop")
	if err != nil || cfg.User != "postgres" || cfg.Name != "shop" || cfg.Port != 5432 {
		t.Fatalf("cfg = %+v, err = %v", cfg, err)
	}
}

func TestOfferAppendsToExistingEnv(t *testing.T) {
	d := t.TempDir()
	p := write(t, d, ".env", "API_KEY=abc") // sem newline final
	var nc *NoConfigError
	_, err := Load(d, "", "shop")
	errors.As(err, &nc)
	_, out, err := offer(t, "sim\n", nc, d)
	if err != nil || !strings.Contains(out, "Acrescentar") || !strings.Contains(out, "Acrescentado") {
		t.Fatalf("err = %v\nsaída:\n%s", err, out)
	}
	got, _ := os.ReadFile(p)
	want := "API_KEY=abc\nDB_USER=postgres\nDB_PASSWORD=postgres\nDB_NAME=shop\nDB_PORT=5432\n"
	if string(got) != want {
		t.Fatalf("conteúdo:\n%q", got)
	}
}

func TestOfferWithEnvFileAppendsThere(t *testing.T) {
	d := t.TempDir()
	p := write(t, d, "custom.env", "FOO=bar\n")
	nc := &NoConfigError{File: p}
	path, _, err := offer(t, "y\n", nc, d)
	if err != nil || path != p {
		t.Fatalf("path = %s, err = %v", path, err)
	}
	if _, err := os.Stat(filepath.Join(d, ".env")); err == nil {
		t.Error(".env não deveria ter sido criado")
	}
	if cfg, err := Load(d, p, "x"); err != nil || cfg.Name != "shop" {
		t.Fatalf("cfg = %+v, err = %v", cfg, err)
	}
}

func TestOfferDeclined(t *testing.T) {
	for _, answer := range []string{"n\n", "\n", "", "talvez\n"} {
		d := t.TempDir()
		_, _, err := offer(t, answer, &NoConfigError{Searched: Candidates}, d)
		if !errors.Is(err, ErrDeclined) {
			t.Errorf("resposta %q: erro = %v", answer, err)
		}
		if _, err := os.Stat(filepath.Join(d, ".env")); err == nil {
			t.Errorf("resposta %q: .env criado", answer)
		}
	}
}
