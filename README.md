# custom-docker-db

A CLI that spins up a local database in Docker, already configured with your project's credentials. It reads the `.env` in the current directory, extracts the `DB_*` / `DATABASE_*` variables, then generates and runs a `docker compose` with that configuration.

Today it supports **Postgres** only. The name is generic on purpose: the plan is to support other databases in the future.

> **Status: under development.** There is no published release yet, and `up` does not start the container yet. See [progress](#progress).

## How it will work

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
| `down` | Stops and removes the container; the volume is kept |
| `exec` | Interactive shell inside the container |
| `--env-file=<path>` | Use only this file, no automatic discovery |
| `--ephemeral` | Start without a volume |

The image is always `postgres:16-alpine`. Requires Docker with the `docker compose` v2 plugin. Supported: Linux (x64 and arm64).

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
| M4 | Generated compose file and `up` (Docker, port, healthcheck, `--ephemeral`) | Not started |
| M5 | `down`, `exec` and config change with a volume prompt | Not started |
| M6 | Distribution: GoReleaser, `install.sh`, npm packages `@pvfm/*` | Not started |
| M7 | `v1.0.0` release | Not started |

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

## Installation

Not available yet. After v1 it will be available via:

- `curl -fsSL https://raw.githubusercontent.com/pvfm/custom-docker-db/main/install.sh | sh`
- `npx @pvfm/custom-docker-db`

## License

[MIT](LICENSE)
