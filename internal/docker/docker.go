// Package docker wraps the docker CLI calls the tool needs.
package docker

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Client is the docker surface used by the commands (fakeable in tests).
type Client interface {
	// Preflight checks the docker binary, the compose v2 plugin and the daemon.
	Preflight() error
	ContainerRunning(name string) (bool, error)
	// ContainerLabels returns the labels of a container; exists is false when
	// there is no such container.
	ContainerLabels(name string) (labels map[string]string, exists bool, err error)
	PortFree(port int) bool
	// Up runs `compose up -d --wait` and returns the combined output.
	Up(composeFile, project string, timeout time.Duration) (string, error)
	// Down removes the container; with wipeVolumes it also removes the
	// volumes (`down -v`). composeFile may be missing on disk.
	Down(composeFile, project string, wipeVolumes bool) (string, error)
	// Exec opens an interactive shell in a running container, attached to
	// the terminal.
	Exec(container, shell string) error
	// Logs returns the last lines of the compose logs.
	Logs(composeFile, project string, lines int) string
}

// CLI is the real Client, backed by the docker binary.
type CLI struct{}

// New returns the real docker client.
func New() Client { return CLI{} }

func run(args ...string) (string, error) {
	cmd := exec.Command("docker", args...)
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	err := cmd.Run()
	return strings.TrimSpace(out.String()), err
}

func (CLI) Preflight() error {
	if _, err := exec.LookPath("docker"); err != nil {
		return errors.New("docker não encontrado; instale em https://docs.docker.com/engine/install/")
	}
	if _, err := run("compose", "version"); err != nil {
		return errors.New("o plugin `docker compose` v2 não foi encontrado; veja https://docs.docker.com/compose/install/")
	}
	if out, err := run("info", "--format", "{{.ServerVersion}}"); err != nil {
		return fmt.Errorf("não foi possível falar com o daemon do Docker (ele está rodando? seu usuário está no grupo docker?): %s", lastLine(out))
	}
	return nil
}

func (CLI) ContainerRunning(name string) (bool, error) {
	out, err := run("inspect", "--format", "{{.State.Running}}", name)
	if err != nil {
		// Unknown container is the normal "not running" case.
		if isNoSuchObject(out) {
			return false, nil
		}
		return false, fmt.Errorf("docker inspect %s: %s", name, lastLine(out))
	}
	return out == "true", nil
}

func (CLI) ContainerLabels(name string) (map[string]string, bool, error) {
	out, err := run("inspect", "--format", "{{json .Config.Labels}}", name)
	if err != nil {
		if isNoSuchObject(out) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("docker inspect %s: %s", name, lastLine(out))
	}
	labels := map[string]string{}
	if err := json.Unmarshal([]byte(out), &labels); err != nil {
		// A container without labels prints "null".
		labels = map[string]string{}
	}
	return labels, true, nil
}

func (CLI) PortFree(port int) bool {
	l, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return false
	}
	l.Close()
	return true
}

func (CLI) Up(composeFile, project string, timeout time.Duration) (string, error) {
	return run("compose", "-p", project, "-f", composeFile,
		"up", "-d", "--wait", "--wait-timeout", fmt.Sprint(int(timeout.Seconds())))
}

func (CLI) Down(composeFile, project string, wipeVolumes bool) (string, error) {
	args := []string{"compose", "-p", project}
	if _, err := os.Stat(composeFile); err == nil {
		args = append(args, "-f", composeFile)
	}
	args = append(args, "down")
	if wipeVolumes {
		args = append(args, "-v")
	}
	return run(args...)
}

func (CLI) Exec(container, shell string) error {
	args := []string{"exec", "-i"}
	if fi, err := os.Stdin.Stat(); err == nil && fi.Mode()&os.ModeCharDevice != 0 {
		args = append(args, "-t")
	}
	cmd := exec.Command("docker", append(args, container, shell)...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	return cmd.Run()
}

func (CLI) Logs(composeFile, project string, lines int) string {
	out, _ := run("compose", "-p", project, "-f", composeFile, "logs", "--no-color", "--tail", fmt.Sprint(lines))
	return out
}

// isNoSuchObject recognizes the "unknown container" answers of docker inspect
// across versions ("Error: No such object: x", "error: no such object: x").
func isNoSuchObject(out string) bool {
	return strings.Contains(strings.ToLower(out), "no such")
}

func lastLine(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	return lines[len(lines)-1]
}
