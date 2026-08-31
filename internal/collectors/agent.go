package collectors

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"logy/internal/events"
)

// AgentSource describes a configurable local agent session log source.
type AgentSource struct {
	Name        string
	PathPattern string // glob or directory of session files
	Format      string // "jsonl" or "json"
}

// Agent reads local agent session logs into events.
type Agent struct{}

func (Agent) Name() string { return "agent" }

// ReadSessions loads session records matching source.PathPattern.
// Malformed lines are skipped; the overall read still succeeds.
func (Agent) ReadSessions(ctx context.Context, source AgentSource) ([]events.Event, error) {
	files, err := resolveAgentFiles(source.PathPattern)
	if err != nil {
		return nil, err
	}

	format := strings.ToLower(strings.TrimSpace(source.Format))
	if format == "" {
		format = "jsonl"
	}

	var out []events.Event
	for _, file := range files {
		if ctx.Err() != nil {
			return out, ctx.Err()
		}
		evts, err := readAgentFile(ctx, file, format, source.Name)
		if err != nil {
			continue
		}
		out = append(out, evts...)
	}
	return out, nil
}

func resolveAgentFiles(pattern string) ([]string, error) {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return nil, nil
	}
	if info, err := os.Stat(pattern); err == nil && info.IsDir() {
		entries, err := os.ReadDir(pattern)
		if err != nil {
			return nil, err
		}
		var files []string
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			files = append(files, filepath.Join(pattern, e.Name()))
		}
		return files, nil
	}
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	return matches, nil
}

func readAgentFile(ctx context.Context, path, format, sourceName string) ([]events.Event, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	switch format {
	case "json":
		return readAgentJSON(f, path, sourceName)
	default:
		return readAgentJSONL(ctx, f, path, sourceName)
	}
}

func readAgentJSONL(ctx context.Context, f *os.File, path, sourceName string) ([]events.Event, error) {
	var out []events.Event
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	lineNo := 0
	for scanner.Scan() {
		if ctx.Err() != nil {
			return out, ctx.Err()
		}
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		evt, ok := sessionEventFromJSON([]byte(line), path, sourceName, lineNo)
		if !ok {
			continue
		}
		out = append(out, evt)
	}
	return out, scanner.Err()
}

func readAgentJSON(f *os.File, path, sourceName string) ([]events.Event, error) {
	data, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}
	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		return nil, nil
	}

	var many []json.RawMessage
	if err := json.Unmarshal(data, &many); err == nil {
		var out []events.Event
		for i, raw := range many {
			evt, ok := sessionEventFromJSON(raw, path, sourceName, i+1)
			if ok {
				out = append(out, evt)
			}
		}
		return out, nil
	}

	evt, ok := sessionEventFromJSON(data, path, sourceName, 1)
	if !ok {
		return nil, nil
	}
	return []events.Event{evt}, nil
}

func sessionEventFromJSON(raw []byte, path, sourceName string, index int) (events.Event, bool) {
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil {
		return events.Event{}, false
	}
	if len(obj) == 0 {
		return events.Event{}, false
	}

	id := stringField(obj, "id")
	summary := stringField(obj, "summary")
	if summary == "" {
		summary = stringField(obj, "title")
	}
	projectPath := stringField(obj, "project_path")
	if projectPath == "" {
		projectPath = stringField(obj, "projectPath")
	}
	started := timeField(obj, "started_at", "startedAt")
	ended := timeField(obj, "ended_at", "endedAt")
	if started.IsZero() {
		started = time.Now().UTC()
	}
	if ended.IsZero() {
		ended = started
	}
	if id == "" {
		id = fmt.Sprintf("%s:%d", filepath.Base(path), index)
	}

	payload, _ := json.Marshal(obj)
	return events.Normalize(events.Event{
		ID:          "agent.session:" + sourceName + ":" + id,
		StartedAt:   started,
		EndedAt:     ended,
		ProjectPath: projectPath,
		Directory:   projectPath,
		Type:        "agent.session",
		Summary:     summary,
		Payload:     payload,
		Source:      "agent",
	}), true
}

func stringField(obj map[string]any, key string) string {
	v, ok := obj[key]
	if !ok || v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	default:
		return strings.TrimSpace(fmt.Sprint(t))
	}
}

func timeField(obj map[string]any, keys ...string) time.Time {
	for _, key := range keys {
		v, ok := obj[key]
		if !ok || v == nil {
			continue
		}
		switch t := v.(type) {
		case string:
			if ts, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(t)); err == nil {
				return ts.UTC()
			}
			if ts, err := time.Parse(time.RFC3339, strings.TrimSpace(t)); err == nil {
				return ts.UTC()
			}
		case float64:
			return time.Unix(int64(t), 0).UTC()
		}
	}
	return time.Time{}
}
