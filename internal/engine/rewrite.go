package engine

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
)

type RewriteManager struct {
	rules []RewriteRule
}

type RewriteRule struct {
	Name       string
	Conditions []RewriteCondition
	Actions    []RewriteAction
}

type RewriteCondition struct {
	Field    string
	Name     string
	Operator string
	Value    string
}

type RewriteAction interface {
	ApplyRequest(ctx *RequestContext) error
	ApplyResponse(ctx *ResponseContext) error
}

type HeaderAction struct {
	Operation string
	Name      string
	Value     string
}

func (a *HeaderAction) ApplyRequest(ctx *RequestContext) error {
	switch a.Operation {
	case "add", "set":
		ctx.Headers[a.Name] = a.Value
	case "remove":
		delete(ctx.Headers, a.Name)
	case "rename":
		if v, ok := ctx.Headers[a.Name]; ok {
			delete(ctx.Headers, a.Name)
			ctx.Headers[a.Value] = v
		}
	}
	return nil
}

func (a *HeaderAction) ApplyResponse(ctx *ResponseContext) error {
	switch a.Operation {
	case "add", "set":
		ctx.Headers[a.Name] = a.Value
	case "remove":
		delete(ctx.Headers, a.Name)
	}
	return nil
}

type BodyAction struct {
	Operation string
	Pattern   string
	Value     string
	Regex     *regexp.Regexp
}

func NewBodyAction(op, pattern, value string) (*BodyAction, error) {
	a := &BodyAction{
		Operation: op,
		Pattern:   pattern,
		Value:     value,
	}
	if op == "regex_replace" {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid regex: %w", err)
		}
		a.Regex = re
	}
	return a, nil
}

func (a *BodyAction) ApplyRequest(ctx *RequestContext) error {
	switch a.Operation {
	case "replace":
		ctx.Body = bytes.ReplaceAll(ctx.Body, []byte(a.Pattern), []byte(a.Value))
	case "regex_replace":
		if a.Regex != nil {
			ctx.Body = a.Regex.ReplaceAll(ctx.Body, []byte(a.Value))
		}
	}
	return nil
}

func (a *BodyAction) ApplyResponse(ctx *ResponseContext) error {
	switch a.Operation {
	case "replace":
		ctx.Body = bytes.ReplaceAll(ctx.Body, []byte(a.Pattern), []byte(a.Value))
	case "regex_replace":
		if a.Regex != nil {
			ctx.Body = a.Regex.ReplaceAll(ctx.Body, []byte(a.Value))
		}
	}
	return nil
}

type URLAction struct {
	Operation string
	URL       string
	Code      int
}

func (a *URLAction) ApplyRequest(ctx *RequestContext) error {
	if a.Operation == "redirect" {
		ctx.Headers["Location"] = a.expandURL(ctx)
	}
	return nil
}

func (a *URLAction) ApplyResponse(ctx *ResponseContext) error {
	return nil
}

func (a *URLAction) expandURL(ctx *RequestContext) string {
	url := a.URL
	url = strings.ReplaceAll(url, "{{.path}}", ctx.Path)
	url = strings.ReplaceAll(url, "{{.ip}}", ctx.RealIP)
	url = strings.ReplaceAll(url, "{{.host}}", ctx.Host)
	url = strings.ReplaceAll(url, "{{.method}}", ctx.Method)
	return url
}

func NewRewriteManager() *RewriteManager {
	return &RewriteManager{
		rules: make([]RewriteRule, 0),
	}
}

func (m *RewriteManager) AddRule(rule RewriteRule) {
	m.rules = append(m.rules, rule)
}

func (m *RewriteManager) ApplyRequest(ctx *RequestContext) error {
	for _, rule := range m.rules {
		if !m.matchConditions(rule.Conditions, ctx) {
			continue
		}
		for _, action := range rule.Actions {
			if err := action.ApplyRequest(ctx); err != nil {
				return err
			}
		}
	}
	return nil
}

func (m *RewriteManager) ApplyResponse(ctx *ResponseContext) error {
	for _, rule := range m.rules {
		for _, action := range rule.Actions {
			if err := action.ApplyResponse(ctx); err != nil {
				return err
			}
		}
	}
	return nil
}

func (m *RewriteManager) matchConditions(conditions []RewriteCondition, ctx *RequestContext) bool {
	for _, cond := range conditions {
		if !m.matchCondition(cond, ctx) {
			return false
		}
	}
	return true
}

func (m *RewriteManager) matchCondition(cond RewriteCondition, ctx *RequestContext) bool {
	switch cond.Field {
	case "path":
		return matchValue(ctx.Path, cond)
	case "headers":
		if v, ok := ctx.Headers[cond.Name]; ok {
			return matchValue(v, cond)
		}
		return cond.Operator == "not_exists" || cond.Operator == "!exists"
	case "query":
		if v, ok := ctx.QueryParams[cond.Name]; ok && len(v) > 0 {
			return matchValue(v[0], cond)
		}
		return cond.Operator == "not_exists"
	case "method":
		return matchValue(ctx.Method, cond)
	case "ip":
		return matchValue(ctx.RealIP, cond)
	default:
		return false
	}
}

func matchValue(value string, cond RewriteCondition) bool {
	switch cond.Operator {
	case "equals":
		return value == cond.Value
	case "contains":
		return strings.Contains(value, cond.Value)
	case "prefix":
		return strings.HasPrefix(value, cond.Value)
	case "suffix":
		return strings.HasSuffix(value, cond.Value)
	case "regex":
		re, err := regexp.Compile(cond.Value)
		if err != nil {
			return false
		}
		return re.MatchString(value)
	case "exists":
		return value != ""
	case "not_exists", "!exists":
		return value == ""
	default:
		return false
	}
}
