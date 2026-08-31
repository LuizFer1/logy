package events

import (
	"encoding/json"
	"regexp"
	"strings"
)

type RedactionRules struct {
	ExcludeGlobs []string
	Mask         string
}

var redactionKeyPattern = regexp.MustCompile(`(?i)\b(token|password|api_key)\b\s*[:=]\s*("[^"]*"|'[^']*'|[^\s,;]+)`)

func Redact(event Event, rules RedactionRules) Event {
	normalized := Normalize(event)
	if matchesAnyGlob(rules.ExcludeGlobs, normalized.ProjectPath, normalized.Directory) {
		return normalized
	}

	mask := rules.Mask
	if mask == "" {
		mask = "[REDACTED]"
	}

	redacted := normalized
	redacted.Summary = redactSummary(redacted.Summary, mask)
	payload, changed := redactPayload(redacted.Payload, mask)
	if changed {
		redacted.Payload = payload
	}
	if redacted.Summary != normalized.Summary || changed {
		redacted.Sensitivity = SensitivityRedacted
	}

	return redacted
}

func redactSummary(summary, mask string) string {
	if summary == "" {
		return ""
	}

	return redactionKeyPattern.ReplaceAllStringFunc(summary, func(match string) string {
		parts := redactionKeyPattern.FindStringSubmatch(match)
		if len(parts) < 2 {
			return match
		}

		return parts[1] + "=" + mask
	})
}

func redactPayload(payload json.RawMessage, mask string) (json.RawMessage, bool) {
	trimmed := strings.TrimSpace(string(payload))
	if trimmed == "" {
		return nil, false
	}

	var decoded any
	if err := json.Unmarshal([]byte(trimmed), &decoded); err != nil {
		return payload, false
	}

	redacted, changed := redactJSONValue(decoded, mask)
	if !changed {
		return payload, false
	}

	encoded, err := json.Marshal(redacted)
	if err != nil {
		return payload, false
	}

	return json.RawMessage(encoded), true
}

func redactJSONValue(value any, mask string) (any, bool) {
	switch typed := value.(type) {
	case map[string]any:
		changed := false
		redacted := make(map[string]any, len(typed))
		for key, child := range typed {
			if isSensitiveKey(key) {
				redacted[key] = mask
				if !changed {
					changed = true
				}
				continue
			}

			redactedChild, childChanged := redactJSONValue(child, mask)
			redacted[key] = redactedChild
			if childChanged {
				changed = true
			}
		}

		return redacted, changed
	case []any:
		changed := false
		redacted := make([]any, len(typed))
		for index, child := range typed {
			redactedChild, childChanged := redactJSONValue(child, mask)
			redacted[index] = redactedChild
			if childChanged {
				changed = true
			}
		}

		return redacted, changed
	default:
		return value, false
	}
}

func isSensitiveKey(key string) bool {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "token", "password", "api_key":
		return true
	default:
		return false
	}
}
