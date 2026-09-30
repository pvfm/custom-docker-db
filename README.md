# custom-docker-db

CLI que sobe um Postgres local em Docker já configurado com as credenciais do seu projeto. Ela lê o `.env` do diretório atual, extrai as variáveis `DB_*` / `DATABASE_*` e gera e executa um `docker compose` com essa configuração.

Hoje suporta apenas **Postgres**. O nome é genérico porque a ideia é evoluir para outros bancos no futuro.

> **Status: em desenvolvimento.** Ainda não há release publicada e o comando `up` ainda não sobe o container. Veja o [andamento](#andamento).

## Como vai funcionar

```bash
custom-docker-db                          # = up, descobre o .env em ./
custom-docker-db up --env-file=.env.local # usa só este arquivo
custom-docker-db up --ephemeral           # sem volume: os dados somem no down
custom-docker-db down                     # para e remove o container, mantém o volume
custom-docker-db exec                     # abre um shell (sh) dentro do container
```

| Comando / flag | O que faz |
| --- | --- |
| `up` (padrão) | Detecta a config, gera o compose, sobe o Postgres, espera o healthcheck e imprime a connection string |
| `down` | Para e remove o container; o volume é mantido |
| `exec` | Shell interativo dentro do container |
| `--env-file=<path>` | Usa apenas este arquivo, sem descoberta automática |
| `--ephemeral` | Sobe sem volume |

A imagem é sempre `postgres:16-alpine`. Requer Docker com o plugin `docker compose` v2. Suporte: Linux (x64 e arm64).

## Como o `.env` é lido

Sem `--env-file`, a CLI procura, nesta ordem, só no diretório atual: `.env`, `.env.local`, `.env.development`. Só contam chaves que começam com `DB_` ou `DATABASE_` (sem diferenciar maiúsculas).

| Arquivos com config de banco | Comportamento |
| --- | --- |
| 1 | Usa esse arquivo |
| Mais de 1 | Lista os arquivos e encerra, pedindo para renomear (ou usar `--env-file`) |
| Nenhum | Mostra o padrão e pergunta se pode criá-lo (ou acrescentá-lo ao `.env` existente) |

| Chave | Vira | Padrão se ausente |
| --- | --- | --- |
| `DB_USER` / `DB_USERNAME` | `POSTGRES_USER` | `postgres` |
| `DB_PASSWORD` / `DB_PASS` | `POSTGRES_PASSWORD` | `postgres` |
| `DB_NAME` / `DB_DATABASE` | `POSTGRES_DB` | nome da pasta |
| `DB_PORT` | porta do host | `5432` |
| `DB_HOST` | ignorado (o banco fica em `localhost`) | — |
| `DB_URL` | usada só se não houver variáveis separadas | — |

`DATABASE_*` é equivalente a `DB_*`. Se o mesmo campo aparece com os dois prefixos no mesmo arquivo, a CLI pede para deixar só um. A interpolação `${VAR}` é suportada. Exemplo do padrão criado quando não há config:

```bash
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=nome-da-pasta
DB_PORT=5432
```

## Andamento

| Marco | Entrega | Status |
| --- | --- | --- |
| M1 | Base do projeto: módulo Go, comandos `up`/`down`/`exec`, Makefile, CI | Concluído |
| M2 | Leitura e parsing do `.env` | Concluído |
| M3 | Oferta interativa do padrão | Concluído |
| M4 | Compose gerado e `up` (Docker, porta, healthcheck, `--ephemeral`) | Não iniciado |
| M5 | `down`, `exec` e mudança de config com pergunta sobre o volume | Não iniciado |
| M6 | Distribuição: GoReleaser, `install.sh`, pacotes npm `@pvfm/*` | Não iniciado |
| M7 | Release `v1.0.0` | Não iniciado |

A especificação completa está em [SPEC.md](SPEC.md).

## Desenvolvimento

Requer Go 1.27 ou mais novo.

```bash
make build   # gera bin/custom-docker-db
make test    # go test ./...
make lint    # go vet + gofmt
```

Estrutura:

- `cmd/custom-docker-db/`: `main` do binário
- `internal/cli/`: comandos (`cobra`)
- `internal/envconfig/`: descoberta e parsing do `.env`, oferta do padrão

## Instalação

Ainda não disponível. Após a v1, estará em:

- `curl -fsSL https://raw.githubusercontent.com/pvfm/custom-docker-db/main/install.sh | sh`
- `npx @pvfm/custom-docker-db`

## Licença

[MIT](LICENSE)
