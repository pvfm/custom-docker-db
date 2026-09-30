// Package engine describes the database engines the tool can run. Everything
// specific to one database (image, environment, healthcheck, URL scheme)
// lives here so new engines are additive.
package engine

import (
	"fmt"
	"net/url"

	"github.com/pvfm/custom-docker-db/internal/envconfig"
)

// Engine is the description of a database engine running in a container.
type Engine struct {
	Name          string
	Image         string
	ContainerPort int
	// DataPath is where the engine stores data inside the container.
	DataPath   string
	VolumeName string
	// Shell is the interactive shell available in the image.
	Shell string
	// Env returns the container environment for a config.
	Env func(envconfig.Config) map[string]string
	// HealthTest is the compose healthcheck test. It may reference the
	// container environment with $$VAR.
	HealthTest []string
	// URL builds the connection string; the password is replaced by *** when
	// mask is set.
	URL func(cfg envconfig.Config, mask bool) string
}

// Postgres is the only engine supported today.
var Postgres = Engine{
	Name:          "postgres",
	Image:         "postgres:16-alpine",
	ContainerPort: 5432,
	DataPath:      "/var/lib/postgresql/data",
	VolumeName:    "pgdata",
	Shell:         "sh",
	Env: func(c envconfig.Config) map[string]string {
		return map[string]string{
			"POSTGRES_USER":     c.User,
			"POSTGRES_PASSWORD": c.Password,
			"POSTGRES_DB":       c.Name,
		}
	},
	HealthTest: []string{"CMD-SHELL", `pg_isready -U "$$POSTGRES_USER" -d "$$POSTGRES_DB"`},
	URL: func(c envconfig.Config, mask bool) string {
		u := url.URL{
			Scheme: "postgres",
			User:   url.UserPassword(c.User, c.Password),
			Host:   fmt.Sprintf("localhost:%d", c.Port),
			Path:   "/" + c.Name,
		}
		if mask {
			// url.UserPassword would escape the asterisks, so build the
			// masked form by hand.
			return fmt.Sprintf("postgres://%s:***@%s%s", url.QueryEscape(c.User), u.Host, u.Path)
		}
		return u.String()
	},
}
