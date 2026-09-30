package engine

import (
	"strings"
	"testing"

	"github.com/pvfm/custom-docker-db/internal/envconfig"
)

func TestPostgresURL(t *testing.T) {
	c := envconfig.Config{User: "u", Password: "p@ss", Name: "db", Port: 5433}
	if got := Postgres.URL(c, true); got != "postgres://u:***@localhost:5433/db" {
		t.Errorf("mascarada = %q", got)
	}
	if got := Postgres.URL(c, false); got != "postgres://u:p%40ss@localhost:5433/db" {
		t.Errorf("completa = %q", got)
	}
	if strings.Contains(Postgres.URL(c, true), "p@ss") {
		t.Error("senha vazou na URL mascarada")
	}
}
