# Logy

Logy é um diário local de desenvolvimento para manter sua memória afiada. Ele acompanha o que foi feito por dia, semana, mês ou projeto e transforma atividade técnica em contexto fácil de recuperar.

O Logy roda como um daemon leve em segundo plano no Windows, grava os dados localmente e oferece uma CLI para consulta. Uma TUI será adicionada futuramente sobre a mesma camada de dados.

## O que ele acompanha

- Repositórios Git encontrados automaticamente pela presença de pastas `.git`.
- Commits, branches, merges, rebases, pushes, pulls, diffs e estado do working tree.
- Comandos executados através de `logy exec`.
- Sessões de agentes de IA por meio de uma captura genérica de logs e adaptadores específicos.
- Notas, decisões, erros recorrentes e possíveis pontos de retomada.
- Alterações de filesystem de forma opcional e agrupada.

## Princípios

- Local por padrão: nada é enviado para a internet em background.
- Privacidade configurável por projeto, com exclusões e mascaramento de segredos.
- Resumos determinísticos e offline sempre disponíveis.
- IA opcional e explícita, acionada apenas por comando.
- Baixo consumo de CPU, memória e disco.

## Uso planejado

```text
logy start
logy status

logy root add C:\\dev
logy root list
logy project list
logy project ignore C:\\dev\\projeto-arquivado
logy project unignore C:\\dev\\projeto-arquivado

logy today
logy week
logy month
logy project show C:\\dev\\meu-projeto
logy note "Decidi separar o coletor de Git do armazenamento"
logy ask "o que eu fiz neste projeto ontem?"
logy summarize --ai
```

O Logy procura repositórios dentro das raízes configuradas e começa a acompanhá-los automaticamente. Projetos ignorados permanecem fora do acompanhamento até serem reativados.

## Privacidade

O banco de dados é local e o usuário escolhe quais projetos e fontes acompanhar. Diffs e transcrições completas ficam desativados por padrão. Arquivos como `.env*`, diretórios de segredos e dependências podem ser excluídos, e padrões sensíveis como `token`, `password` e `api_key` são mascarados antes da persistência.

O comando `logy summarize --ai` só envia contexto quando o usuário o solicita e quando um provedor foi configurado. Chaves de API são lidas por variável de ambiente e não são armazenadas no banco.

## Status

O projeto está em desenvolvimento inicial. O design do MVP está documentado em [`docs/superpowers/specs/2026-08-31-logy-design.md`](docs/superpowers/specs/2026-08-31-logy-design.md).

## Direção técnica

- Go.
- SQLite em modo WAL.
- Daemon único em background.
- Named pipe local para controle entre CLI e daemon.
- CLI como primeira interface; TUI como próxima camada de apresentação.

## Desenvolvimento

O plano de implementação está em [`docs/superpowers/plans/2026-08-31-logy-mvp.md`](docs/superpowers/plans/2026-08-31-logy-mvp.md). A implementação será acompanhada por testes unitários, testes de integração e benchmarks de ingestão, descoberta e consulta.

## Licença

Ainda não definida.
