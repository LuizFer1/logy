# Logy

Logy é um diário local de desenvolvimento para manter sua memória afiada. Ele acompanha o que foi feito por dia, semana, mês ou projeto e transforma atividade técnica em contexto fácil de recuperar.

O Logy roda como um daemon leve em segundo plano no Windows, grava os dados localmente e oferece uma CLI para consulta. Uma TUI poderá ser adicionada futuramente sobre a mesma camada de dados.

## Install

**Windows (PowerShell):**

```powershell
irm https://raw.githubusercontent.com/LuizFer1/logy/main/scripts/install.ps1 | iex
```

**macOS / Linux:**

```bash
curl -fsSL https://raw.githubusercontent.com/LuizFer1/logy/main/scripts/install.sh | bash
```

Pin with `LOGY_VERSION=v0.1.0`. Forks: `LOGY_GITHUB_REPO=owner/name`.

## Desenvolvimento

```powershell
go test ./...
.\scripts\build-dev.ps1
.\logyDEV.exe version
```

Opcional: defina `LOGY_HOME` para isolar config/dados (padrão `~/.logy`).

Release: ver [`docs/release.md`](docs/release.md).

## Uso

```text
logy help
logy help start
logy help sm

logy start          # str — 1ª vez pergunta pastas-raiz
logy status         # st
logy stop           # stp

logy root add C:\trabalho
logy root add       # interativo (várias pastas)
logy rt list
logy project list   # pr

logy exec -- go test ./...
logy today           # td
logy sm --ai        # summarize
logy ask "o que eu fiz ontem?"
logy up --yes       # update
```

### Aliases

| Comando | Atalho | Comando | Atalho |
|---------|--------|---------|--------|
| start | str | summarize | sm |
| stop | stp | events | ev |
| status | st | note | nt |
| today | td | exec | ex |
| week | wk | ask | ak |
| month | mo | purge | pg |
| root | rt | startup | su |
| project | pr | doctor | doc |
| version | ver | update | up |
| help | h | | |

`logy start` sobe o daemon **em background** e devolve o terminal. Pare com `logy stop` / `logy stp` ou ao desligar o PC.

## O que ele acompanha

- Repositórios Git encontrados automaticamente pela presença de pastas `.git`
- Commits, branches, status e diffstat
- Comandos via `logy exec` (sem shell-string; stdin descartado)
- Sessões de agentes via adaptador genérico JSON/JSONL (`internal/collectors`)
- Mudanças de filesystem opcionais (desligado por padrão)
- Notas manuais

## Privacidade

- Dados 100% locais; nada é enviado em background
- Diffs/transcrições completas desativados por padrão
- Exclusões padrão (`.env*`, `secrets/**`, `node_modules/**`, `vendor/**`)
- Mascaramento de `token`, `password`, `api_key`
- `logy summarize --ai` / `logy ask --ai` só quando pedido e com `ai.enabled` + endpoint + `keyEnv`
- Chaves de API vêm de variável de ambiente e não são gravadas no SQLite

## Configuração

Arquivo `LOGY_HOME/config.yaml` (criado ao salvar raízes):

```yaml
dataDir: C:\Users\voce\.logy\data
roots:
  - C:\dev
discoveryDepth: 3
discoveryInterval: 5m
ai:
  enabled: false
  endpoint: ""
  model: ""
  keyEnv: ""
```

## Direção técnica

- Go + SQLite (WAL, pure Go)
- Daemon único com lock exclusivo
- Named pipe local `\\.\pipe\logy-<user>` (sem TCP)
- CLI primeiro; TUI depois

## Desenvolvimento

```text
go test ./...
go vet ./...
```

Design: [`docs/superpowers/specs/2026-08-31-logy-design.md`](docs/superpowers/specs/2026-08-31-logy-design.md)  
Plano: [`docs/superpowers/plans/2026-08-31-logy-mvp.md`](docs/superpowers/plans/2026-08-31-logy-mvp.md)

## Licença

Ainda não definida.
