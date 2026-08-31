package main

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

// commandAliases maps short names to canonical commands.
var commandAliases = map[string]string{
	"str": "start",
	"stp": "stop",
	"st":  "status",
	"td":  "today",
	"wk":  "week",
	"mo":  "month",
	"ak":  "ask",
	"nt":  "note",
	"ex":  "exec",
	"ev":  "events",
	"sm":  "summarize",
	"pg":  "purge",
	"su":  "startup",
	"rt":  "root",
	"pr":  "project",
	"doc": "doctor",
	"ver": "version",
	"up":  "update",
	"h":   "help",
}

type commandHelp struct {
	Name    string
	Aliases []string
	Usage   string
	Summary string
	Details string
}

var commandHelps = map[string]commandHelp{
	"start": {
		Name: "start", Aliases: []string{"str"},
		Usage:   "logy start [--foreground]",
		Summary: "Sobe o daemon em background",
		Details: `Inicia o Logy em segundo plano (o terminal volta).
Na primeira vez, pede pastas-raiz para procurar projetos Git.

Flags:
  --foreground, -f   Roda no terminal (debug)

Aliases: str

Exemplos:
  logy start
  logy str
  logy start --foreground`,
	},
	"stop": {
		Name: "stop", Aliases: []string{"stp"},
		Usage:   "logy stop",
		Summary: "Para o daemon",
		Details: `Envia stop pelo named pipe e encerra o processo em background.

Aliases: stp

Exemplos:
  logy stop
  logy stp`,
	},
	"status": {
		Name: "status", Aliases: []string{"st"},
		Usage:   "logy status",
		Summary: "Mostra se o daemon está rodando",
		Details: `Consulta o named pipe e lista coletores ativos.

Aliases: st

Exemplos:
  logy status
  logy st`,
	},
	"today": {
		Name: "today", Aliases: []string{"td"},
		Usage:   "logy today",
		Summary: "Resumo do dia",
		Details: `Agrega eventos de hoje (offline, sem daemon).

Aliases: td`,
	},
	"week": {
		Name: "week", Aliases: []string{"wk"},
		Usage:   "logy week",
		Summary: "Resumo da semana",
		Details: `Agrega eventos dos últimos 7 dias.

Aliases: wk`,
	},
	"month": {
		Name: "month", Aliases: []string{"mo"},
		Usage:   "logy month",
		Summary: "Resumo do mês",
		Details: `Agrega eventos do mês calendário atual.

Aliases: mo`,
	},
	"ask": {
		Name: "ask", Aliases: []string{"ak"},
		Usage:   "logy ask [--ai] <pergunta>",
		Summary: "Pergunta sobre o histórico",
		Details: `Interpreta período/projeto em português e lista evidências.
Com --ai, usa o provedor configurado (opt-in).

Aliases: ak

Exemplos:
  logy ask "o que fiz ontem no projeto Logy?"
  logy ak --ai "resumo da semana"`,
	},
	"note": {
		Name: "note", Aliases: []string{"nt"},
		Usage:   "logy note [--project <path>] <texto>",
		Summary: "Adiciona uma nota manual",
		Details: `Grava uma nota no SQLite, opcionalmente ligada a um projeto.

Aliases: nt

Exemplos:
  logy note "Decidi usar SQLite"
  logy nt --project C:\dev\app "migrar API"`,
	},
	"exec": {
		Name: "exec", Aliases: []string{"ex"},
		Usage:   "logy exec -- <comando> [args...]",
		Summary: "Roda um comando e registra o evento",
		Details: `Executa sem shell (argv). Não captura stdin.
Grava terminal.command no banco.

Aliases: ex

Exemplos:
  logy exec -- go test ./...
  logy ex -- git status`,
	},
	"events": {
		Name: "events", Aliases: []string{"ev"},
		Usage:   "logy events [--since YYYY-MM-DD] [--project <path>]",
		Summary: "Lista eventos",
		Details: `Consulta o banco com filtros opcionais.

Aliases: ev

Exemplos:
  logy events --since 2026-08-01
  logy ev --project C:\dev\app`,
	},
	"summarize": {
		Name: "summarize", Aliases: []string{"sm"},
		Usage:   "logy summarize [--ai]",
		Summary: "Resumo da semana",
		Details: `Resumo determinístico; --ai envia contexto sanitizado ao provedor.

Aliases: sm

Exemplos:
  logy summarize
  logy sm --ai`,
	},
	"purge": {
		Name: "purge", Aliases: []string{"pg"},
		Usage:   "logy purge --older-than YYYY-MM-DD [--project <path>] [--dry-run]",
		Summary: "Apaga eventos antigos",
		Details: `Não apaga notas, raízes nem ignores.

Aliases: pg

Exemplos:
  logy purge --older-than 2025-01-01 --dry-run
  logy pg --older-than 2025-01-01`,
	},
	"startup": {
		Name: "startup", Aliases: []string{"su"},
		Usage:   "logy startup enable|disable|status",
		Summary: "Registro no login do Windows",
		Details: `Habilita/desabilita autostart (HKCU Run + VBS oculto).

Aliases: su

Exemplos:
  logy startup enable
  logy su status`,
	},
	"root": {
		Name: "root", Aliases: []string{"rt"},
		Usage:   "logy root add [path] | logy root list",
		Summary: "Pastas-raiz de descoberta Git",
		Details: `Sem path, root add entra em modo interativo (várias pastas).
O Logy procura .git nas subpastas e para de descer ao achar um repo.

Aliases: rt

Exemplos:
  logy root add C:\trabalho
  logy rt add
  logy root list`,
	},
	"project": {
		Name: "project", Aliases: []string{"pr"},
		Usage:   "logy project list|show|ignore|unignore ...",
		Summary: "Projetos descobertos",
		Details: `Lista, mostra, ignora ou reativa um projeto.

Aliases: pr

Exemplos:
  logy project list
  logy pr show C:\dev\app
  logy project ignore C:\dev\arquivo`,
	},
	"doctor": {
		Name: "doctor", Aliases: []string{"doc"},
		Usage:   "logy doctor",
		Summary: "Diagnóstico local",
		Details: `Checa banco, raízes, daemon, lock, startup e AI.

Aliases: doc`,
	},
	"version": {
		Name: "version", Aliases: []string{"ver"},
		Usage:   "logy version",
		Summary: "Versão do binário",
		Details: `Mostra versão (e commit/data em builds de release).

Aliases: ver`,
	},
	"update": {
		Name: "update", Aliases: []string{"up"},
		Usage:   "logy update [--check] [--yes]",
		Summary: "Atualiza a partir do GitHub Releases",
		Details: `Baixa o asset, verifica SHA256 e substitui o executável.

Flags:
  --check   Só verifica (exit 3 se houver update)
  --yes     Atualiza sem perguntar

Aliases: up

Exemplos:
  logy update --check
  logy up --yes`,
	},
	"help": {
		Name: "help", Aliases: []string{"h"},
		Usage:   "logy help [comando]",
		Summary: "Ajuda geral ou de um comando",
		Details: `Sem argumentos: lista comandos e aliases.
Com comando (ou alias): ajuda detalhada.

Aliases: h

Exemplos:
  logy help
  logy help start
  logy help str
  logy help sm`,
	},
}

func resolveCommand(name string) string {
	name = strings.TrimSpace(strings.ToLower(name))
	if canonical, ok := commandAliases[name]; ok {
		return canonical
	}
	return name
}

func aliasesFor(command string) []string {
	var out []string
	for alias, canonical := range commandAliases {
		if canonical == command && alias != command {
			out = append(out, alias)
		}
	}
	sort.Strings(out)
	return out
}

func (c *cli) cmdHelp(args []string) error {
	if len(args) == 0 {
		writeUsage(c.stdout, "")
		return nil
	}
	if len(args) > 1 {
		return fmt.Errorf("usage: logy help [comando]")
	}
	name := resolveCommand(args[0])
	h, ok := commandHelps[name]
	if !ok {
		return fmt.Errorf("unknown command: %s\nrun: logy help", args[0])
	}
	writeCommandHelp(c.stdout, h)
	return nil
}

func writeCommandHelp(w io.Writer, h commandHelp) {
	fmt.Fprintf(w, "%s — %s\n\n", h.Name, h.Summary)
	fmt.Fprintf(w, "Usage:\n  %s\n\n", h.Usage)
	if len(h.Aliases) > 0 {
		fmt.Fprintf(w, "Aliases: %s\n\n", strings.Join(h.Aliases, ", "))
	}
	fmt.Fprintln(w, strings.TrimSpace(h.Details))
}

func writeUsage(w io.Writer, problem string) {
	if problem != "" {
		fmt.Fprintln(w, problem)
		fmt.Fprintln(w)
	}
	fmt.Fprintln(w, "Logy — diário local de desenvolvimento")
	fmt.Fprintln(w)
	fmt.Fprintln(w, usageSummary)
	fmt.Fprintln(w, "       logy help [comando]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Commands:")
	for _, name := range commandNames {
		h := commandHelps[name]
		alias := ""
		if als := aliasesFor(name); len(als) > 0 {
			alias = " (" + strings.Join(als, ", ") + ")"
		}
		summary := h.Summary
		if summary == "" {
			summary = name
		}
		fmt.Fprintf(w, "  %-12s %s%s\n", name, summary, alias)
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Examples:")
	fmt.Fprintln(w, "  logy str")
	fmt.Fprintln(w, "  logy st")
	fmt.Fprintln(w, "  logy sm")
	fmt.Fprintln(w, "  logy help start")
}
