# custom-docker-db

Spin up a local database in Docker, configured from the `.env` your project already has.

Clone a project, run one command, and get a database that matches its `DB_*` / `DATABASE_*` variables: same user, password, database name and port. No hand-written `docker-compose.yml`.

```console
$ cd my-project
$ custom-docker-db
Config de /home/me/my-project/.env: usuário=app banco=app_dev porta=5432
Subindo o Postgres (aguardando o healthcheck, até 60 s)...
Banco pronto: postgres://app:***@localhost:5432/app_dev
```

Today it supports **Postgres** (`postgres:16-alpine`). The name is generic on purpose: other databases are planned.

## Requirements

- Linux (x64 or arm64)
- Docker with the `docker compose` v2 plugin, and permission to talk to the Docker daemon

## Installation

### From source (works today)

Requires Go 1.27 or newer.

```bash
git clone git@github.com:pvfm/custom-docker-db.git
cd custom-docker-db
make build
install -m 755 bin/custom-docker-db ~/.local/bin/   # make sure ~/.local/bin is in your PATH
custom-docker-db --version
```

Or, without cloning:

```bash
go install github.com/pvfm/custom-docker-db/cmd/custom-docker-db@latest
```

### Prebuilt binaries

From the first published release (`v1.0.0`) on, these will also work:

```bash
curl -fsSL https://raw.githubusercontent.com/pvfm/custom-docker-db/main/install.sh | sh
npx @pvfm/custom-docker-db
```

`install.sh` verifies the SHA-256 of the download. Set `CDD_VERSION=v1.0.0` to pin a version or `CDD_INSTALL_DIR` to change the destination (default `~/.local/bin`).

## Quick start

1. Make sure the project has database variables in a `.env`:

   ```bash
   DB_USER=app
   DB_PASSWORD=secret
   DB_NAME=app_dev
   DB_PORT=5432
   ```

2. Start the database from the project directory:

   ```bash
   custom-docker-db
   ```

3. Point your app at `localhost:<DB_PORT>`, or open a shell inside the container:

   ```bash
   custom-docker-db exec
   psql -U "$POSTGRES_USER" -d "$POSTGRES_DB"
   ```

4. Stop it when you are done:

   ```bash
   custom-docker-db down
   ```

No database variables in the project yet? Run `custom-docker-db` anyway: it shows the defaults it expects and asks whether it may create them.

## Commands

| Command | What it does |
| --- | --- |
| `custom-docker-db` | Same as `up` |
| `custom-docker-db up` | Detect the config, generate the compose file, start the database, wait for the healthcheck and print the connection string |
| `custom-docker-db down` | Stop and remove the container. The volume is kept, so the data is back on the next `up` |
| `custom-docker-db exec` | Open an interactive shell (`sh`) inside the running container |
| `custom-docker-db --version` | Print the version |
| `custom-docker-db --help` | Show help (also `custom-docker-db <command> --help`) |

### Flags (`up`)

| Flag | What it does |
| --- | --- |
| `--env-file=<path>` | Use only this file. Skips automatic discovery |
| `--ephemeral` | Start without a volume: the data is deleted on `down` |

### Examples

```bash
custom-docker-db                               # auto-discover .env, start
custom-docker-db up --env-file=.env.local      # use a specific file
custom-docker-db up --ephemeral                # throwaway database for a quick test
custom-docker-db exec                          # shell inside the container
custom-docker-db down                          # remove the container, keep the data
```

## How the `.env` is read

Without `--env-file`, the CLI looks, in this order and only in the current directory, for `.env`, `.env.local` and `.env.development`. Only keys starting with `DB_` or `DATABASE_` count (case-insensitive).

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
| `DB_HOST` | ignored (the database is always on `localhost`) | none |
| `DB_URL` | used only when there are no separate variables | none |

- `DATABASE_*` is equivalent to `DB_*`. If the same field appears with both prefixes in one file, the CLI asks you to keep only one.
- `DB_URL` / `DATABASE_URL` must be a `postgres://` or `postgresql://` URL, e.g. `postgres://app:secret@localhost:5432/app_dev`.
- `${VAR}` interpolation is supported, e.g. `DATABASE_URL=postgres://${DB_USER}:${DB_PASSWORD}@localhost/app`.

The default written when a project has no database config:

```bash
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=directory-name
DB_PORT=5432
```

## What `up` does for you

- Generates a compose file at `~/.local/share/custom-docker-db/<dir>-<hash>/docker-compose.yml` (or under `$XDG_DATA_HOME`). Nothing is written inside your project, except the `.env` when you accept the default.
- Names the container `cdd-<dir>-<hash>`. The hash comes from the project path, so two projects with the same folder name never collide.
- Publishes the port on `127.0.0.1` only, so the database is not exposed to your network.
- Checks Docker first, and refuses to start when the port is already in use (it does nothing and tells you which port).
- Is idempotent: running `up` again on a healthy database just prints the connection string.

### When the config changes

Postgres only applies the user, password and database name when it first creates the volume. On `up`, if those changed since the last run, the CLI asks before deleting the volume and recreating the database; answering no leaves everything untouched. Changing only the port recreates the container and keeps your data. With `--ephemeral` there is no data to protect, so the container is recreated without asking.

## Troubleshooting

| Message | What to do |
| --- | --- |
| `a porta N já está em uso` | Free the port or change `DB_PORT` in your `.env` |
| `não foi possível falar com o daemon do Docker` | Start Docker, and check that your user can run `docker ps` (for example, is in the `docker` group) |
| `o plugin docker compose v2 não foi encontrado` | Install the Docker Compose plugin |
| `config de banco encontrada em mais de um arquivo` | Rename the extra `.env` files or pass `--env-file` |
| `o banco não está rodando` (on `exec`) | Run `custom-docker-db up` first |
| The app cannot log in after you changed the password | Run `custom-docker-db up` and answer `s` to recreate the volume |

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

### Releasing

Pushing a `v*` tag runs `.github/workflows/release.yml`: GoReleaser publishes the GitHub release (binaries and `checksums.txt`), then the npm packages are published. It needs an `NPM_TOKEN` repository secret and the `@pvfm` scope on npm.

```bash
git tag v1.0.0 && git push origin v1.0.0
```

## License

[MIT](LICENSE)
