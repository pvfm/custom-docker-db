package compose

import (
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/pvfm/custom-docker-db/internal/engine"
	"github.com/pvfm/custom-docker-db/internal/envconfig"
)

func spec(ephemeral bool) Spec {
	return Spec{
		Engine:    engine.Postgres,
		Config:    envconfig.Config{User: "alice", Password: "p$ss", Name: "shop", Port: 5433},
		Project:   "cdd-shop-1234abcd",
		Ephemeral: ephemeral,
	}
}

type parsed struct {
	Services map[string]struct {
		Image         string            `yaml:"image"`
		ContainerName string            `yaml:"container_name"`
		Environment   map[string]string `yaml:"environment"`
		Ports         []string          `yaml:"ports"`
		Volumes       []string          `yaml:"volumes"`
		Labels        map[string]string `yaml:"labels"`
		Healthcheck   struct {
			Test    []string `yaml:"test"`
			Retries int      `yaml:"retries"`
		} `yaml:"healthcheck"`
	} `yaml:"services"`
	Volumes map[string]any `yaml:"volumes"`
}

func render(t *testing.T, s Spec) parsed {
	t.Helper()
	data, err := Render(s)
	if err != nil {
		t.Fatal(err)
	}
	var p parsed
	if err := yaml.Unmarshal(data, &p); err != nil {
		t.Fatalf("yaml inválido: %v\n%s", err, data)
	}
	return p
}

func TestRenderPersistent(t *testing.T) {
	p := render(t, spec(false))
	svc := p.Services["postgres"]
	if svc.Image != "postgres:16-alpine" || svc.ContainerName != "cdd-shop-1234abcd" {
		t.Fatalf("svc = %+v", svc)
	}
	if svc.Environment["POSTGRES_USER"] != "alice" || svc.Environment["POSTGRES_DB"] != "shop" {
		t.Errorf("env = %v", svc.Environment)
	}
	if len(svc.Ports) != 1 || svc.Ports[0] != "127.0.0.1:5433:5432" {
		t.Errorf("ports = %v", svc.Ports)
	}
	if len(svc.Volumes) != 1 || svc.Volumes[0] != "pgdata:/var/lib/postgresql/data" {
		t.Errorf("volumes = %v", svc.Volumes)
	}
	if _, ok := p.Volumes["pgdata"]; !ok {
		t.Errorf("volume pgdata não declarado: %v", p.Volumes)
	}
	if svc.Healthcheck.Retries != 30 || !strings.Contains(svc.Healthcheck.Test[1], "pg_isready") {
		t.Errorf("healthcheck = %+v", svc.Healthcheck)
	}
	if svc.Labels[ConfigLabel] == "" || svc.Labels[EphemeralLabel] != "false" || svc.Labels[PortLabel] != "5433" {
		t.Errorf("labels = %v", svc.Labels)
	}
}

func TestRenderEphemeralHasNoVolume(t *testing.T) {
	p := render(t, spec(true))
	if len(p.Services["postgres"].Volumes) != 0 || len(p.Volumes) != 0 {
		t.Fatalf("efêmero com volume: %+v", p)
	}
	if p.Services["postgres"].Labels[EphemeralLabel] != "true" {
		t.Errorf("labels = %v", p.Services["postgres"].Labels)
	}
}

func TestRenderEscapesDollar(t *testing.T) {
	p := render(t, spec(false))
	if got := p.Services["postgres"].Environment["POSTGRES_PASSWORD"]; got != "p$$ss" {
		t.Fatalf("senha = %q (esperado p$$ss, para o compose não interpolar)", got)
	}
}

func TestRenderQuotesNumericPassword(t *testing.T) {
	s := spec(false)
	s.Config.Password = "12345"
	p := render(t, s)
	if got := p.Services["postgres"].Environment["POSTGRES_PASSWORD"]; got != "12345" {
		t.Fatalf("senha = %q", got)
	}
}

func TestConfigHash(t *testing.T) {
	a := spec(false).Config
	b := a
	b.Password = "outra"
	c := a
	c.Port = 9999
	if ConfigHash(engine.Postgres, a) != ConfigHash(engine.Postgres, c) {
		t.Error("só a porta mudou: o hash deveria continuar igual")
	}
	if ConfigHash(engine.Postgres, a) == ConfigHash(engine.Postgres, b) {
		t.Error("senha diferente deveria mudar o hash")
	}
}

func TestLocate(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_DATA_HOME", xdg)

	p1, err := Locate("/work/MyApp")
	if err != nil {
		t.Fatal(err)
	}
	p2, _ := Locate("/other/MyApp")
	if p1.Project == p2.Project {
		t.Error("mesmo nome de pasta em caminhos diferentes deve gerar projetos diferentes")
	}
	if !strings.HasPrefix(p1.Project, "cdd-myapp-") || p1.Project != strings.ToLower(p1.Project) {
		t.Errorf("project = %q", p1.Project)
	}
	if want := filepath.Join(xdg, "custom-docker-db", strings.TrimPrefix(p1.Project, "cdd-"), "docker-compose.yml"); p1.File != want {
		t.Errorf("file = %q, esperado %q", p1.File, want)
	}
	again, _ := Locate("/work/MyApp")
	if again != p1 {
		t.Error("Locate deve ser estável")
	}
}

func TestLocateDefaultsToLocalShare(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("HOME", "/home/fulano")
	p, err := Locate("/work/app")
	if err != nil || !strings.HasPrefix(p.Dir, "/home/fulano/.local/share/custom-docker-db/") {
		t.Fatalf("dir = %q, err = %v", p.Dir, err)
	}
}
