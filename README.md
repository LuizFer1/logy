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
logy start
logy status
logy stop
logy doctor

logy start
# na 1ª vez, o Logy pergunta as pastas-raiz (trabalho, startups, pessoal…)
# digite um caminho por linha; linha vazia termina

logy root add C:\trabalho
logy root add
# sem caminho: modo interativo para várias pastas
logy root list
logy project list
logy project show C:\dev\meu-projeto
logy project ignore C:\dev\projeto-arquivado
logy project unignore C:\dev\projeto-arquivado

logy exec -- go test ./...
logy note --project C:\dev\meu-projeto "Decidi separar o coletor de Git do armazenamento"

logy today
logy week
logy month
logy events --since 2026-08-01
logy ask "o que eu fiz neste projeto ontem?"
logy summarize
logy summarize --ai

logy purge --older-than 2025-01-01 --dry-run
logy purge --older-than 2025-01-01

logy startup enable
logy startup status
logy startup disable

logy update --check
logy update --yes
```

`logy start` sobe o daemon **em background** e devolve o terminal. O processo continua após fechar a janela; pare com `logy stop` ou ao desligar o PC. Use `logy start --foreground` só para depurar.

Consultas (`today`, `ask`, `events`, …) funcionam com o daemon parado.

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
