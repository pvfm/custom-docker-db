# Spec: custom-docker-db

> Versão em documento (com diagrama e comentários): https://claude.ai/code/artifact/4647afaf-5c96-4b94-a667-4c7f9bdc5eab
> Atualizado em 2026-09-29.

## Visão geral

`custom-docker-db` é uma CLI que sobe um Postgres local em Docker já configurado com as credenciais que o projeto espera. Ela lê o `.env` do diretório onde foi executada, extrai as variáveis que começam com `DB_` ou `DATABASE_` e gera e executa um `docker compose` com essa configuração.

O problema que resolve: hoje cada projeto exige montar à mão um compose com usuário, senha, nome do banco e porta iguais aos do `.env`. A ferramenta elimina esse passo e garante que o banco local bate com a config da aplicação.

Nome genérico de propósito: a ideia é suportar outros bancos no futuro. A v1 cobre só Postgres, com a lógica específica (imagem, healthcheck, esquema da URL) isolada para facilitar novos bancos.

Fora do escopo da v1: outros bancos (MySQL, Mongo), bancos remotos, migrations, extensões e scripts de init.

### Fluxo do `up`

```mermaid
flowchart TD
    A[Executa a CLI em ./] --> B{--env-file?}
    B -- sim --> C[Lê o arquivo indicado]
    B -- não --> D[Busca em ordem<br/>.env → .env.local → .env.development]
    D --> E{Arquivos com config?}
    E -- nenhum --> F[Mostra o padrão e oferece criar<br/>aceitou: cria o .env e segue]
    E -- vários --> G[Lista e encerra<br/>pedindo para renomear]
    E -- um --> H[Monta a config<br/>separadas, senão DB_URL]
    C --> H
    H --> I{Porta livre?}
    I -- ocupada --> J[Avisa a porta e encerra]
    I -- livre --> K[compose up + healthcheck]
    K --> L[Imprime connection string]
```

Só um arquivo com config segue direto. Sem nenhum, a CLI oferece criar o padrão; com vários, ou com porta ocupada, encerra sem mexer em nada.

## Uso da CLI

Três subcomandos na v1 (`up` como padrão, `down` e `exec`) e duas flags: `--env-file` e `--ephemeral`.

```bash
custom-docker-db                       # = up, descoberta automática em ./
custom-docker-db up --env-file=.env.local
custom-docker-db up --ephemeral        # banco sem volume, some no down
custom-docker-db down                  # para e remove o container, mantém o volume
custom-docker-db exec                  # abre um shell dentro do container
```

| Comando / flag | O que faz |
| --- | --- |
| `up` | Detecta a config, gera o compose, sobe e espera o healthcheck; imprime a connection string |
| `down` | `docker compose down` do projeto; o volume persistente é mantido |
| `exec` | Shell interativo no container (`sh`: a imagem alpine não traz `bash`) |
| `--env-file=<path>` | Usa só este arquivo; desliga a descoberta. Única forma aceita, sem `-env-file` |
| `--ephemeral` | Sobe sem volume: os dados somem no `down` |

Sugestões para depois da v1: `psql` (abre o psql já autenticado), `url` (só imprime a connection string), `logs`, `status` e `reset` (`down` + apaga volume + `up`).

Imagem fixa: `postgres:16-alpine`, sem flag e sem mensagem sobre ela.

## Descoberta e parsing do .env

Só entram chaves que começam com `DB_` ou `DATABASE_` (sem diferenciar maiúsculas); variáveis separadas têm prioridade e `DB_URL` / `DATABASE_URL` é o fallback.

**Ordem de busca** (sem `--env-file`), só no diretório atual:

1. `.env`
2. `.env.local`
3. `.env.development`

**Resultado da busca:**

| Arquivos com config de banco | Comportamento |
| --- | --- |
| Exatamente 1 | Usa esse arquivo |
| Mais de 1 | Lista os arquivos encontrados e encerra, pedindo para renomear os que não devem ser usados (ou usar `--env-file`) |
| Nenhum | Mostra o padrão esperado (prefixo `DB_`) e pergunta se pode criá-lo. Sem `.env`, cria o arquivo; com `.env` sem config de banco, acrescenta as linhas no final. Informa o que foi escrito e segue |

**Conflito de prefixo:** se o mesmo campo aparece como `DB_X` e `DATABASE_X` (ex.: `DB_USER` e `DATABASE_USER`), a CLI informa que existem os dois, pede para deixar só um e encerra.

**Padrão esperado / criado:**

```bash
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=<nome-do-diretorio>
DB_PORT=5432
```

**Mapeamento** (`DB_X` e `DATABASE_X` são equivalentes):

| Chave no .env | Vira no container | Padrão se ausente |
| --- | --- | --- |
| `DB_USER` / `DB_USERNAME` | `POSTGRES_USER` | `postgres` |
| `DB_PASSWORD` / `DB_PASS` | `POSTGRES_PASSWORD` | `postgres` |
| `DB_NAME` / `DB_DATABASE` | `POSTGRES_DB` | nome do diretório |
| `DB_PORT` | porta do host em `ports: "<porta>:5432"` | `5432` |
| `DB_HOST` | ignorado; o banco sempre fica em `localhost` | — |
| `DB_URL` / `DATABASE_URL` | só usada se não houver variáveis separadas; `postgres://user:pass@host:port/db` | — |

**Parsing:** formato dotenv: `CHAVE=valor`, aspas, comentários `#`, prefixo `export` opcional e interpolação `${VAR}` (ex.: `DATABASE_URL=postgres://${DB_USER}:${DB_PASSWORD}@localhost/app`), que a `godotenv` já resolve.

## Docker compose gerado

Antes de subir, a CLI checa se a porta do host está livre; se estiver ocupada, avisa qual porta e encerra sem fazer nada.

```yaml
services:
  postgres:
    image: postgres:16-alpine
    container_name: cdd-<dir>-<hash>
    environment:
      POSTGRES_USER: app
      POSTGRES_PASSWORD: secret
      POSTGRES_DB: app_dev
    ports:
      - "5432:5432"
    volumes:            # omitido com --ephemeral
      - pgdata:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U app -d app_dev"]
      interval: 2s
      retries: 30
volumes:
  pgdata:
```

- **Nome do projeto/container:** `cdd-<dir>-<hash>`, para não colidir entre projetos. `down` e `exec` acham o container por esse nome.
- **Persistência:** volume nomeado por padrão; com `--ephemeral` não há volume e os dados somem no `down`.
- **Extensões e scripts de init:** fora da v1.
- **Pré-requisito:** Docker com o plugin `docker compose` v2; a CLI checa no início e falha com mensagem clara.

**Onde fica o compose gerado:** `~/.local/share/custom-docker-db/<dir>-<hash>/docker-compose.yml`, fora do repo. O `<hash>` são os 8 primeiros caracteres do SHA-256 do caminho absoluto do projeto, para dois projetos com o mesmo nome de pasta não dividirem o mesmo arquivo. Respeita `$XDG_DATA_HOME` quando definido.

## Erros e casos de borda

Toda falha sai com código diferente de zero e uma mensagem que diz o que fazer.

| Situação | Comportamento |
| --- | --- |
| Nenhum dos três `.env` tem `DB_`/`DATABASE_` | Mostra o padrão e pergunta se cria; aceitando, escreve, informa e segue |
| Mais de um `.env` com config de banco | Lista os arquivos e encerra, pedindo para renomear |
| `DB_X` e `DATABASE_X` no mesmo arquivo | Informa o conflito, pede para deixar só um e encerra |
| `--env-file` inexistente | Erro com o caminho |
| `--env-file` sem `DB_`/`DATABASE_` | Mostra o padrão e pergunta se acrescenta ao arquivo |
| URL com esquema diferente de `postgres`/`postgresql` | Erro: banco não suportado |
| Docker ou `docker compose` ausente / daemon parado | Erro antes de gerar qualquer arquivo |
| Porta do host ocupada | Avisa a porta e encerra, sem fazer nada |
| Container do projeto já rodando com a mesma config | Não faz nada, imprime a connection string |
| Container existe com config diferente | Pergunta se apaga o volume e recria; recusando, encerra sem mudar nada |
| Healthcheck não passa em 60 s | Erro e mostra as últimas linhas de `docker compose logs` |

**Por que perguntar sobre o volume:** o Postgres só lê `POSTGRES_USER`, `POSTGRES_PASSWORD` e `POSTGRES_DB` na primeira inicialização do volume, então mudar a senha no `.env` sem recriar o volume deixa a aplicação sem conseguir autenticar. A CLI grava um hash da config como label do container para detectar a mudança. Com `--ephemeral` não há volume e o container é recriado sem perguntar.

## Linguagem e distribuição: Go, via install.sh e npm

Decidido: **Go**, alvo só Linux (x64 e arm64) na v1. O binário é único e estático, então o mesmo artefato serve ao `install.sh` e ao npm.

| Critério | Go | Rust | Node/TS | Python |
| --- | --- | --- | --- | --- |
| Distribuição | Binário único | Binário único | Precisa Node | Precisa Python |
| Cross-compile | `GOOS/GOARCH` nativo | Possível, mais atrito | N/A | N/A |
| Velocidade de desenvolvimento | Alta | Média | Alta | Alta |
| Parsing .env/URL/YAML | `godotenv`, `net/url`, `yaml.v3` | `dotenvy`, `url`, `serde_yaml` | `dotenv`, `URL`, `yaml` | `python-dotenv`, `urllib`, `PyYAML` |

**Bibliotecas:** `spf13/cobra` (subcomandos e flags), `joho/godotenv` (.env, já com interpolação), `text/template` (compose), `os/exec` (chama `docker compose`).

**Distribuição:**

1. **GoReleaser** gera os binários `linux-amd64` e `linux-arm64` e publica no GitHub Releases de `github.com/pvfm/custom-docker-db`.
2. **install.sh:** `curl -fsSL https://raw.githubusercontent.com/pvfm/custom-docker-db/main/install.sh | sh` detecta a arquitetura, baixa o binário da release, confere o checksum e instala em `~/.local/bin`.
3. **npm (Go dentro do npm):** mesmo padrão do esbuild e do Biome. `@pvfm/custom-docker-db` é o pacote principal, com um `bin` em JS de poucas linhas; `@pvfm/cdd-linux-x64` e `@pvfm/cdd-linux-arm64` entram em `optionalDependencies`, cada um com o binário Go. Uso: `npx @pvfm/custom-docker-db`.

**Escopo npm:** `@pvfm` exige uma conta ou organização `pvfm` no npm (o escopo é do npm, não do GitHub). Pacotes com escopo são privados por padrão: publicar com `npm publish --access public`.

**Runtime:** v1 só Docker. A chamada ao runtime fica isolada em um ponto do código para suportar Podman depois.

## Perguntas em aberto

Nenhuma: todas as decisões da v1 estão fechadas e incorporadas acima.

## Marcos

Sete marcos levam da base do projeto à v1 publicada; cada um termina com algo que dá para rodar ou testar. Status atualizado no doc.

- [x] **M1 · Base do projeto:** módulo Go `github.com/pvfm/custom-docker-db`, `cobra` com `up`/`down`/`exec` vazios, Makefile e CI (lint + testes). *Pronto quando:* `go build` gera o binário e `--help` lista os comandos.
- [x] **M2 · Leitura do .env:** busca em ordem, filtro `DB_`/`DATABASE_`, variáveis separadas com fallback para URL, interpolação, conflito de prefixo. *Pronto quando:* testes cobrem 0, 1 e vários arquivos, URL, conflito e `--env-file`.
- [x] **M3 · Padrão interativo:** mostra o padrão, pergunta, cria ou acrescenta ao `.env` e informa. *Pronto quando:* rodar numa pasta vazia cria o `.env` depois de confirmar.
- [ ] **M4 · Compose e `up`:** template em `~/.local/share/.../<dir>-<hash>/`, checagem de Docker e porta, `compose up`, healthcheck, connection string, `--ephemeral`. *Pronto quando:* `up` num projeto real deixa o banco acessível pela app.
- [ ] **M5 · `down`, `exec` e mudança de config:** `down`, `exec` com `sh`, label com hash da config e pergunta sobre apagar o volume. *Pronto quando:* mudar a senha no `.env` e rodar `up` leva à pergunta.
- [ ] **M6 · Distribuição:** GoReleaser (linux amd64/arm64), `install.sh` com checksum, pacotes `@pvfm/*` no npm. *Pronto quando:* `curl ... | sh` e `npx @pvfm/custom-docker-db` funcionam numa máquina limpa.
- [ ] **M7 · Release v1.0.0:** README, tag `v1.0.0`, release publicada. *Pronto quando:* um usuário instala e sobe o banco só lendo o README.
