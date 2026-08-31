package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"logy/internal/ai"
	"logy/internal/collectors"
	"logy/internal/config"
	"logy/internal/control"
	"logy/internal/daemon"
	"logy/internal/discovery"
	"logy/internal/events"
	"logy/internal/maintenance"
	"logy/internal/platform"
	"logy/internal/report"
	"logy/internal/storage"
)

func (c *cli) loadConfig() (config.Config, error) {
	cfg, err := config.LoadConfig(c.configPath())
	if err != nil {
		return config.Config{}, err
	}
	def := config.DefaultConfig()
	if cfg.DataDir == "" || cfg.DataDir == def.DataDir {
		cfg.DataDir = filepath.Join(c.opts.Home, "data")
	}
	return cfg, nil
}

func (c *cli) configPath() string {
	return filepath.Join(c.opts.Home, "config.yaml")
}

func (c *cli) saveConfig(cfg config.Config) error {
	return config.SaveConfig(c.configPath(), cfg)
}

func (c *cli) openDB() (*storage.DB, error) {
	cfg, err := c.loadConfig()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(cfg.DataDir, 0755); err != nil {
		return nil, err
	}
	return storage.Open(filepath.Join(cfg.DataDir, "logy.db"))
}

func absPath(value string) (string, error) {
	path, err := filepath.Abs(value)
	if err != nil {
		return "", err
	}
	return filepath.Clean(path), nil
}

func (c *cli) cmdRoot(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: logy root add <path> | logy root list")
	}
	switch args[0] {
	case "add":
		if len(args) < 2 {
			return errors.New("usage: logy root add <path>")
		}
		path, err := absPath(args[1])
		if err != nil {
			return err
		}
		cfg, err := c.loadConfig()
		if err != nil {
			return err
		}
		for _, existing := range cfg.Roots {
			if strings.EqualFold(existing, path) {
				return nil
			}
		}
		cfg.Roots = append(cfg.Roots, path)
		if err := c.saveConfig(cfg); err != nil {
			return err
		}
		db, err := c.openDB()
		if err != nil {
			return err
		}
		defer db.Close()
		return db.AddRoot(context.Background(), path)
	case "list":
		cfg, err := c.loadConfig()
		if err != nil {
			return err
		}
		for _, root := range cfg.Roots {
			fmt.Fprintln(c.stdout, root)
		}
		return nil
	default:
		return fmt.Errorf("unknown root command: %s", args[0])
	}
}

func (c *cli) cmdProject(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: logy project list | show | ignore | unignore")
	}
	db, err := c.openDB()
	if err != nil {
		return err
	}
	defer db.Close()
	ctx := context.Background()

	switch args[0] {
	case "list":
		projects, err := db.ListProjects(ctx)
		if err != nil {
			return err
		}
		ignored, err := db.ListIgnoredPaths(ctx)
		if err != nil {
			return err
		}
		ignoredSet := make(map[string]struct{}, len(ignored))
		for _, path := range ignored {
			ignoredSet[strings.ToLower(path)] = struct{}{}
		}
		for _, project := range projects {
			if _, ok := ignoredSet[strings.ToLower(project.Path)]; ok {
				fmt.Fprintf(c.stdout, "%s  [ignored]\n", project.Path)
				continue
			}
			fmt.Fprintln(c.stdout, project.Path)
		}
		return nil
	case "show":
		if len(args) < 2 {
			return errors.New("usage: logy project show <path>")
		}
		path, err := absPath(args[1])
		if err != nil {
			return err
		}
		fmt.Fprintln(c.stdout, path)
		ignored, err := db.IsIgnored(ctx, path)
		if err != nil {
			return err
		}
		if ignored {
			fmt.Fprintln(c.stdout, "status: ignored")
		} else {
			fmt.Fprintln(c.stdout, "status: tracked")
		}
		notes, err := db.ListNotes(ctx, path, time.Time{}, time.Time{})
		if err != nil {
			return err
		}
		for _, note := range notes {
			fmt.Fprintf(c.stdout, "note: %s\n", note.Content)
		}
		evts, err := db.Search(ctx, events.EventFilter{ProjectPath: path})
		if err != nil {
			return err
		}
		fmt.Fprintf(c.stdout, "events: %d\n", len(evts))
		return nil
	case "ignore":
		if len(args) < 2 {
			return errors.New("usage: logy project ignore <path>")
		}
		path, err := absPath(args[1])
		if err != nil {
			return err
		}
		return db.IgnoreProject(ctx, path)
	case "unignore":
		if len(args) < 2 {
			return errors.New("usage: logy project unignore <path>")
		}
		path, err := absPath(args[1])
		if err != nil {
			return err
		}
		return db.UnignoreProject(ctx, path)
	default:
		return fmt.Errorf("unknown project command: %s", args[0])
	}
}

func (c *cli) cmdNote(args []string) error {
	projectPath := ""
	var parts []string
	for i := 0; i < len(args); i++ {
		if args[i] == "--project" {
			if i+1 >= len(args) {
				return errors.New("usage: logy note [--project <path>] <text>")
			}
			path, err := absPath(args[i+1])
			if err != nil {
				return err
			}
			projectPath = path
			i++
			continue
		}
		parts = append(parts, args[i])
	}
	content := strings.TrimSpace(strings.Join(parts, " "))
	if content == "" {
		return errors.New("usage: logy note [--project <path>] <text>")
	}
	db, err := c.openDB()
	if err != nil {
		return err
	}
	defer db.Close()
	return db.AddNote(context.Background(), projectPath, content)
}

func (c *cli) cmdEvents(args []string) error {
	filter := events.EventFilter{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--since":
			if i+1 >= len(args) {
				return errors.New("usage: logy events [--since YYYY-MM-DD] [--project <path>]")
			}
			day, err := time.ParseInLocation("2006-01-02", args[i+1], c.opts.Now.Location())
			if err != nil {
				return fmt.Errorf("invalid --since date: %w", err)
			}
			filter.From = day
			i++
		case "--project":
			if i+1 >= len(args) {
				return errors.New("usage: logy events [--since YYYY-MM-DD] [--project <path>]")
			}
			path, err := absPath(args[i+1])
			if err != nil {
				return err
			}
			filter.ProjectPath = path
			i++
		default:
			return fmt.Errorf("unknown events flag: %s", args[i])
		}
	}
	db, err := c.openDB()
	if err != nil {
		return err
	}
	defer db.Close()
	evts, err := db.Search(context.Background(), filter)
	if err != nil {
		return err
	}
	for _, ev := range evts {
		fmt.Fprintf(c.stdout, "%s %s %s\n", ev.StartedAt.Format(time.RFC3339), ev.Type, ev.Summary)
	}
	return nil
}

func (c *cli) cmdToday([]string) error { return c.printPeriodSummary("today", report.PeriodToday) }
func (c *cli) cmdWeek([]string) error  { return c.printPeriodSummary("week", report.PeriodWeek) }
func (c *cli) cmdMonth([]string) error { return c.printPeriodSummary("month", report.PeriodMonth) }

func (c *cli) printPeriodSummary(name string, period func(time.Time) (time.Time, time.Time)) error {
	from, to := period(c.opts.Now)
	db, err := c.openDB()
	if err != nil {
		return err
	}
	defer db.Close()
	summary, err := report.Summarize(context.Background(), db, events.EventFilter{From: from, To: to}, name)
	if err != nil {
		return err
	}
	fmt.Fprintln(c.stdout, summary.Text)
	return nil
}

func (c *cli) cmdAsk(args []string) error {
	useAI := false
	var parts []string
	for _, arg := range args {
		if arg == "--ai" {
			useAI = true
			continue
		}
		parts = append(parts, arg)
	}
	question := strings.TrimSpace(strings.Join(parts, " "))
	if question == "" {
		return errors.New("usage: logy ask [--ai] <question>")
	}
	db, err := c.openDB()
	if err != nil {
		return err
	}
	defer db.Close()
	answer, err := report.Ask(context.Background(), db, question, c.opts.Now)
	if err != nil {
		return err
	}
	text := answer.Text
	if useAI {
		text, err = c.withOptionalAI(text, true)
		if err != nil {
			return err
		}
	}
	fmt.Fprintln(c.stdout, text)
	if len(answer.Evidence) > 0 {
		fmt.Fprintln(c.stdout, "evidence:")
		for _, e := range answer.Evidence {
			fmt.Fprintf(c.stdout, "  %s %s %s\n", e.StartedAt.Format(time.RFC3339), e.EventID, e.Summary)
		}
	}
	return nil
}

func (c *cli) cmdSummarize(args []string) error {
	useAI := false
	for _, arg := range args {
		if arg == "--ai" {
			useAI = true
			continue
		}
		return fmt.Errorf("unknown summarize flag: %s", arg)
	}
	from, to := report.PeriodWeek(c.opts.Now)
	db, err := c.openDB()
	if err != nil {
		return err
	}
	defer db.Close()
	filter := events.EventFilter{From: from, To: to}
	summary, err := report.Summarize(context.Background(), db, filter, "week")
	if err != nil {
		return err
	}
	text := summary.Text
	if useAI {
		text, err = c.withOptionalAI(text, true)
		if err != nil {
			return err
		}
	}
	fmt.Fprintln(c.stdout, text)
	return nil
}

func (c *cli) withOptionalAI(deterministic string, useAI bool) (string, error) {
	cfg, err := c.loadConfig()
	if err != nil {
		return "", err
	}
	provider, err := ai.NewHTTPProvider(cfg.AI)
	if err != nil {
		return "", err
	}
	if useAI && provider == nil {
		return deterministic + "\n(note: AI disabled or not configured)", nil
	}
	db, err := c.openDB()
	if err != nil {
		return "", err
	}
	defer db.Close()
	from, to := report.PeriodWeek(c.opts.Now)
	evts, err := db.Search(context.Background(), events.EventFilter{From: from, To: to})
	if err != nil {
		return "", err
	}
	prompt := ai.BuildContext(evts, ai.ContextRules{
		ExcludeGlobs: config.DefaultProjectConfig().Ignore,
	})
	return ai.AnswerWithOptionalAI(deterministic, useAI, provider, prompt), nil
}

func (c *cli) cmdPurge(args []string) error {
	opts := maintenance.RetentionOptions{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--older-than":
			if i+1 >= len(args) {
				return errors.New("usage: logy purge --older-than YYYY-MM-DD [--project <path>] [--dry-run]")
			}
			day, err := time.ParseInLocation("2006-01-02", args[i+1], c.opts.Now.Location())
			if err != nil {
				return fmt.Errorf("invalid --older-than date: %w", err)
			}
			opts.OlderThan = day
			i++
		case "--project":
			if i+1 >= len(args) {
				return errors.New("usage: logy purge --older-than YYYY-MM-DD [--project <path>] [--dry-run]")
			}
			path, err := absPath(args[i+1])
			if err != nil {
				return err
			}
			opts.ProjectPath = path
			i++
		case "--dry-run":
			opts.DryRun = true
		default:
			return fmt.Errorf("unknown purge flag: %s", args[i])
		}
	}
	if opts.OlderThan.IsZero() {
		return errors.New("usage: logy purge --older-than YYYY-MM-DD [--project <path>] [--dry-run]")
	}
	db, err := c.openDB()
	if err != nil {
		return err
	}
	defer db.Close()
	result, err := maintenance.PurgeEvents(context.Background(), db, opts)
	if err != nil {
		return err
	}
	if result.DryRun {
		fmt.Fprintf(c.stdout, "dry-run: would delete %d events\n", result.Deleted)
		return nil
	}
	fmt.Fprintf(c.stdout, "deleted %d events\n", result.Deleted)
	return nil
}

func (c *cli) cmdStartup(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: logy startup enable|disable|status")
	}
	switch args[0] {
	case "enable":
		return platform.EnableStartup(c.opts.Home)
	case "disable":
		return platform.DisableStartup()
	case "status":
		enabled, err := platform.StartupEnabled()
		if err != nil {
			return err
		}
		if enabled {
			fmt.Fprintln(c.stdout, "startup: enabled")
		} else {
			fmt.Fprintln(c.stdout, "startup: disabled")
		}
		return nil
	default:
		return fmt.Errorf("unknown startup command: %s", args[0])
	}
}

func (c *cli) cmdDoctor([]string) error {
	cfg, err := c.loadConfig()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(c.opts.Home, 0755); err != nil {
		return err
	}

	db, err := c.openDB()
	if err != nil {
		fmt.Fprintf(c.stdout, "database: error (%v)\n", err)
		return err
	}
	defer db.Close()
	fmt.Fprintf(c.stdout, "database: %s\n", filepath.Join(cfg.DataDir, "logy.db"))

	fmt.Fprintf(c.stdout, "roots: %d\n", len(cfg.Roots))
	for _, root := range cfg.Roots {
		fmt.Fprintf(c.stdout, "  %s\n", root)
	}

	fmt.Fprintf(c.stdout, "lock: %s\n", filepath.Join(c.opts.Home, "logy.lock"))

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	conn, err := control.DialName(ctx, c.opts.Pipe)
	if err != nil {
		fmt.Fprintln(c.stdout, "daemon: not running")
	} else {
		_ = conn.Close()
		fmt.Fprintln(c.stdout, "daemon: running")
	}

	if enabled, err := platform.StartupEnabled(); err != nil {
		fmt.Fprintf(c.stdout, "startup: error (%v)\n", err)
	} else if enabled {
		fmt.Fprintln(c.stdout, "startup: enabled")
	} else {
		fmt.Fprintln(c.stdout, "startup: disabled")
	}

	if cfg.AI.Enabled {
		fmt.Fprintf(c.stdout, "ai: enabled (endpoint=%s model=%s keyEnv=%s)\n", cfg.AI.Endpoint, cfg.AI.Model, cfg.AI.KeyEnv)
	} else {
		fmt.Fprintln(c.stdout, "ai: disabled")
	}
	fmt.Fprintln(c.stdout, "agents: none configured")
	return nil
}

func (c *cli) cmdStart([]string) error {
	if err := os.MkdirAll(c.opts.Home, 0755); err != nil {
		return err
	}
	cfg, err := c.loadConfig()
	if err != nil {
		return err
	}
	db, err := c.openDB()
	if err != nil {
		return err
	}
	defer db.Close()

	logger, logFile, err := daemon.NewRotatingLogger(filepath.Join(c.opts.Home, "logy.log"))
	if err != nil {
		return err
	}
	defer logFile.Close()

	ignored, err := db.ListIgnoredPaths(context.Background())
	if err != nil {
		return err
	}
	found, err := discovery.Scan(cfg.Roots, discovery.ScanOptions{
		MaxDepth: cfg.DiscoveryDepth,
		Ignored:  ignored,
	})
	if err != nil {
		return err
	}
	var projects []collectors.Project
	for _, result := range found {
		if err := db.UpsertProject(context.Background(), storage.Project{Path: result.Path, Name: result.Name}); err != nil {
			return err
		}
		projects = append(projects, collectors.Project{Path: result.Path, Name: result.Name})
	}

	d, err := daemon.New(daemon.Options{
		LockPath:        filepath.Join(c.opts.Home, "logy.lock"),
		Sink:            db,
		Logger:          logger,
		FlushSize:       4,
		FlushInterval:   5 * time.Second,
		CollectInterval: cfg.DiscoveryInterval,
		Collectors: []daemon.Collector{
			collectors.FanOut{Collector: collectors.Git{}, Projects: projects},
		},
	})
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := d.Start(ctx); err != nil {
		return err
	}

	ln, err := control.ListenName(c.opts.Pipe)
	if err != nil {
		_ = d.Stop(context.Background())
		return err
	}

	handler := control.Handler{
		Status: func() control.StatusPayload {
			st := d.Status(context.Background())
			return control.StatusPayload{Running: st.Running, Collectors: st.Collectors}
		},
		Stop: func() error {
			cancel()
			return nil
		},
		Reload: func() error { return nil },
	}

	serveErr := control.Serve(ctx, ln, handler)
	stopErr := d.Stop(context.Background())
	if serveErr != nil {
		return serveErr
	}
	return stopErr
}

func (c *cli) cmdExec(args []string) error {
	command, commandArgs, err := parseExecArgs(args)
	if err != nil {
		return err
	}
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	evt, err := collectors.Exec(ctx, collectors.Project{Path: wd, Name: filepath.Base(wd)}, command, commandArgs)
	if err != nil {
		return err
	}
	db, err := c.openDB()
	if err != nil {
		return err
	}
	defer db.Close()
	if err := db.AppendEvents(context.Background(), []events.Event{evt}); err != nil {
		return err
	}
	if evt.Summary != "" {
		fmt.Fprintln(c.stdout, evt.Summary)
	}
	return nil
}

func parseExecArgs(args []string) (string, []string, error) {
	if len(args) > 0 && args[0] == "--" {
		args = args[1:]
	}
	if len(args) == 0 {
		return "", nil, errors.New("usage: logy exec -- <command> [args]")
	}
	return args[0], args[1:], nil
}

func (c *cli) cmdStatus([]string) error {
	resp, err := c.control(control.Request{Command: "status"})
	if err != nil {
		return err
	}
	if resp.Status == nil {
		fmt.Fprintln(c.stdout, "running: true")
		return nil
	}
	fmt.Fprintf(c.stdout, "running: %v\n", resp.Status.Running)
	if len(resp.Status.Collectors) > 0 {
		fmt.Fprintf(c.stdout, "collectors: %s\n", strings.Join(resp.Status.Collectors, ", "))
	}
	return nil
}

func (c *cli) cmdStop([]string) error {
	_, err := c.control(control.Request{Command: "stop"})
	return err
}

func (c *cli) control(req control.Request) (control.Response, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	conn, err := control.DialName(ctx, c.opts.Pipe)
	if err != nil {
		return control.Response{}, errors.New("logy daemon is not running")
	}
	defer conn.Close()
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}
	resp, err := control.Call(conn, req)
	if err != nil {
		return control.Response{}, err
	}
	if !resp.OK {
		if resp.Error != "" {
			return resp, errors.New(resp.Error)
		}
		return resp, errors.New("control request failed")
	}
	return resp, nil
}
