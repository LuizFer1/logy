# Logy — Especificação de Design

**Data:** 2026-08-31  
**Status:** Design aprovado para revisão final

## Objetivo

Logy é um daemon local em Go que mantém um histórico pesquisável do trabalho do usuário por dia, período e projeto. Seu objetivo é reforçar a memória do usuário com resumos determinísticos offline e sínteses opcionais por IA.

O primeiro alvo é Windows. A arquitetura deve manter interfaces portáveis para permitir suporte futuro a macOS e Linux. O MVP será orientado por CLI; uma TUI poderá ser adicionada posteriormente sem alterar os coletores ou o armazenamento.

## Escopo do MVP

- Daemon Go em segundo plano.
- CLI para configuração, controle, consultas e notas.
- Descoberta automática de repositórios pela procura de pastas `.git` dentro de raízes configuradas.
- Exclusão e reativação individual de projetos.
- Armazenamento local em SQLite.
- Coletores de Git, terminal controlado pelo Logy e agentes de IA.
- Coletor de filesystem opcional e agrupado para detectar mudanças relevantes.
- Resumos diário, semanal, mensal e por projeto.
- `logy ask` desde o início, com respostas determinísticas e IA opcional.
- Mascaramento de segredos e exclusões configuráveis por projeto.
- Nenhum envio de dados em background.

Ficam fora do MVP: dashboard web, servidor HTTP local, integração profunda com todos os shells, edição automática de código e execução de comandos a partir do `ask`.

## Arquitetura

O processo principal é dividido em camadas:

```text
coletores -> normalizador de eventos -> SQLite -> serviços de consulta -> CLI/TUI futura
```

### Daemon

O daemon inicia com o usuário do Windows quando habilitado, mantém um lock para impedir instâncias duplicadas, grava logs rotacionados e executa shutdown seguro. Controle entre CLI e daemon usa named pipe local, sem porta TCP.

O daemon não faz polling agressivo. A descoberta de projetos usa as raízes configuradas e uma profundidade máxima; a frequência é configurável. Coletores gravam em pequenos lotes para reduzir I/O.

### Eventos

Todo coletor produz um evento normalizado com:

- timestamp inicial e final;
- projeto e diretório;
- tipo, como `git.commit`, `git.push`, `terminal.command` ou `agent.session`;
- resumo legível;
- payload estruturado;
- origem;
- nível de sensibilidade.

O normalizador aplica regras de exclusão e redaction antes da persistência.

## Descoberta e configuração

O usuário cadastra raízes de descoberta:

```text
logy root add C:\\dev
logy root list
```

O Logy procura pastas `.git` nessas raízes e acompanha automaticamente o diretório que contém cada uma. Novos repositórios são encontrados em varreduras periódicas leves. Projetos ignorados não são readicionados automaticamente.

Comandos de projeto:

```text
logy project list
logy project ignore C:\\dev\\projeto
logy project unignore C:\\dev\\projeto
```

Configuração por projeto controla coletores, retenção e conteúdo armazenado:

```yaml
enabled: true
collect:
  git: true
  terminal: true
  filesystem: false
  agents: true
store:
  diffs: false
  transcripts: false
ignore:
  - .env*
  - secrets/**
  - node_modules/**
  - vendor/**
redact:
  - token
  - password
  - api_key
```

Git e agentes são as fontes principais. Filesystem fica desativado por padrão. O terminal começa com `logy exec -- <comando>`, evitando interceptação invasiva do shell.

## Coletores

### Git

Registra commits, branches, checkout, merges, rebase, push, pull, fetch, diff/stat e estado do working tree. O conteúdo completo de diff é opcional; metadados e estatísticas são coletados por padrão.

### Terminal

`logy exec` registra comando, diretório, duração, código de saída e resumo da saída quando permitido. Não captura stdin, senhas ou conteúdo de terminais fora do wrapper inicial.

### Agentes de IA

Uma interface comum lê sessões de arquivos locais. Adaptadores específicos para ferramentas instaladas podem ser adicionados sem alterar o modelo de eventos. Caminhos e formatos são configuráveis por fonte.

### Filesystem

Quando habilitado, usa debounce e agrupamento por projeto para detectar mudanças relevantes, evitando um evento individual para cada gravação.

## Armazenamento

SQLite local em modo WAL, com escrita em lote e índices por data, projeto, tipo e origem. Tabelas principais:

- `roots`: raízes de descoberta;
- `projects`: repositórios detectados e status;
- `ignored_projects`: exclusões persistentes;
- `events`: eventos normalizados e payload;
- `notes`: notas manuais;
- `summaries`: resumos determinísticos ou de IA;
- `agent_sources`: adaptadores e caminhos;
- `collector_state`: cursores e última coleta.

Diffs e transcrições completas ficam em campos ou armazenamento separado, somente quando habilitados. Retenção e limpeza automática são configuráveis. O banco nunca armazena chaves de API.

## CLI e consultas

Comandos iniciais:

```text
logy start
logy stop
logy status
logy doctor
logy today
logy week
logy month
logy events --since 2026-08-01
logy project show C:\\dev\\app
logy note "Decidi migrar para PostgreSQL"
logy summarize --ai
logy ask "o que fiz no projeto Logy ontem?"
```

Consultas continuam disponíveis com o daemon parado. Ações de controle usam named pipe quando o daemon está em execução.

O serviço de consulta terá uma fronteira estável:

```go
type QueryService interface {
    Search(ctx context.Context, filter EventFilter) ([]Event, error)
    Summarize(ctx context.Context, filter EventFilter) (Summary, error)
    Ask(ctx context.Context, question string) (Answer, error)
}
```

`ask` interpreta termos de período, projeto e atividade, consulta eventos estruturados e responde deterministicamente quando possível. Perguntas abertas podem usar IA opcional. A resposta apresenta as evidências usadas e nunca executa comandos ou altera arquivos.

## Resumos e IA

Resumos offline agregam projetos, commits, branches, arquivos alterados, comandos, erros, sessões de agentes, notas, decisões e pontos de retomada. `logy summarize --ai` é explícito, usa um provedor configurado pelo usuário e envia apenas o contexto filtrado após exclusões e redaction.

A IA é desligada por padrão, não roda no background e recebe configuração de endpoint, modelo e referência a variável de ambiente para a chave.

## Privacidade e segurança

- Dados ficam 100% locais, salvo acionamento explícito de IA.
- Projetos, coletores, diffs e transcrições são configuráveis.
- `.env`, chaves e diretórios sensíveis têm exclusões padrão.
- Segredos conhecidos são mascarados antes da persistência.
- O usuário pode apagar eventos por projeto ou período.
- Named pipe é local ao usuário e não há API TCP no MVP.
- O Logy não executa ações via `ask`.

## Desempenho e confiabilidade

Metas operacionais do MVP:

- menos de 50 MB de memória em operação normal;
- CPU praticamente nula sem eventos;
- sem processo auxiliar permanente;
- gravações agrupadas e consultas offline;
- varredura limitada a raízes, profundidade e intervalo configuráveis.

O daemon deve retomar após reinicialização ou desligamento abrupto sem duplicar eventos quando o estado do coletor permitir. Falhas de um coletor são isoladas, registradas e não interrompem os demais.

## Verificação

O projeto terá testes unitários para normalização, filtros, redaction e agregações; testes de integração para SQLite, descoberta de `.git` e named pipe; fixtures para coletores; teste de retomada; e benchmarks de ingestão, descoberta e consulta.

`logy doctor` verificará permissões, banco, raízes, lock do daemon, espaço disponível e adaptadores de agentes.

## Recomendações futuras

Após validar o MVP, os próximos recursos de maior valor são TUI, integração opcional com shell, calendário de atividade, tags, detecção de pendências e adaptadores específicos de agentes. Um dashboard web e sincronização remota ficam deliberadamente posteriores para preservar o caráter leve e local.
