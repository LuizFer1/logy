package main

import (
	"fmt"
	"io"
	"os"
)

var commandNames = []string{
	"start",
	"status",
	"stop",
	"today",
	"week",
	"month",
	"ask",
	"note",
}

type commandHandler func([]string) error

var commandHandlers = map[string]commandHandler{
	"start":  noopCommand,
	"status": noopCommand,
	"stop":   noopCommand,
	"today":  noopCommand,
	"week":   noopCommand,
	"month":  noopCommand,
	"ask":    noopCommand,
	"note":   noopCommand,
}

const usageSummary = "Usage: logy <command> [arguments]"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		writeUsage(stderr, "missing command")
		return 1
	}

	switch args[0] {
	case "-h", "--help", "help":
		writeUsage(stdout, "")
		return 0
	}

	handler, ok := commandHandlers[args[0]]
	if !ok {
		writeUsage(stderr, fmt.Sprintf("unknown command: %s", args[0]))
		return 1
	}

	if err := handler(args[1:]); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	return 0
}

func noopCommand([]string) error {
	return nil
}

func writeUsage(w io.Writer, problem string) {
	if problem != "" {
		fmt.Fprintln(w, problem)
		fmt.Fprintln(w)
	}

	fmt.Fprintln(w, usageSummary)
	fmt.Fprintln(w, "Commands:")
	for _, name := range commandNames {
		fmt.Fprintf(w, "  %s\n", name)
	}
}
