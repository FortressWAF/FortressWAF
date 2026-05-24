package compliance

import (
	"regexp"
	"sync"
)

type PIIMasker struct {
	mu       sync.RWMutex
	patterns map[string]*regexp.Regexp
	enabled  bool
}

func NewPIIMasker() *PIIMasker {
	pm := &PIIMasker{
		patterns: make(map[string]*regexp.Regexp),
		enabled:  true,
	}
	pm.initPatterns()
	return pm
}

func (pm *PIIMasker) initPatterns() {
	pm.patterns["email"] = regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)
	pm.patterns["credit_card"] = regexp.MustCompile(`\b(?:\d[ -]*?){13,16}\b`)
	pm.patterns["ssn"] = regexp.MustCompile(`\b\d{3}[- ]?\d{2}[- ]?\d{4}\b`)
	pm.patterns["phone"] = regexp.MustCompile(`\b(?:\+?1[-.\s]?)?\(?\d{3}\)?[-.\s]?\d{3}[-.\s]?\d{4}\b`)
	pm.patterns["ip"] = regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b`)
	pm.patterns["api_key"] = regexp.MustCompile(`(?i)(?:api[_-]?key|apikey|secret[_-]?key)['":\s=]+[a-zA-Z0-9_\-]{20,}`)
	pm.patterns["password"] = regexp.MustCompile(`(?i)(?:password|passwd|pwd)['":\s=]+[^\s]{6,}`)
	pm.patterns["jwt"] = regexp.MustCompile(`eyJ[a-zA-Z0-9_-]*\.eyJ[a-zA-Z0-9_-]*\.[a-zA-Z0-9_-]*`)
}

func (pm *PIIMasker) Mask(data string) string {
	if !pm.enabled {
		return data
	}

	result := data
	for name, pattern := range pm.patterns {
		result = pattern.ReplaceAllStringFunc(result, func(match string) string {
			return pm.maskValue(name, match)
		})
	}
	return result
}

func (pm *PIIMasker) maskValue(piiType, value string) string {
	switch piiType {
	case "email":
		parts := regexp.MustCompile(`@`).Split(value, 2)
		if len(parts) == 2 {
			return parts[0][:1] + "***@" + parts[1]
		}
		return value[:1] + "***"
	case "credit_card":
		return "****-****-****-" + value[len(value)-4:]
	case "ssn":
		return "***-**-" + value[len(value)-4:]
	case "phone":
		return "***-***-" + value[len(value)-4:]
	case "ip":
		return value[:len(value)-4] + "****"
	case "api_key", "password":
		return "[REDACTED]"
	case "jwt":
		return "eyJ***.[REDACTED].***"
	default:
		return "***"
	}
}

func (pm *PIIMasker) Detect(data string) map[string][]string {
	results := make(map[string][]string)

	for name, pattern := range pm.patterns {
		matches := pattern.FindAllString(data, -1)
		if len(matches) > 0 {
			results[name] = matches
		}
	}

	return results
}

func (pm *PIIMasker) Enable() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.enabled = true
}

func (pm *PIIMasker) Disable() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.enabled = false
}
