package diagnostics

import (
	"encoding/json"
	"net/url"
	"regexp"
	"strings"
)

const redacted = "[REDACTED]"

var (
	bearerPattern     = regexp.MustCompile(`(?i)\bbearer\s+[a-z0-9._~+/=-]+`)
	jwtPattern        = regexp.MustCompile(`\b[A-Za-z0-9_-]{12,}\.[A-Za-z0-9_-]{12,}\.[A-Za-z0-9_-]{12,}\b`)
	keyPattern        = regexp.MustCompile(`(?i)\b(sk-[a-z0-9_-]{12,}|AKIA[A-Z0-9]{16})\b`)
	assignmentPattern = regexp.MustCompile(`(?i)(password|passwd|secret|token|api[_-]?key|access[_-]?key|authorization|cookie|csrf)=([^\s&;]+)`)
)

func RedactMap(input map[string]any) map[string]any {
	if input == nil {
		return nil
	}
	encoded, err := json.Marshal(input)
	if err != nil {
		return map[string]any{"redaction_error": "value could not be represented safely"}
	}
	var value map[string]any
	if err := json.Unmarshal(encoded, &value); err != nil {
		return map[string]any{"redaction_error": "value could not be represented safely"}
	}
	return redactObject(value)
}

func redactObject(input map[string]any) map[string]any {
	output := make(map[string]any, len(input))
	for key, value := range input {
		if sensitiveKey(key) {
			output[key] = redacted
			continue
		}
		output[key] = redactValue(value)
	}
	return output
}

func redactValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return redactObject(typed)
	case []any:
		result := make([]any, len(typed))
		for i := range typed {
			result[i] = redactValue(typed[i])
		}
		return result
	case string:
		return redactString(typed)
	default:
		return value
	}
}

func sensitiveKey(key string) bool {
	normalized := strings.ToLower(strings.NewReplacer("-", "_", " ", "_").Replace(key))
	for _, fragment := range []string{
		"password", "passwd", "secret", "token", "credential", "authorization",
		"cookie", "csrf", "api_key", "apikey", "access_key", "private_key", "dsn",
		"connection_string",
	} {
		if strings.Contains(normalized, fragment) {
			return true
		}
	}
	return false
}

func redactString(value string) string {
	value = bearerPattern.ReplaceAllString(value, "Bearer "+redacted)
	value = jwtPattern.ReplaceAllString(value, redacted)
	value = keyPattern.ReplaceAllString(value, redacted)
	value = assignmentPattern.ReplaceAllString(value, "$1="+redacted)
	if parsed, err := url.Parse(value); err == nil && parsed.Scheme != "" && parsed.Host != "" {
		if parsed.User != nil {
			parsed.User = url.User(redacted)
		}
		if parsed.RawQuery != "" {
			parsed.RawQuery = "redacted=" + url.QueryEscape(redacted)
		}
		value = parsed.String()
	}
	return value
}
