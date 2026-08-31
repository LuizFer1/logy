package report

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode"

	"logy/internal/events"
)

type Answer struct {
	Text          string
	Evidence      []Evidence
	Deterministic bool
}

var (
	projetoRe = regexp.MustCompile(`(?i)\bprojeto\s+([^\s?.,!;:]+)`)
	pathRe    = regexp.MustCompile(`[A-Za-z]:\\[^\s?.,!;:]+|/[^\s?.,!;:]+`)
)

// Ask interprets a natural-language question and returns a deterministic answer.
// It never executes commands or modifies files.
func Ask(ctx context.Context, s Searcher, question string, now time.Time) (Answer, error) {
	from, to, _ := parsePeriod(question, now)
	projectPath, projectName := parseProject(question)

	filter := events.EventFilter{
		From:        from,
		To:          to,
		ProjectPath: projectPath,
	}

	evts, err := s.Search(ctx, filter)
	if err != nil {
		return Answer{}, err
	}

	if projectName != "" && projectPath == "" {
		evts = filterByProjectName(evts, projectName)
	}

	if len(evts) == 0 {
		return Answer{
			Text:          "Nenhum evento encontrado.",
			Evidence:      nil,
			Deterministic: true,
		}, nil
	}

	var b strings.Builder
	var evidence []Evidence
	for _, ev := range evts {
		ev = events.Normalize(ev)
		e := toEvidence(ev)
		evidence = append(evidence, e)
		fmt.Fprintf(&b, "%s %s [%s] %s\n",
			ev.StartedAt.Format(time.RFC3339),
			ev.Type,
			ev.ID,
			ev.Summary,
		)
	}

	return Answer{
		Text:          strings.TrimSpace(b.String()),
		Evidence:      evidence,
		Deterministic: true,
	}, nil
}

func parsePeriod(question string, now time.Time) (from, to time.Time, name string) {
	q := strings.ToLower(normalizeAccents(question))
	switch {
	case strings.Contains(q, "ontem"):
		y, m, d := now.Date()
		yesterday := time.Date(y, m, d, 12, 0, 0, 0, now.Location()).AddDate(0, 0, -1)
		from, to = PeriodToday(yesterday)
		return from, to, "ontem"
	case strings.Contains(q, "hoje"):
		from, to = PeriodToday(now)
		return from, to, "hoje"
	case strings.Contains(q, "semana"):
		from, to = PeriodWeek(now)
		return from, to, "semana"
	case strings.Contains(q, "mes") || strings.Contains(q, "mês"):
		// normalizeAccents already folded mês → mes, but keep both for safety
		from, to = PeriodMonth(now)
		return from, to, "mes"
	default:
		return time.Time{}, time.Time{}, ""
	}
}

func parseProject(question string) (path, name string) {
	if m := pathRe.FindString(question); m != "" {
		return filepath.Clean(m), ""
	}
	if m := projetoRe.FindStringSubmatch(question); len(m) == 2 {
		return "", strings.TrimSpace(m[1])
	}
	return "", ""
}

func filterByProjectName(evts []events.Event, name string) []events.Event {
	needle := strings.ToLower(strings.TrimSpace(name))
	if needle == "" {
		return evts
	}
	var out []events.Event
	for _, ev := range evts {
		p := events.Normalize(ev).ProjectPath
		base := strings.ToLower(filepath.Base(p))
		full := strings.ToLower(p)
		if base == needle || strings.Contains(full, needle) || strings.Contains(base, needle) {
			out = append(out, ev)
		}
	}
	if out == nil {
		return []events.Event{}
	}
	return out
}

func normalizeAccents(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch r {
		case 'á', 'à', 'ã', 'â', 'Á', 'À', 'Ã', 'Â':
			b.WriteRune('a')
		case 'é', 'ê', 'É', 'Ê':
			b.WriteRune('e')
		case 'í', 'Í':
			b.WriteRune('i')
		case 'ó', 'ô', 'õ', 'Ó', 'Ô', 'Õ':
			b.WriteRune('o')
		case 'ú', 'Ú':
			b.WriteRune('u')
		case 'ç', 'Ç':
			b.WriteRune('c')
		default:
			if unicode.IsUpper(r) {
				b.WriteRune(unicode.ToLower(r))
			} else {
				b.WriteRune(r)
			}
		}
	}
	return b.String()
}
