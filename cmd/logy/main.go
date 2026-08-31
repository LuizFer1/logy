package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"logy/internal/config"
	"logy/internal/control"
	"logy/internal/version"
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
	"exec",
	"events",
	"summarize",
	"purge",
	"startup",
	"root",
	"project",
	"doctor",
	"version",
	"update",
}

const usageSummary = "Usage: logy <command> [arguments]"

type cliOptions struct {
	Home string
	Pipe string
	Now  time.Time
	// StartBackground launches the daemon process. Nil uses a detached OS child.
	StartBackground func(home, pipe string) error
	Stdin           io.Reader
	Interactive     bool // force prompts (tests)
	NonInteractive  bool // force no prompts (tests/CI)
}

type cli struct {
	opts   cliOptions
	stdout io.Writer
	stderr io.Writer
	stdin  io.Reader
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	return runWith(cliOptions{}, args, stdout, stderr)
}

func runWith(opts cliOptions, args []string, stdout, stderr io.Writer) int {
	if opts.Home == "" {
		home, err := defaultHome()
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		opts.Home = home
	}
	if opts.Pipe == "" {
		if pipe := strings.TrimSpace(os.Getenv("LOGY_PIPE")); pipe != "" {
			opts.Pipe = pipe
		} else {
			opts.Pipe = control.PipeName()
		}
	}
	if opts.Now.IsZero() {
		opts.Now = time.Now()
	}
	stdin := opts.Stdin
	if stdin == nil {
		stdin = os.Stdin
	}

	c := &cli{opts: opts, stdout: stdout, stderr: stderr, stdin: stdin}

	if len(args) == 0 {
		writeUsage(stderr, "missing command")
		return 1
	}

	switch args[0] {
	case "-h", "--help", "help":
		writeUsage(stdout, "")
		return 0
	}

	handlers := map[string]func([]string) error{
		"start":     c.cmdStart,
		"status":    c.cmdStatus,
		"stop":      c.cmdStop,
		"today":     c.cmdToday,
		"week":      c.cmdWeek,
		"month":     c.cmdMonth,
		"ask":       c.cmdAsk,
		"note":      c.cmdNote,
		"exec":      c.cmdExec,
		"events":    c.cmdEvents,
		"summarize": c.cmdSummarize,
		"purge":     c.cmdPurge,
		"startup":   c.cmdStartup,
		"root":      c.cmdRoot,
		"project":   c.cmdProject,
		"doctor":    c.cmdDoctor,
		"version":   c.cmdVersion,
		"update":    c.cmdUpdate,
	}

	handler, ok := handlers[args[0]]
	if !ok {
		writeUsage(stderr, fmt.Sprintf("unknown command: %s", args[0]))
		return 1
	}

	if err := handler(args[1:]); err != nil {
		if errors.Is(err, errUpdateAvailable) {
			return 3
		}
		fmt.Fprintln(stderr, err)
		return 1
	}

	return 0
}

func defaultHome() (string, error) {
	if home := strings.TrimSpace(os.Getenv("LOGY_HOME")); home != "" {
		return home, nil
	}
	return config.ConfigDir()
}

func (c *cli) cmdVersion(args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("usage: logy version")
	}
	fmt.Fprintln(c.stdout, version.String())
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
