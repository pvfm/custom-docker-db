// Package compose renders the docker compose file and decides where it lives.
package compose

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/pvfm/custom-docker-db/internal/engine"
	"github.com/pvfm/custom-docker-db/internal/envconfig"
)

// Labels stored on the container so a later run can detect changes.
const (
	// ConfigLabel holds the hash of the settings the database only reads when
	// it initializes its data directory.
	ConfigLabel = "com.custom-docker-db.config-hash"
	// PortLabel is the host port the container publishes.
	PortLabel = "com.custom-docker-db.port"
	// EphemeralLabel is "true" when the container was started without a volume.
	EphemeralLabel = "com.custom-docker-db.ephemeral"
)

// Spec is everything needed to render a compose file.
type Spec struct {
	Engine    engine.Engine
	Config    envconfig.Config
	Project   string
	Ephemeral bool
}

// Paths says where the compose file of a project lives.
type Paths struct {
	Dir  string
	File string
	// Project is both the compose project name and the container name.
	Project string
}

type file struct {
	Services map[string]service  `yaml:"services"`
	Volumes  map[string]struct{} `yaml:"volumes,omitempty"`
}

type service struct {
	Image         string            `yaml:"image"`
	ContainerName string            `yaml:"container_name"`
	Environment   map[string]string `yaml:"environment"`
	Ports         []string          `yaml:"ports"`
	Volumes       []string          `yaml:"volumes,omitempty"`
	Labels        map[string]string `yaml:"labels"`
	Healthcheck   healthcheck       `yaml:"healthcheck"`
}

type healthcheck struct {
	Test     []string `yaml:"test"`
	Interval string   `yaml:"interval"`
	Timeout  string   `yaml:"timeout"`
	Retries  int      `yaml:"retries"`
}

// Locate computes the compose location for a project directory:
// $XDG_DATA_HOME (or ~/.local/share)/custom-docker-db/<dir>-<hash>/docker-compose.yml.
func Locate(projectDir string) (Paths, error) {
	abs, err := filepath.Abs(projectDir)
	if err != nil {
		return Paths{}, err
	}
	sum := sha256.Sum256([]byte(abs))
	slug := strings.ToLower(envconfig.DefaultName(abs)) + "-" + hex.EncodeToString(sum[:])[:8]

	base := os.Getenv("XDG_DATA_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return Paths{}, err
		}
		base = filepath.Join(home, ".local", "share")
	}
	dir := filepath.Join(base, "custom-docker-db", slug)
	return Paths{Dir: dir, File: filepath.Join(dir, "docker-compose.yml"), Project: "cdd-" + slug}, nil
}

// Write renders the spec and stores it (0600: it contains the password).
func Write(p Paths, s Spec) error {
	data, err := Render(s)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(p.Dir, 0o700); err != nil {
		return err
	}
	return os.WriteFile(p.File, data, 0o600)
}

// Render produces the compose YAML.
func Render(s Spec) ([]byte, error) {
	e := s.Engine
	env := map[string]string{}
	for k, v := range e.Env(s.Config) {
		env[k] = escapeDollar(v)
	}
	svc := service{
		Image:         e.Image,
		ContainerName: s.Project,
		Environment:   env,
		// Bound to localhost only: the default credentials are weak.
		Ports: []string{fmt.Sprintf("127.0.0.1:%d:%d", s.Config.Port, e.ContainerPort)},
		Labels: map[string]string{
			ConfigLabel:    ConfigHash(s.Engine, s.Config),
			EphemeralLabel: fmt.Sprint(s.Ephemeral),
			PortLabel:      fmt.Sprint(s.Config.Port),
		},
		Healthcheck: healthcheck{
			Test:     e.HealthTest,
			Interval: "2s",
			Timeout:  "5s",
			Retries:  30,
		},
	}
	f := file{Services: map[string]service{e.Name: svc}}
	if !s.Ephemeral {
		svc.Volumes = []string{e.VolumeName + ":" + e.DataPath}
		f.Services[e.Name] = svc
		f.Volumes = map[string]struct{}{e.VolumeName: {}}
	}
	return yaml.Marshal(f)
}

// ConfigHash identifies the settings the database only applies on first
// initialization (user, password, database name). The port is left out: a new
// port only recreates the container and keeps the data.
func ConfigHash(e engine.Engine, c envconfig.Config) string {
	env := e.Env(c)
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	h := sha256.New()
	for _, k := range keys {
		fmt.Fprint(h, k, "=", env[k], "\x00")
	}
	return hex.EncodeToString(h.Sum(nil))[:16]
}

// escapeDollar stops compose from interpolating $ in user-provided values.
func escapeDollar(s string) string { return strings.ReplaceAll(s, "$", "$$") }
