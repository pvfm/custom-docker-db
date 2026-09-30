package cli

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pvfm/custom-docker-db/internal/compose"
	"github.com/pvfm/custom-docker-db/internal/docker"
	"github.com/pvfm/custom-docker-db/internal/engine"
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

// fakeDocker is a docker.Client that records calls instead of running docker.
type fakeDocker struct {
	preflightErr error
	exists       bool
	running      bool
	labels       map[string]string
	portBusy     bool
	upErr        error
	logs         string
	calls        []string
	downWipe     []bool
	execShell    string
}

func (f *fakeDocker) Preflight() error { f.calls = append(f.calls, "preflight"); return f.preflightErr }
func (f *fakeDocker) ContainerRunning(string) (bool, error) {
	f.calls = append(f.calls, "running?")
	return f.running, nil
}
func (f *fakeDocker) ContainerLabels(string) (map[string]string, bool, error) {
	f.calls = append(f.calls, "labels?")
	return f.labels, f.exists, nil
}
func (f *fakeDocker) PortFree(int) bool { f.calls = append(f.calls, "port?"); return !f.portBusy }
func (f *fakeDocker) Up(string, string, time.Duration) (string, error) {
	f.calls = append(f.calls, "up")
	return "saída do compose", f.upErr
}
func (f *fakeDocker) Down(_, _ string, wipe bool) (string, error) {
	f.calls = append(f.calls, "down")
	f.downWipe = append(f.downWipe, wipe)
	return "", nil
}
func (f *fakeDocker) Exec(_, shell string) error {
	f.calls = append(f.calls, "exec")
	f.execShell = shell
	return nil
}
func (f *fakeDocker) Logs(string, string, int) string {
	f.calls = append(f.calls, "logs")
	return f.logs
}

func (f *fakeDocker) called(name string) bool {
	for _, c := range f.calls {
		if c == name {
			return true
		}
	}
	return false
}

// setup isolates the test: temp project dir, temp data dir, fake docker.
func setup(t *testing.T, env string) (dir string, f *fakeDocker) {
	t.Helper()
	dir = t.TempDir()
	if env != "" {
		if err := os.WriteFile(filepath.Join(dir, ".env"), []byte(env), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	t.Chdir(dir)
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	f = &fakeDocker{}
	orig := newDocker
	newDocker = func() docker.Client { return f }
	t.Cleanup(func() { newDocker = orig })
	return dir, f
}

func run(t *testing.T, stdin string, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	cmd := NewRootCmd()
	var o, e bytes.Buffer
	cmd.SetOut(&o)
	cmd.SetErr(&e)
	cmd.SetIn(strings.NewReader(stdin))
	cmd.SetArgs(args)
	err = cmd.Execute()
	return o.String(), e.String(), err
}

func mustGetwd(t *testing.T) string {
	t.Helper()
	d, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return d
}

// existing marks the fake as holding a container started with cfg.
func existing(f *fakeDocker, cfg envconfig.Config, ephemeral, running bool) {
	f.exists, f.running = true, running
	f.labels = map[string]string{
		compose.ConfigLabel:    compose.ConfigHash(engine.Postgres, cfg),
		compose.EphemeralLabel: fmt.Sprint(ephemeral),
		compose.PortLabel:      fmt.Sprint(cfg.Port),
	}
}

func TestUpHappyPath(t *testing.T) {
	_, f := setup(t, "DB_USER=alice\nDB_PASSWORD=segredo\nDB_NAME=shop\nDB_PORT=5433\n")
	for _, args := range [][]string{{}, {"up"}} {
		f.calls = nil
		out, _, err := run(t, "", args...)
		if err != nil {
			t.Fatalf("args %v: %v", args, err)
		}
		if !strings.Contains(out, "postgres://alice:***@localhost:5433/shop") {
			t.Errorf("sem connection string mascarada:\n%s", out)
		}
		if strings.Contains(out, "segredo") {
			t.Errorf("senha vazou na saída:\n%s", out)
		}
		if !f.called("preflight") || !f.called("up") || f.called("down") {
			t.Errorf("chamadas = %v", f.calls)
		}
	}
	paths, _ := compose.Locate(mustGetwd(t))
	data, err := os.ReadFile(paths.File)
	if err != nil || !strings.Contains(string(data), "POSTGRES_USER: alice") || !strings.Contains(string(data), "pgdata") {
		t.Fatalf("compose: %v\n%s", err, data)
	}
	info, _ := os.Stat(paths.File)
	if info.Mode().Perm() != 0o600 {
		t.Errorf("permissão do compose = %v", info.Mode().Perm())
	}
}

func TestUpEphemeral(t *testing.T) {
	setup(t, "DB_USER=alice\n")
	out, _, err := run(t, "", "up", "--ephemeral")
	if err != nil || !strings.Contains(out, "efêmero") {
		t.Fatalf("err = %v\n%s", err, out)
	}
	paths, _ := compose.Locate(mustGetwd(t))
	data, _ := os.ReadFile(paths.File)
	if strings.Contains(string(data), "pgdata") {
		t.Errorf("compose efêmero com volume:\n%s", data)
	}
}

func TestUpPortBusyDoesNothing(t *testing.T) {
	_, f := setup(t, "DB_PORT=5433\n")
	f.portBusy = true
	_, _, err := run(t, "", "up")
	if err == nil || !strings.Contains(err.Error(), "5433") || !strings.Contains(err.Error(), "nada foi feito") {
		t.Fatalf("erro = %v", err)
	}
	if f.called("up") {
		t.Error("não deveria subir com a porta ocupada")
	}
	paths, _ := compose.Locate(mustGetwd(t))
	if _, err := os.Stat(paths.File); err == nil {
		t.Error("compose não deveria ter sido escrito")
	}
}

func TestUpRunningContainerSameConfigSkipsPortCheck(t *testing.T) {
	_, f := setup(t, "DB_USER=alice\n")
	cfg, _ := envconfig.Load(mustGetwd(t), "", envconfig.DefaultName(mustGetwd(t)))
	existing(f, cfg, false, true)
	f.portBusy = true // a porta é do nosso próprio container
	if _, _, err := run(t, "", "up"); err != nil {
		t.Fatal(err)
	}
	if f.called("port?") || f.called("down") || !f.called("up") {
		t.Errorf("chamadas = %v", f.calls)
	}
}

func TestUpPreflightFailure(t *testing.T) {
	_, f := setup(t, "DB_USER=alice\n")
	f.preflightErr = errors.New("docker não encontrado")
	_, _, err := run(t, "")
	if err == nil || !strings.Contains(err.Error(), "docker não encontrado") {
		t.Fatalf("erro = %v", err)
	}
	paths, _ := compose.Locate(mustGetwd(t))
	if _, err := os.Stat(paths.File); err == nil {
		t.Error("nada deve ser escrito antes do preflight passar")
	}
}

func TestUpFailureShowsLogs(t *testing.T) {
	_, f := setup(t, "DB_USER=alice\n")
	f.upErr, f.logs = errors.New("exit status 1"), "FATAL: linha do log"
	_, stderr, err := run(t, "")
	if err == nil || !strings.Contains(err.Error(), "não ficou saudável") {
		t.Fatalf("erro = %v", err)
	}
	if !strings.Contains(stderr, "FATAL: linha do log") || !strings.Contains(stderr, "saída do compose") {
		t.Errorf("stderr:\n%s", stderr)
	}
}

func TestUpOffersDefaultInEmptyDir(t *testing.T) {
	dir, _ := setup(t, "")
	out, _, err := run(t, "s\n", "up")
	if err != nil {
		t.Fatalf("erro = %v\n%s", err, out)
	}
	got, err := os.ReadFile(filepath.Join(dir, ".env"))
	if err != nil || !strings.Contains(string(got), "DB_PORT=5432") {
		t.Fatalf(".env = %q, err = %v", got, err)
	}
	if !strings.Contains(out, "Banco pronto") {
		t.Errorf("saída:\n%s", out)
	}
}

func TestUpDeclinedDefaultStops(t *testing.T) {
	dir, f := setup(t, "")
	_, _, err := run(t, "n\n", "up")
	if !errors.Is(err, envconfig.ErrDeclined) {
		t.Fatalf("erro = %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".env")); err == nil {
		t.Error(".env não deveria existir")
	}
	if f.called("preflight") {
		t.Error("não deve tocar no docker depois de recusar")
	}
}

// --- mudança de config -------------------------------------------------

func oldCfg() envconfig.Config {
	return envconfig.Config{User: "alice", Password: "antiga", Name: "shop", Port: 5432}
}

func TestUpPasswordChangedConfirmWipes(t *testing.T) {
	_, f := setup(t, "DB_USER=alice\nDB_PASSWORD=nova\nDB_NAME=shop\n")
	existing(f, oldCfg(), false, true)
	out, _, err := run(t, "s\n", "up")
	if err != nil {
		t.Fatalf("erro = %v\n%s", err, out)
	}
	if !strings.Contains(out, "Apagar o volume e recriar? [s/N]") {
		t.Errorf("sem a pergunta:\n%s", out)
	}
	if len(f.downWipe) != 1 || !f.downWipe[0] || !f.called("up") {
		t.Errorf("down(wipe) = %v, chamadas = %v", f.downWipe, f.calls)
	}
	if !f.called("port?") {
		t.Error("depois de recriar, a porta deve ser checada")
	}
}

func TestUpPasswordChangedDeclinedDoesNothing(t *testing.T) {
	_, f := setup(t, "DB_USER=alice\nDB_PASSWORD=nova\nDB_NAME=shop\n")
	existing(f, oldCfg(), false, true)
	for _, answer := range []string{"n\n", "\n", ""} {
		f.calls, f.downWipe = nil, nil
		_, _, err := run(t, answer, "up")
		if err == nil || !strings.Contains(err.Error(), "nada foi feito") {
			t.Fatalf("resposta %q: erro = %v", answer, err)
		}
		if f.called("down") || f.called("up") {
			t.Errorf("resposta %q: chamadas = %v", answer, f.calls)
		}
	}
	paths, _ := compose.Locate(mustGetwd(t))
	if _, err := os.Stat(paths.File); err == nil {
		t.Error("compose não deveria ter sido reescrito")
	}
}

func TestUpOnlyPortChangedKeepsVolume(t *testing.T) {
	_, f := setup(t, "DB_USER=alice\nDB_PASSWORD=antiga\nDB_NAME=shop\nDB_PORT=6000\n")
	existing(f, oldCfg(), false, true) // mesma senha, porta 5432 -> 6000
	out, _, err := run(t, "", "up")
	if err != nil {
		t.Fatalf("erro = %v\n%s", err, out)
	}
	if strings.Contains(out, "[s/N]") || !strings.Contains(out, "Porta alterada") {
		t.Errorf("mudar só a porta não deve perguntar:\n%s", out)
	}
	// o compose up recria o container sozinho; a nova porta precisa ser checada
	if f.called("down") || !f.called("port?") || !f.called("up") {
		t.Errorf("chamadas = %v", f.calls)
	}
}

func TestUpPortChangedToBusyPortDoesNothing(t *testing.T) {
	_, f := setup(t, "DB_USER=alice\nDB_PASSWORD=antiga\nDB_NAME=shop\nDB_PORT=6000\n")
	existing(f, oldCfg(), false, true)
	f.portBusy = true
	_, _, err := run(t, "", "up")
	if err == nil || !strings.Contains(err.Error(), "6000") || f.called("up") || f.called("down") {
		t.Fatalf("erro = %v, chamadas = %v", err, f.calls)
	}
}

func TestUpOldEphemeralChangedWipesWithoutAsking(t *testing.T) {
	_, f := setup(t, "DB_USER=alice\nDB_PASSWORD=nova\nDB_NAME=shop\n")
	existing(f, oldCfg(), true, true)
	out, _, err := run(t, "", "up", "--ephemeral")
	if err != nil {
		t.Fatalf("erro = %v\n%s", err, out)
	}
	if strings.Contains(out, "[s/N]") || len(f.downWipe) != 1 || !f.downWipe[0] {
		t.Errorf("down(wipe) = %v\n%s", f.downWipe, out)
	}
}

func TestUpPersistentToEphemeralKeepsNamedVolume(t *testing.T) {
	_, f := setup(t, "DB_USER=alice\nDB_PASSWORD=antiga\nDB_NAME=shop\n")
	existing(f, oldCfg(), false, true)
	out, _, err := run(t, "", "up", "--ephemeral")
	if err != nil {
		t.Fatalf("erro = %v\n%s", err, out)
	}
	if strings.Contains(out, "[s/N]") || len(f.downWipe) != 1 || f.downWipe[0] {
		t.Errorf("down(wipe) = %v\n%s", f.downWipe, out)
	}
}

func TestPlanChange(t *testing.T) {
	h := "abc"
	lab := func(hash, eph string) map[string]string {
		return map[string]string{compose.ConfigLabel: hash, compose.EphemeralLabel: eph}
	}
	cases := []struct {
		name      string
		labels    map[string]string
		ephemeral bool
		want      change
	}{
		{"igual", lab(h, "false"), false, changeNone},
		{"igual efêmero", lab(h, "true"), true, changeNone},
		{"senha mudou", lab("outro", "false"), false, changeConfirm},
		{"senha mudou, antigo efêmero", lab("outro", "true"), false, changeWipe},
		{"senha mudou, novo efêmero", lab("outro", "false"), true, changeRecreate},
		{"só o modo mudou: persistente -> efêmero", lab(h, "false"), true, changeRecreate},
		{"só o modo mudou: efêmero -> persistente", lab(h, "true"), false, changeWipe},
		{"sem labels", map[string]string{}, false, changeConfirm},
	}
	for _, c := range cases {
		if got := planChange(c.labels, h, c.ephemeral); got != c.want {
			t.Errorf("%s: got %v, esperado %v", c.name, got, c.want)
		}
	}
}

// --- down e exec -------------------------------------------------------

func TestDownNoContainer(t *testing.T) {
	_, f := setup(t, "")
	out, _, err := run(t, "", "down")
	if err != nil || !strings.Contains(out, "nada a fazer") || f.called("down") {
		t.Fatalf("err = %v, chamadas = %v\n%s", err, f.calls, out)
	}
}

func TestDownPersistentKeepsVolume(t *testing.T) {
	_, f := setup(t, "")
	existing(f, oldCfg(), false, true)
	out, _, err := run(t, "", "down")
	if err != nil || len(f.downWipe) != 1 || f.downWipe[0] || !strings.Contains(out, "volume foi mantido") {
		t.Fatalf("err = %v, wipe = %v\n%s", err, f.downWipe, out)
	}
}

func TestDownEphemeralWipesData(t *testing.T) {
	_, f := setup(t, "")
	existing(f, oldCfg(), true, true)
	out, _, err := run(t, "", "down")
	if err != nil || len(f.downWipe) != 1 || !f.downWipe[0] || !strings.Contains(out, "efêmeros") {
		t.Fatalf("err = %v, wipe = %v\n%s", err, f.downWipe, out)
	}
}

func TestDownStoppedContainerStillRemoved(t *testing.T) {
	_, f := setup(t, "")
	existing(f, oldCfg(), false, false)
	if _, _, err := run(t, "", "down"); err != nil || len(f.downWipe) != 1 {
		t.Fatalf("err = %v, wipe = %v", err, f.downWipe)
	}
}

func TestDownPreflightFailure(t *testing.T) {
	_, f := setup(t, "")
	f.preflightErr = errors.New("daemon parado")
	if _, _, err := run(t, "", "down"); err == nil || !strings.Contains(err.Error(), "daemon parado") {
		t.Fatalf("erro = %v", err)
	}
}

func TestExecOpensShell(t *testing.T) {
	_, f := setup(t, "")
	existing(f, oldCfg(), false, true)
	if _, _, err := run(t, "", "exec"); err != nil {
		t.Fatal(err)
	}
	if !f.called("exec") || f.execShell != "sh" {
		t.Errorf("chamadas = %v, shell = %q", f.calls, f.execShell)
	}
}

func TestExecRequiresRunningDatabase(t *testing.T) {
	_, f := setup(t, "")
	for name, prep := range map[string]func(){
		"sem container": func() {},
		"parado":        func() { existing(f, oldCfg(), false, false) },
	} {
		prep()
		_, _, err := run(t, "", "exec")
		if err == nil || !strings.Contains(err.Error(), "não está rodando") {
			t.Errorf("%s: erro = %v", name, err)
		}
		if f.called("exec") {
			t.Errorf("%s: não deveria abrir o shell", name)
		}
	}
}
