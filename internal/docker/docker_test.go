package docker

import (
	"net"
	"testing"
)

func TestPortFree(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	if (CLI{}).PortFree(port) {
		t.Error("porta ocupada reportada como livre")
	}
	l.Close()
	if !(CLI{}).PortFree(port) {
		t.Error("porta liberada reportada como ocupada")
	}
}

func TestIsNoSuchObject(t *testing.T) {
	for _, out := range []string{
		"Error: No such object: cdd-x",
		"error: no such object: cdd-x",
		"Error response from daemon: No such container: cdd-x",
	} {
		if !isNoSuchObject(out) {
			t.Errorf("não reconheceu %q", out)
		}
	}
	if isNoSuchObject("Cannot connect to the Docker daemon") {
		t.Error("erro do daemon tratado como container inexistente")
	}
}
