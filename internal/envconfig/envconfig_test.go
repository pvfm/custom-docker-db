package envconfig

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func write(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestDiscovery(t *testing.T) {
	t.Run("nenhum arquivo", func(t *testing.T) {
		_, err := Load(t.TempDir(), "", "app")
		var e *NoConfigError
		if !errors.As(err, &e) || e.File != "" || len(e.Searched) != 3 {
			t.Fatalf("erro = %v", err)
		}
	})
	t.Run("arquivo sem chaves de banco", func(t *testing.T) {
		d := t.TempDir()
		write(t, d, ".env", "PORT=3000\nAPI_KEY=x\n")
		_, err := Load(d, "", "app")
		var e *NoConfigError
		if !errors.As(err, &e) {
			t.Fatalf("erro = %v", err)
		}
	})
	t.Run("um arquivo", func(t *testing.T) {
		d := t.TempDir()
		write(t, d, ".env", "PORT=3000\n")
		write(t, d, ".env.local", "DB_USER=alice\n")
		cfg, err := Load(d, "", "app")
		if err != nil {
			t.Fatal(err)
		}
		if cfg.User != "alice" || filepath.Base(cfg.Source) != ".env.local" {
			t.Fatalf("cfg = %+v", cfg)
		}
	})
	t.Run("vários arquivos", func(t *testing.T) {
		d := t.TempDir()
		write(t, d, ".env", "DB_USER=a\n")
		write(t, d, ".env.development", "DATABASE_USER=b\n")
		_, err := Load(d, "", "app")
		var e *MultipleFilesError
		if !errors.As(err, &e) || strings.Join(e.Files, ",") != ".env,.env.development" {
			t.Fatalf("erro = %v", err)
		}
	})
	t.Run("ignora .env.example", func(t *testing.T) {
		d := t.TempDir()
		write(t, d, ".env.example", "DB_USER=a\n")
		_, err := Load(d, "", "app")
		var e *NoConfigError
		if !errors.As(err, &e) {
			t.Fatalf("erro = %v", err)
		}
	})
}

func TestEnvFileFlag(t *testing.T) {
	d := t.TempDir()
	write(t, d, ".env", "DB_USER=a\n")
	custom := write(t, d, "custom.env", "DB_USER=custom\nDB_PORT=6000\n")

	cfg, err := Load(d, custom, "app")
	if err != nil || cfg.User != "custom" || cfg.Port != 6000 {
		t.Fatalf("cfg = %+v, err = %v", cfg, err)
	}

	if _, err := Load(d, filepath.Join(d, "nope.env"), "app"); err == nil || !strings.Contains(err.Error(), "não encontrado") {
		t.Fatalf("erro = %v", err)
	}

	empty := write(t, d, "empty.env", "FOO=bar\n")
	var e *NoConfigError
	if _, err := Load(d, empty, "app"); !errors.As(err, &e) || e.File != empty {
		t.Fatalf("erro = %v", err)
	}
}

func TestSeparateVariables(t *testing.T) {
	d := t.TempDir()
	write(t, d, ".env", `# comentário
export db_user = "alice"
DB_PASSWORD='s3cret'
DB_NAME=shop
DB_PORT=5433
DB_HOST=db.internal
`)
	cfg, err := Load(d, "", "app")
	if err != nil {
		t.Fatal(err)
	}
	want := Config{User: "alice", Password: "s3cret", Name: "shop", Port: 5433, Source: filepath.Join(d, ".env")}
	if cfg != want {
		t.Fatalf("cfg = %+v, esperado %+v", cfg, want)
	}
}

func TestAliasesAndDatabasePrefix(t *testing.T) {
	d := t.TempDir()
	write(t, d, ".env", "DATABASE_USERNAME=bob\nDATABASE_PASS=pw\nDATABASE_DATABASE=orders\n")
	cfg, err := Load(d, "", "app")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.User != "bob" || cfg.Password != "pw" || cfg.Name != "orders" || cfg.Port != DefaultPort {
		t.Fatalf("cfg = %+v", cfg)
	}
}

func TestDefaults(t *testing.T) {
	d := t.TempDir()
	write(t, d, ".env", "DB_HOST=localhost\n")
	cfg, err := Load(d, "", "meuapp")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.User != "postgres" || cfg.Password != "postgres" || cfg.Name != "meuapp" || cfg.Port != 5432 {
		t.Fatalf("cfg = %+v", cfg)
	}
}

func TestURLFallback(t *testing.T) {
	d := t.TempDir()
	write(t, d, ".env", "DATABASE_URL=postgres://u:p%40ss@db:6543/mydb?sslmode=disable\n")
	cfg, err := Load(d, "", "app")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.User != "u" || cfg.Password != "p@ss" || cfg.Name != "mydb" || cfg.Port != 6543 {
		t.Fatalf("cfg = %+v", cfg)
	}
}

func TestURLPartialAndScheme(t *testing.T) {
	d := t.TempDir()
	write(t, d, ".env", "DB_URL=postgresql://localhost\n")
	cfg, err := Load(d, "", "app")
	if err != nil || cfg.User != "postgres" || cfg.Name != "app" || cfg.Port != 5432 {
		t.Fatalf("cfg = %+v, err = %v", cfg, err)
	}

	write(t, d, ".env", "DB_URL=mysql://root@localhost/x\n")
	if _, err := Load(d, "", "app"); err == nil || !strings.Contains(err.Error(), "não suportado") {
		t.Fatalf("erro = %v", err)
	}
}

func TestSeparateVariablesWinOverURL(t *testing.T) {
	d := t.TempDir()
	write(t, d, ".env", "DB_URL=postgres://u:p@h:1111/urldb\nDB_USER=alice\n")
	cfg, err := Load(d, "", "app")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.User != "alice" || cfg.Name != "app" || cfg.Port != 5432 {
		t.Fatalf("cfg = %+v", cfg)
	}
}

func TestInterpolation(t *testing.T) {
	d := t.TempDir()
	// BASE_USER não é variável de banco: só existe para ser interpolada.
	write(t, d, ".env", "BASE_USER=carol\nDATABASE_URL=postgres://${BASE_USER}:x@localhost:7000/app\n")
	cfg, err := Load(d, "", "app")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.User != "carol" || cfg.Port != 7000 {
		t.Fatalf("cfg = %+v", cfg)
	}
}

func TestPrefixConflict(t *testing.T) {
	d := t.TempDir()
	write(t, d, ".env", "DB_USER=a\nDATABASE_USER=b\nDB_PORT=5432\n")
	_, err := Load(d, "", "app")
	var e *PrefixConflictError
	if !errors.As(err, &e) || strings.Join(e.Keys, ",") != "DATABASE_USER,DB_USER" {
		t.Fatalf("erro = %v", err)
	}
}

func TestInvalidPort(t *testing.T) {
	d := t.TempDir()
	for _, p := range []string{"abc", "0", "70000"} {
		write(t, d, ".env", "DB_PORT="+p+"\n")
		if _, err := Load(d, "", "app"); err == nil || !strings.Contains(err.Error(), "porta inválida") {
			t.Errorf("porta %q: erro = %v", p, err)
		}
	}
}
