package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/FortressWAF/FortressWAF/internal/engine"
)

type DryRunMode struct {
	Enabled       bool
	LogFile       string
	OutputFormat  string
	IncludeBody  bool
	Silent       bool
	file         *os.File
	encoder      *json.Encoder
}

type DryRunLog struct {
	Timestamp    string          `json:"timestamp"`
	Method       string          `json:"method"`
	Path         string          `json:"path"`
	Host         string          `json:"host"`
	ClientIP     string          `json:"client_ip"`
	UserAgent    string          `json:"user_agent"`
	Headers      map[string]string `json:"headers"`
	Body         string          `json:"body,omitempty"`
	MatchedRules []MatchedRule   `json:"matched_rules"`
	FinalDecision string        `json:"final_decision"`
	ThreatScore  float64         `json:"threat_score"`
	WhatIf       WhatIfDecision  `json:"what_if"`
}

type MatchedRule struct {
	RuleID   string  `json:"rule_id"`
	RuleName string  `json:"rule_name"`
	Severity string  `json:"severity"`
	Score    float64 `json:"score"`
	Action   string  `json:"action"`
	Evidence string  `json:"evidence"`
}

type WhatIfDecision struct {
	WouldBlock   bool     `json:"would_block"`
	WouldAllow   bool     `json:"would_allow"`
	WouldChallenge bool   `json:"would_challenge"`
	WouldRateLimit bool   `json:"would_rate_limit"`
	Reason       string   `json:"reason"`
	Score        float64  `json:"score"`
}

func NewDryRunMode(cfg DryRunConfig) (*DryRunMode, error) {
	d := &DryRunMode{
		Enabled:      cfg.Enabled,
		LogFile:      cfg.LogFile,
		OutputFormat: cfg.OutputFormat,
		IncludeBody:  cfg.IncludeBody,
		Silent:       cfg.Silent,
	}

	if d.OutputFormat == "" {
		d.OutputFormat = "json"
	}

	if d.Enabled && d.LogFile != "" {
		f, err := os.OpenFile(d.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, fmt.Errorf("open dryrun log: %w", err)
		}
		d.file = f
		d.encoder = json.NewEncoder(f)
	}

	return d, nil
}

type DryRunConfig struct {
	Enabled      bool
	LogFile      string
	OutputFormat string
	IncludeBody  bool
	Silent       bool
}

func (d *DryRunMode) LogRequest(r *DryRunLog) error {
	if !d.Enabled {
		return nil
	}

	r.Timestamp = time.Now().UTC().Format(time.RFC3339)

	whatIf := d.calculateWhatIf(r)
	r.WhatIf = whatIf

	if d.file != nil {
		if err := d.encoder.Encode(r); err != nil {
			return fmt.Errorf("encode dryrun log: %w", err)
		}
	}

	if !d.Silent {
		slog.Info("dryrun request",
			"method", r.Method,
			"path", r.Path,
			"client_ip", r.ClientIP,
			"would_block", whatIf.WouldBlock,
			"score", whatIf.Score,
		)
	}

	return nil
}

func (d *DryRunMode) calculateWhatIf(r *DryRunLog) WhatIfDecision {
	dec := WhatIfDecision{
		WouldBlock:     false,
		WouldAllow:     true,
		WouldChallenge: false,
		WouldRateLimit: false,
	}

	score := r.ThreatScore
	for _, rule := range r.MatchedRules {
		score += rule.Score
	}

	dec.Score = score

	if score >= 90 {
		dec.WouldBlock = true
		dec.WouldAllow = false
		dec.Reason = "cumulative threat score >= 90"
	} else if score >= 50 {
		dec.WouldChallenge = true
		dec.WouldAllow = false
		dec.Reason = "threat score >= 50, would issue challenge"
	} else {
		dec.WouldAllow = true
		dec.Reason = "request would be allowed"
	}

	return dec
}

func (d *DryRunMode) Close() error {
	if d.file != nil {
		return d.file.Close()
	}
	return nil
}

func LogDecision(ctx *engine.RequestContext, decision *engine.Decision, rules []engine.Decision) error {
	cfg := GetDryRunConfig()
	if cfg == nil || !cfg.Enabled {
		return nil
	}

	dryrun := &DryRunLog{
		Method:       ctx.Method,
		Path:         ctx.Path,
		Host:         ctx.Headers["Host"],
		ClientIP:     ctx.RealIP,
		UserAgent:    ctx.UserAgent,
		Headers:      ctx.Headers,
		FinalDecision: string(decision.Action),
		ThreatScore:  ctx.ThreatScore,
	}

	if cfg.IncludeBody && len(ctx.Body) > 0 {
		dryrun.Body = string(ctx.Body)
	}

	for _, r := range rules {
		dryrun.MatchedRules = append(dryrun.MatchedRules, MatchedRule{
			RuleID:   r.RuleID,
			RuleName: r.RuleName,
			Severity: r.Severity,
			Score:    r.Score,
			Action:   string(r.Action),
			Evidence: r.Evidence,
		})
	}

	return dryrun.Log()
}

func (d *DryRunLog) Log() error {
	return nil
}

var dryRunConfig *DryRunConfig

func GetDryRunConfig() *DryRunConfig {
	return dryRunConfig
}

func SetDryRunConfig(cfg *DryRunConfig) {
	dryRunConfig = cfg
}

var _ = slog.Info
