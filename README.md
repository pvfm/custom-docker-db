# custom-docker-db

A CLI that spins up a local database in Docker, already configured with your project's credentials. It reads the `.env` in the current directory, extracts the `DB_*` / `DATABASE_*` variables, then generates and runs a `docker compose` with that configuration.

Today it supports **Postgres** only. The name is generic on purpose: the plan is to support other databases in the future.

> **Status: feature-complete for v1, first release pending.** `up`, `down` and `exec` work; the `v1.0.0` release has not been published yet. See [progress](#progress).

## Usage

```bash
custom-docker-db                          # = up, discovers the .env in ./
custom-docker-db up --env-file=.env.local # use only this file
custom-docker-db up --ephemeral           # no volume: data is lost on down
custom-docker-db down                     # stop and remove the container, keep the volume
custom-docker-db exec                     # open a shell (sh) inside the container
```

| Command / flag | What it does |
| --- | --- |
| `up` (default) | Detects the config, generates the compose file, starts Postgres, waits for the healthcheck and prints the connection string |
| `down` | Stops and removes the container; the volume is kept (in `--ephemeral` mode the data is deleted) |
| `exec` | Interactive shell (`sh`) inside the running container |
| `--env-file=<path>` | Use only this file, no automatic discovery |
| `--ephemeral` | Start without a volume |

The image is always `postgres:16-alpine`. The port is published on `127.0.0.1` only, and the generated compose file lives in `~/.local/share/custom-docker-db/<dir>-<hash>/` (or `$XDG_DATA_HOME`). Requires Docker with the `docker compose` v2 plugin. Supported: Linux (x64 and arm64).

### When the config changes

Postgres only applies the user, password and database name when it first creates the volume. On `up`, if those changed since the last run, the CLI asks before deleting the volume and recreating the database; answering no leaves everything untouched. Changing only the port recreates the container and keeps your data.

## How the `.env` is read

Without `--env-file`, the CLI looks, in this order and only in the current directory, for: `.env`, `.env.local`, `.env.development`. Only keys starting with `DB_` or `DATABASE_` count (case-insensitive).

| Files with database config | Behavior |
| --- | --- |
| 1 | Uses that file |
| More than 1 | Lists the files and exits, asking you to rename them (or use `--env-file`) |
| None | Shows the default and asks whether it may create it (or append it to the existing `.env`) |

| Key | Becomes | Default when missing |
| --- | --- | --- |
| `DB_USER` / `DB_USERNAME` | `POSTGRES_USER` | `postgres` |
| `DB_PASSWORD` / `DB_PASS` | `POSTGRES_PASSWORD` | `postgres` |
| `DB_NAME` / `DB_DATABASE` | `POSTGRES_DB` | directory name |
| `DB_PORT` | host port | `5432` |
| `DB_HOST` | ignored (the database is always on `localhost`) | — |
| `DB_URL` | used only when there are no separate variables | — |

`DATABASE_*` is equivalent to `DB_*`. If the same field appears with both prefixes in the same file, the CLI asks you to keep only one. `${VAR}` interpolation is supported. The default written when there is no config:

```bash
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=directory-name
DB_PORT=5432
```

## Progress

| Milestone | Deliverable | Status |
| --- | --- | --- |
| M1 | Project base: Go module, `up`/`down`/`exec` commands, Makefile, CI | Done |
| M2 | `.env` discovery and parsing | Done |
| M3 | Interactive offer of the default config | Done |
| M4 | Generated compose file and `up` (Docker, port, healthcheck, `--ephemeral`) | Done |
| M5 | `down`, `exec` and config change with a volume prompt | Done |
| M6 | Distribution: GoReleaser, `install.sh`, npm packages `@pvfm/*` | Done |
| M7 | `v1.0.0` release | In progress (ready to tag) |

## Development

Requires Go 1.27 or newer.

```bash
make build   # builds bin/custom-docker-db
make test    # go test ./...
make lint    # go vet + gofmt
```

Layout:

- `cmd/custom-docker-db/`: binary `main`
- `internal/cli/`: commands (`cobra`)
- `internal/envconfig/`: `.env` discovery and parsing, default config offer
- `internal/engine/`: database-specific definition (image, env, healthcheck, URL)
- `internal/compose/`: compose file rendering and location
- `internal/docker/`: docker CLI wrapper
- `internal/prompt/`: yes/no confirmation
- `npm/`, `scripts/`, `install.sh`, `.goreleaser.yaml`: distribution

## Installation

Not available yet: the release pipeline is ready, but the first release (`v1.0.0`) has not been published. Once it is:

```bash
# Linux x64 / arm64, installs to ~/.local/bin
curl -fsSL https://raw.githubusercontent.com/pvfm/custom-docker-db/main/install.sh | sh

# or with npm
npx @pvfm/custom-docker-db
```

`install.sh` verifies the SHA-256 of the download. Set `CDD_VERSION=v1.0.0` to pin a version or `CDD_INSTALL_DIR` to change the destination.

## Releasing

Pushing a `v*` tag runs `.github/workflows/release.yml`: GoReleaser publishes the GitHub release (binaries + `checksums.txt`), then the npm packages are published. This needs an `NPM_TOKEN` repository secret and the `@pvfm` scope on npm.

```bash
git tag v1.0.0 && git push origin v1.0.0
```

## License

[MIT](LICENSE)
