package rules

import (
	"fmt"
	"log/slog"
	"net"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

type Operator string

const (
	OpEquals   Operator = "equals"
	OpContains Operator = "contains"
	OpRegex    Operator = "regex"
	OpPrefix   Operator = "prefix"
	OpSuffix   Operator = "suffix"
	OpIPMatch  Operator = "ip_match"
	OpGeoMatch Operator = "geo_match"
	OpExists   Operator = "exists"
	OpGt       Operator = "gt"
	OpLt       Operator = "lt"
	OpIn       Operator = "in"
	OpNotIn    Operator = "not_in"
)

type Action string

const (
	ActionBlock     Action = "block"
	ActionAllow     Action = "allow"
	ActionChallenge Action = "challenge"
	ActionMonitor   Action = "monitor"
	ActionRateLimit Action = "rate_limit"
)

type Phase string

const (
	PhaseRequest  Phase = "request"
	PhaseResponse Phase = "response"
	PhaseConnect  Phase = "connect"
)

type Rule struct {
	ID          string                 `yaml:"id"`
	Name        string                 `yaml:"name"`
	Description string                 `yaml:"description"`
	Enabled     bool                   `yaml:"enabled"`
	Severity    string                 `yaml:"severity"`
	Action      string                 `yaml:"action"`
	Phase       string                 `yaml:"phase"`
	Priority    int                    `yaml:"priority"`
	Field       string                 `yaml:"field"`
	Operator    string                 `yaml:"operator"`
	Value       string                 `yaml:"value"`
	Transform   []string               `yaml:"transform"`
	Tags        []string               `yaml:"tags"`
	Params      map[string]interface{} `yaml:"params,omitempty"`
	compiledRE  *regexp.Regexp
}

type FieldValue struct {
	QueryParams map[string][]string
	FormParams  map[string][]string
	Headers     map[string]string
	Cookies     map[string]string
	Body        []byte
	Path        string
	Method      string
	RealIP      string
	UserAgent   string
	ContentType string
	Country     string
	ASN         uint
	SessionID   string
	UserID      string
	Host        string
}

type Decision struct {
	RuleID   string
	RuleName string
	Action   string
	Severity string
	Score    float64
	Evidence string
	Matched  bool
}

type Engine struct {
	mu     sync.RWMutex
	rules  []Rule
	byID   map[string]*Rule
	dryRun bool
}

func NewEngine(dryRun bool) *Engine {
	return &Engine{
		rules:  make([]Rule, 0),
		byID:   make(map[string]*Rule),
		dryRun: dryRun,
	}
}

func (e *Engine) Name() string { return "rules" }

func (e *Engine) Inspect(field *FieldValue) (*Decision, error) {
	e.mu.RLock()
	rules := make([]Rule, len(e.rules))
	copy(rules, e.rules)
	e.mu.RUnlock()

	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}

		matched, err := e.evaluateRule(&rule, field)
		if err != nil {
			slog.Error("rule evaluation error", "rule_id", rule.ID, "error", err)
			continue
		}

		if matched {
			if e.dryRun {
				slog.Info("dry-run: rule would match",
					"rule_id", rule.ID,
					"rule_name", rule.Name,
					"action", rule.Action,
				)
				continue
			}

			return &Decision{
				RuleID:   rule.ID,
				RuleName: rule.Name,
				Action:   rule.Action,
				Severity: rule.Severity,
				Score:    e.severityToScore(rule.Severity),
				Evidence: fmt.Sprintf("rule %s matched: %s %s %s %s", rule.ID, rule.Field, rule.Operator, rule.Value, field.Path),
				Matched:  true,
			}, nil
		}
	}

	return &Decision{Matched: false, Action: "allow"}, nil
}

func (e *Engine) evaluateRule(rule *Rule, field *FieldValue) (bool, error) {
	value := e.extractField(rule.Field, field)
	if value == "" {
		return false, nil
	}

	transformed := applyTransforms(value, rule.Transform)
	if transformed == "" {
		return false, nil
	}

	return e.applyOperator(rule.Operator, transformed, rule.Value, rule)
}

func (e *Engine) extractField(fieldName string, field *FieldValue) string {
	parts := strings.SplitN(fieldName, ".", 2)
	if len(parts) < 2 {
		return ""
	}

	category := parts[0]
	key := parts[1]

	switch category {
	case "headers":
		if v, ok := field.Headers[key]; ok {
			return v
		}
		for k, v := range field.Headers {
			if strings.EqualFold(k, key) {
				return v
			}
		}
	case "query":
		if v, ok := field.QueryParams[key]; ok && len(v) > 0 {
			return v[0]
		}
	case "form":
		if v, ok := field.FormParams[key]; ok && len(v) > 0 {
			return v[0]
		}
	case "cookies":
		if v, ok := field.Cookies[key]; ok {
			return v
		}
	case "path":
		return field.Path
	case "method":
		return field.Method
	case "ip":
		return field.RealIP
	case "user_agent":
		return field.UserAgent
	case "content_type":
		return field.ContentType
	case "country":
		return field.Country
	case "asn":
		return fmt.Sprintf("%d", field.ASN)
	case "session":
		return field.SessionID
	case "user_id":
		return field.UserID
	case "host":
		return field.Host
	case "body":
		if field.Body != nil {
			return string(field.Body)
		}
	}

	return ""
}

func applyTransforms(value string, transforms []string) string {
	result := value
	for _, t := range transforms {
		switch strings.ToLower(t) {
		case "lowercase":
			result = strings.ToLower(result)
		case "uppercase":
			result = strings.ToUpper(result)
		case "trim":
			result = strings.TrimSpace(result)
		case "urldecode":
			result = strings.ReplaceAll(result, "%25", "%")
			result = strings.ReplaceAll(result, "%20", " ")
			result = strings.ReplaceAll(result, "%3C", "<")
			result = strings.ReplaceAll(result, "%3E", ">")
			result = strings.ReplaceAll(result, "%27", "'")
			result = strings.ReplaceAll(result, "%22", "\"")
			result = strings.ReplaceAll(result, "%3B", ";")
			result = strings.ReplaceAll(result, "%2F", "/")
			result = strings.ReplaceAll(result, "%3D", "=")
			result = strings.ReplaceAll(result, "%26", "&")
			result = strings.ReplaceAll(result, "%23", "#")
			result = strings.ReplaceAll(result, "%3F", "?")
		case "remove_null":
			result = strings.ReplaceAll(result, "\x00", "")
		case "remove_comments":
			result = strings.ReplaceAll(result, "/*", " ")
			result = strings.ReplaceAll(result, "*/", " ")
			result = strings.ReplaceAll(result, "--", " ")
		case "normalize_path":
			result = strings.ReplaceAll(result, "//", "/")
			result = strings.ReplaceAll(result, "/./", "/")
		case "compress_whitespace":
			re := regexp.MustCompile(`\s+`)
			result = re.ReplaceAllString(result, " ")
		}
	}
	return result
}

func (e *Engine) applyOperator(op, fieldValue, ruleValue string, rule *Rule) (bool, error) {
	switch Operator(op) {
	case OpEquals:
		return fieldValue == ruleValue, nil

	case OpContains:
		return strings.Contains(fieldValue, ruleValue), nil

	case OpRegex:
		if rule.compiledRE != nil {
			return rule.compiledRE.MatchString(fieldValue), nil
		}
		re, err := regexp.Compile(ruleValue)
		if err != nil {
			return false, fmt.Errorf("invalid regex %q: %w", ruleValue, err)
		}
		rule.compiledRE = re
		return re.MatchString(fieldValue), nil

	case OpPrefix:
		return strings.HasPrefix(fieldValue, ruleValue), nil

	case OpSuffix:
		return strings.HasSuffix(fieldValue, ruleValue), nil

	case OpIPMatch:
		ip := net.ParseIP(fieldValue)
		if ip == nil {
			return false, nil
		}
		_, cidr, err := net.ParseCIDR(ruleValue)
		if err != nil {
			return ip.String() == ruleValue, nil
		}
		return cidr.Contains(ip), nil

	case OpGeoMatch:
		return strings.EqualFold(fieldValue, ruleValue), nil

	case OpExists:
		return fieldValue != "", nil

	case OpGt:
		var fv, rv float64
		fmt.Sscanf(fieldValue, "%f", &fv)
		fmt.Sscanf(ruleValue, "%f", &rv)
		return fv > rv, nil

	case OpLt:
		var fv, rv float64
		fmt.Sscanf(fieldValue, "%f", &fv)
		fmt.Sscanf(ruleValue, "%f", &rv)
		return fv < rv, nil

	case OpIn:
		parts := strings.Split(ruleValue, ",")
		for _, p := range parts {
			if strings.TrimSpace(p) == fieldValue {
				return true, nil
			}
		}
		return false, nil

	case OpNotIn:
		parts := strings.Split(ruleValue, ",")
		for _, p := range parts {
			if strings.TrimSpace(p) == fieldValue {
				return false, nil
			}
		}
		return true, nil
	}

	return false, nil
}

func (e *Engine) severityToScore(severity string) float64 {
	switch severity {
	case "critical":
		return 95
	case "high":
		return 75
	case "medium":
		return 50
	case "low":
		return 25
	case "info":
		return 5
	default:
		return 50
	}
}

func (e *Engine) LoadRules(rules []Rule) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.rules = make([]Rule, len(rules))
	e.byID = make(map[string]*Rule, len(rules))

	for i, rule := range rules {
		if rule.Operator == "regex" || rule.Operator == "regex_match" {
			if re, err := regexp.Compile(rule.Value); err == nil {
				rule.compiledRE = re
			}
		}
		e.rules[i] = rule
		e.byID[rule.ID] = &e.rules[i]
	}

	e.sortRules()
}

func (e *Engine) sortRules() {
	sort.Slice(e.rules, func(i, j int) bool {
		if e.rules[i].Priority != e.rules[j].Priority {
			return e.rules[i].Priority < e.rules[j].Priority
		}
		return e.rules[i].ID < e.rules[j].ID
	})
}

func (e *Engine) AtomicSwap(rules []Rule) {
	newEngine := NewEngine(e.dryRun)
	newEngine.LoadRules(rules)

	e.mu.Lock()
	e.rules = newEngine.rules
	e.byID = newEngine.byID
	e.mu.Unlock()
}

func (e *Engine) GetRule(id string) *Rule {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.byID[id]
}

func (e *Engine) GetAllRules() []Rule {
	e.mu.RLock()
	defer e.mu.RUnlock()
	result := make([]Rule, len(e.rules))
	copy(result, e.rules)
	return result
}

func (e *Engine) GetEnabledRules() []Rule {
	e.mu.RLock()
	defer e.mu.RUnlock()
	var result []Rule
	for _, r := range e.rules {
		if r.Enabled {
			result = append(result, r)
		}
	}
	return result
}

func (e *Engine) AddRule(rule Rule) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, exists := e.byID[rule.ID]; exists {
		return fmt.Errorf("rule %s already exists", rule.ID)
	}

	if rule.Operator == "regex" || rule.Operator == "regex_match" {
		if re, err := regexp.Compile(rule.Value); err == nil {
			rule.compiledRE = re
		} else {
			return fmt.Errorf("invalid regex: %w", err)
		}
	}

	e.rules = append(e.rules, rule)
	e.byID[rule.ID] = &e.rules[len(e.rules)-1]
	e.sortRules()

	return nil
}

func (e *Engine) UpdateRule(id string, updated Rule) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	rule, exists := e.byID[id]
	if !exists {
		return fmt.Errorf("rule %s not found", id)
	}

	*rule = updated
	if rule.Operator == "regex" || rule.Operator == "regex_match" {
		if re, err := regexp.Compile(rule.Value); err == nil {
			rule.compiledRE = re
		}
	}
	e.sortRules()

	return nil
}

func (e *Engine) DeleteRule(id string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, exists := e.byID[id]; !exists {
		return fmt.Errorf("rule %s not found", id)
	}

	delete(e.byID, id)
	filtered := make([]Rule, 0, len(e.rules))
	for _, r := range e.rules {
		if r.ID != id {
			filtered = append(filtered, r)
		}
	}
	e.rules = filtered

	return nil
}

func (e *Engine) ReorderRule(id string, priority int) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	rule, exists := e.byID[id]
	if !exists {
		return fmt.Errorf("rule %s not found", id)
	}

	rule.Priority = priority
	e.sortRules()

	return nil
}

func ParseRulesYAML(data []byte) ([]Rule, error) {
	var rules []Rule
	if err := yaml.Unmarshal(data, &rules); err != nil {
		return nil, fmt.Errorf("parse rules yaml: %w", err)
	}
	return rules, nil
}

func (e *Engine) TestRule(rule Rule, field *FieldValue) (*Decision, error) {
	matched, err := e.evaluateRule(&rule, field)
	if err != nil {
		return nil, err
	}

	return &Decision{
		RuleID:   rule.ID,
		RuleName: rule.Name,
		Action:   rule.Action,
		Severity: rule.Severity,
		Score:    e.severityToScore(rule.Severity),
		Matched:  matched,
		Evidence: fmt.Sprintf("test: rule %s evaluated: matched=%v", rule.ID, matched),
	}, nil
}

func DefaultRules() []Rule {
	return []Rule{
		{
			ID: "PLATFORM001", Name: "Block Common Attack Tools", Enabled: true,
			Severity: "high", Action: "block", Phase: "request", Priority: 1,
			Field: "user_agent", Operator: "regex",
			Value: `(?i)(?:sqlmap|nikto|nmap|masscan|gobuster|dirbuster|wpscan|burpsuite|zap|acunetix|netsparker)`,
			Tags:  []string{"automation", "scanner"},
		},
		{
			ID: "PLATFORM002", Name: "Block Path Traversal", Enabled: true,
			Severity: "critical", Action: "block", Phase: "request", Priority: 2,
			Field: "path", Operator: "regex",
			Value: `(?i)(?:/\.\.|\.\./|\.\.\\|/etc/passwd|/windows/win\.ini)`,
			Tags:  []string{"lfi", "path-traversal"},
		},
		{
			ID: "PLATFORM003", Name: "Block Private IP Ranges", Enabled: true,
			Severity: "high", Action: "block", Phase: "request", Priority: 3,
			Field: "headers.X-Forwarded-For", Operator: "regex",
			Value: `(?i)(?:^127\.|^10\.|^172\.(?:1[6-9]|2[0-9]|3[01])\.|^192\.168\.)`,
			Tags:  []string{"spoofing", "security"},
		},
		{
			ID: "PLATFORM004", Name: "Block Suspicious Methods", Enabled: true,
			Severity: "high", Action: "block", Phase: "request", Priority: 4,
			Field: "method", Operator: "in",
			Value: "CONNECT,TRACE,TRACK,PUT,DELETE,PATCH",
			Tags:  []string{"method", "restriction"},
		},
	}
}

func (e *Engine) InspectEngine(field *FieldValue) (*Decision, error) {
	return e.Inspect(field)
}

var (
	_ = yaml.Marshal
	_ = time.Second
	_ = slog.Debug
)
