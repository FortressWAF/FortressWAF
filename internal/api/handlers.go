package api

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/FortressWAF/FortressWAF/internal/config"
	"github.com/FortressWAF/FortressWAF/internal/rules"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"gopkg.in/yaml.v3"
)

type Handlers struct {
	mu         sync.RWMutex
	configMgr  *config.Manager
	ruleEngine *rules.Engine
	startTime  time.Time
	version    string

	alerts        []alertRecord
	alertsMu      sync.RWMutex
	alertConfig   alertConfigRecord
	alertChannels []alertChannelRecord
	alertChMu     sync.RWMutex

	tenants   []tenantRecord
	tenantsMu sync.RWMutex

	patches   []patchRecord
	patchesMu sync.RWMutex

	logStore   []logEntry
	logStoreMu sync.RWMutex
	logCh      chan logEntry

	apiKeys    map[string]apiKeyRecord
	apiKeysMu  sync.RWMutex
	sessions   map[string]sessionRecord
	sessionsMu sync.RWMutex
}

func NewHandlers(cfgMgr *config.Manager, ruleEngine *rules.Engine, version string) *Handlers {
	h := &Handlers{
		configMgr:  cfgMgr,
		ruleEngine: ruleEngine,
		startTime:  time.Now(),
		version:    version,
		logCh:      make(chan logEntry, 10000),
		apiKeys:    make(map[string]apiKeyRecord),
		sessions:   make(map[string]sessionRecord),
		alertConfig: alertConfigRecord{
			Threshold:       100,
			IntervalSeconds: 60,
			Enabled:         true,
			Channels:        []string{},
			Rules:           []string{},
		},
	}
	go h.drainLogChannel()
	return h
}

func (h *Handlers) drainLogChannel() {
	for entry := range h.logCh {
		h.logStoreMu.Lock()
		h.logStore = append(h.logStore, entry)
		if len(h.logStore) > 100000 {
			h.logStore = h.logStore[len(h.logStore)-50000:]
		}
		h.logStoreMu.Unlock()
	}
}

// --- response helpers ---

type apiResponse struct {
	Data interface{} `json:"data,omitempty"`
	Meta *metaInfo   `json:"meta,omitempty"`
}

type metaInfo struct {
	Total int `json:"total"`
	Page  int `json:"page"`
}

type apiError struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeOK(w http.ResponseWriter, data interface{}) {
	writeJSON(w, http.StatusOK, apiResponse{Data: data})
}

func writeCreated(w http.ResponseWriter, data interface{}) {
	writeJSON(w, http.StatusCreated, apiResponse{Data: data})
}

func writeOKWithMeta(w http.ResponseWriter, data interface{}, total, page int) {
	writeJSON(w, http.StatusOK, apiResponse{Data: data, Meta: &metaInfo{Total: total, Page: page}})
}

func writeErr(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w, status, apiError{Error: msg, Code: code})
}

func writeBadRequest(w http.ResponseWriter, msg string) {
	writeErr(w, http.StatusBadRequest, "BAD_REQUEST", msg)
}

func writeUnauthorized(w http.ResponseWriter) {
	writeErr(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
}

func writeForbidden(w http.ResponseWriter) {
	writeErr(w, http.StatusForbidden, "FORBIDDEN", "access denied")
}

func writeNotFound(w http.ResponseWriter, msg string) {
	writeErr(w, http.StatusNotFound, "NOT_FOUND", msg)
}

func writeConflict(w http.ResponseWriter, msg string) {
	writeErr(w, http.StatusConflict, "CONFLICT", msg)
}

func writeInternalErr(w http.ResponseWriter) {
	writeErr(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
}

func readJSON(r *http.Request, v interface{}) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}

func readJSONStrict(r *http.Request, v interface{}) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

func getPathParam(r *http.Request, name string) string {
	return mux.Vars(r)[name]
}

func getQueryInt(r *http.Request, name string, def int) int {
	v := r.URL.Query().Get(name)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

// --- Sites ---

func (h *Handlers) ListSites(w http.ResponseWriter, r *http.Request) {
	cfg := h.configMgr.Get()
	h.mu.RLock()
	defer h.mu.RUnlock()

	sites := make([]config.SiteConfig, len(cfg.Sites))
	copy(sites, cfg.Sites)

	writeOKWithMeta(w, sites, len(sites), 1)
}

type createSiteRequest struct {
	Name        string   `json:"name"`
	Domains     []string `json:"domains"`
	OriginURL   string   `json:"origin_url"`
	Port        int      `json:"port"`
	RuleProfile string   `json:"rule_profile"`
	TLS         bool     `json:"tls"`
	WAFEnabled  bool     `json:"waf_enabled"`
}

func (h *Handlers) CreateSite(w http.ResponseWriter, r *http.Request) {
	var req createSiteRequest
	if err := readJSONStrict(r, &req); err != nil {
		writeBadRequest(w, "invalid JSON: "+err.Error())
		return
	}
	if req.Name == "" {
		writeBadRequest(w, "name is required")
		return
	}
	if len(req.Domains) == 0 && req.OriginURL == "" {
		writeBadRequest(w, "domains or origin_url is required")
		return
	}

	err := h.configMgr.UpdateConfig(func(cfg *config.Config) {
		for _, s := range cfg.Sites {
			if s.Name == req.Name {
				return
			}
		}
		site := config.SiteConfig{
			Name:       req.Name,
			Domains:    req.Domains,
			Upstream:   req.OriginURL,
			Port:       req.Port,
			TLS:        req.TLS,
			WAFEnabled: req.WAFEnabled,
		}
		if site.Port == 0 {
			site.Port = 80
		}
		cfg.Sites = append(cfg.Sites, site)
	})
	if err != nil {
		if strings.Contains(err.Error(), "already") {
			writeConflict(w, "site already exists")
			return
		}
		writeInternalErr(w)
		return
	}

	writeCreated(w, map[string]string{"name": req.Name})
}

func (h *Handlers) GetSite(w http.ResponseWriter, r *http.Request) {
	id := getPathParam(r, "id")
	cfg := h.configMgr.Get()
	site := cfg.GetSite(id)
	if site == nil {
		writeNotFound(w, "site not found")
		return
	}
	writeOK(w, site)
}

type updateSiteRequest struct {
	Domains    *[]string `json:"domains,omitempty"`
	OriginURL  *string   `json:"origin_url,omitempty"`
	Port       *int      `json:"port,omitempty"`
	TLS        *bool     `json:"tls,omitempty"`
	WAFEnabled *bool     `json:"waf_enabled,omitempty"`
}

func (h *Handlers) UpdateSite(w http.ResponseWriter, r *http.Request) {
	id := getPathParam(r, "id")
	var req updateSiteRequest
	if err := readJSONStrict(r, &req); err != nil {
		writeBadRequest(w, "invalid JSON: "+err.Error())
		return
	}

	found := false
	err := h.configMgr.UpdateConfig(func(cfg *config.Config) {
		for i := range cfg.Sites {
			if cfg.Sites[i].Name == id {
				found = true
				if req.Domains != nil {
					cfg.Sites[i].Domains = *req.Domains
				}
				if req.OriginURL != nil {
					cfg.Sites[i].Upstream = *req.OriginURL
				}
				if req.Port != nil {
					cfg.Sites[i].Port = *req.Port
				}
				if req.TLS != nil {
					cfg.Sites[i].TLS = *req.TLS
				}
				if req.WAFEnabled != nil {
					cfg.Sites[i].WAFEnabled = *req.WAFEnabled
				}
				return
			}
		}
	})
	if err != nil {
		writeInternalErr(w)
		return
	}
	if !found {
		writeNotFound(w, "site not found")
		return
	}

	writeOK(w, map[string]string{"name": id})
}

func (h *Handlers) DeleteSite(w http.ResponseWriter, r *http.Request) {
	id := getPathParam(r, "id")
	found := false
	err := h.configMgr.UpdateConfig(func(cfg *config.Config) {
		for i, s := range cfg.Sites {
			if s.Name == id {
				found = true
				cfg.Sites = append(cfg.Sites[:i], cfg.Sites[i+1:]...)
				return
			}
		}
	})
	if err != nil {
		writeInternalErr(w)
		return
	}
	if !found {
		writeNotFound(w, "site not found")
		return
	}
	writeOK(w, map[string]string{"deleted": id})
}

func (h *Handlers) EnableSite(w http.ResponseWriter, r *http.Request) {
	id := getPathParam(r, "name")
	found := false
	err := h.configMgr.UpdateConfig(func(cfg *config.Config) {
		for i := range cfg.Sites {
			if cfg.Sites[i].Name == id {
				found = true
				cfg.Sites[i].WAFEnabled = true
				return
			}
		}
	})
	if err != nil {
		writeInternalErr(w)
		return
	}
	if !found {
		writeNotFound(w, "site not found")
		return
	}
	writeOK(w, map[string]string{"name": id, "waf_enabled": "true"})
}

func (h *Handlers) DisableSite(w http.ResponseWriter, r *http.Request) {
	id := getPathParam(r, "name")
	found := false
	err := h.configMgr.UpdateConfig(func(cfg *config.Config) {
		for i := range cfg.Sites {
			if cfg.Sites[i].Name == id {
				found = true
				cfg.Sites[i].WAFEnabled = false
				return
			}
		}
	})
	if err != nil {
		writeInternalErr(w)
		return
	}
	if !found {
		writeNotFound(w, "site not found")
		return
	}
	writeOK(w, map[string]string{"name": id, "waf_enabled": "false"})
}

// --- Rules ---

func (h *Handlers) ListRules(w http.ResponseWriter, r *http.Request) {
	cfg := h.configMgr.Get()
	h.mu.RLock()
	defer h.mu.RUnlock()

	allRules := cfg.Rules
	filtered := make([]config.RuleConfig, 0, len(allRules))

	severity := r.URL.Query().Get("severity")
	tag := r.URL.Query().Get("tag")
	site := r.URL.Query().Get("site")
	status := r.URL.Query().Get("status")

	for _, rule := range allRules {
		if severity != "" && rule.Severity != severity {
			continue
		}
		if tag != "" {
			found := false
			for _, t := range rule.Tags {
				if t == tag {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		if status == "enabled" && !rule.Enabled {
			continue
		}
		if status == "disabled" && rule.Enabled {
			continue
		}
		filtered = append(filtered, rule)
	}

	if site != "" {
		siteCfg := cfg.GetSite(site)
		if siteCfg != nil && siteCfg.RuleOverrides != nil {
			overridden := make([]config.RuleConfig, 0, len(filtered))
			for _, r := range filtered {
				if _, ok := siteCfg.RuleOverrides[r.ID]; ok {
					overridden = append(overridden, r)
				}
			}
			filtered = overridden
		}
	}

	writeOKWithMeta(w, filtered, len(filtered), 1)
}

type createRuleRequest struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Severity    string                 `json:"severity"`
	Action      string                 `json:"action"`
	Phase       string                 `json:"phase"`
	Priority    int                    `json:"priority"`
	Field       string                 `json:"field"`
	Operator    string                 `json:"operator"`
	Value       string                 `json:"value"`
	Transform   []string               `json:"transform"`
	Tags        []string               `json:"tags"`
	Params      map[string]interface{} `json:"params,omitempty"`
}

func (h *Handlers) CreateRule(w http.ResponseWriter, r *http.Request) {
	var req createRuleRequest
	if err := readJSONStrict(r, &req); err != nil {
		writeBadRequest(w, "invalid JSON: "+err.Error())
		return
	}
	if req.ID == "" {
		writeBadRequest(w, "id is required")
		return
	}
	if req.Field == "" {
		writeBadRequest(w, "field is required")
		return
	}
	if req.Operator == "" {
		writeBadRequest(w, "operator is required")
		return
	}

	cfg := h.configMgr.Get()
	if cfg.GetRule(req.ID) != nil {
		writeConflict(w, "rule already exists")
		return
	}

	ruleCfg := config.RuleConfig{
		ID:          req.ID,
		Name:        req.Name,
		Description: req.Description,
		Severity:    req.Severity,
		Action:      req.Action,
		Phase:       req.Phase,
		Priority:    req.Priority,
		Field:       req.Field,
		Operator:    req.Operator,
		Value:       req.Value,
		Transform:   req.Transform,
		Tags:        req.Tags,
		Params:      req.Params,
	}

	err := h.configMgr.UpdateConfig(func(cfg *config.Config) {
		cfg.Rules = append(cfg.Rules, ruleCfg)
	})
	if err != nil {
		writeBadRequest(w, err.Error())
		return
	}

	rule := rules.Rule{
		ID:          ruleCfg.ID,
		Name:        ruleCfg.Name,
		Description: ruleCfg.Description,
		Severity:    ruleCfg.Severity,
		Action:      ruleCfg.Action,
		Phase:       ruleCfg.Phase,
		Priority:    ruleCfg.Priority,
		Field:       ruleCfg.Field,
		Operator:    ruleCfg.Operator,
		Value:       ruleCfg.Value,
		Transform:   ruleCfg.Transform,
		Tags:        ruleCfg.Tags,
		Params:      ruleCfg.Params,
	}
	if err := h.ruleEngine.AddRule(rule); err != nil {
		slog.Warn("failed to add rule to engine", "error", err)
	}

	writeCreated(w, ruleCfg)
}

func (h *Handlers) GetRule(w http.ResponseWriter, r *http.Request) {
	id := getPathParam(r, "id")
	cfg := h.configMgr.Get()
	rule := cfg.GetRule(id)
	if rule == nil {
		writeNotFound(w, "rule not found")
		return
	}
	writeOK(w, rule)
}

func (h *Handlers) UpdateRule(w http.ResponseWriter, r *http.Request) {
	id := getPathParam(r, "id")
	var req createRuleRequest
	if err := readJSONStrict(r, &req); err != nil {
		writeBadRequest(w, "invalid JSON: "+err.Error())
		return
	}

	found := false
	err := h.configMgr.UpdateConfig(func(cfg *config.Config) {
		for i := range cfg.Rules {
			if cfg.Rules[i].ID == id {
				found = true
				if req.Name != "" {
					cfg.Rules[i].Name = req.Name
				}
				if req.Description != "" {
					cfg.Rules[i].Description = req.Description
				}
				if req.Severity != "" {
					cfg.Rules[i].Severity = req.Severity
				}
				if req.Action != "" {
					cfg.Rules[i].Action = req.Action
				}
				if req.Phase != "" {
					cfg.Rules[i].Phase = req.Phase
				}
				if req.Field != "" {
					cfg.Rules[i].Field = req.Field
				}
				if req.Operator != "" {
					cfg.Rules[i].Operator = req.Operator
				}
				if req.Value != "" {
					cfg.Rules[i].Value = req.Value
				}
				if req.Transform != nil {
					cfg.Rules[i].Transform = req.Transform
				}
				if req.Tags != nil {
					cfg.Rules[i].Tags = req.Tags
				}
				if req.Params != nil {
					cfg.Rules[i].Params = req.Params
				}
				if req.Priority != 0 {
					cfg.Rules[i].Priority = req.Priority
				}
				return
			}
		}
	})
	if err != nil {
		writeBadRequest(w, err.Error())
		return
	}
	if !found {
		writeNotFound(w, "rule not found")
		return
	}
	writeOK(w, map[string]string{"id": id, "updated": "true"})
}

func (h *Handlers) DeleteRule(w http.ResponseWriter, r *http.Request) {
	id := getPathParam(r, "id")
	found := false
	err := h.configMgr.UpdateConfig(func(cfg *config.Config) {
		for i, rule := range cfg.Rules {
			if rule.ID == id {
				found = true
				cfg.Rules = append(cfg.Rules[:i], cfg.Rules[i+1:]...)
				return
			}
		}
	})
	if err != nil {
		writeInternalErr(w)
		return
	}
	if !found {
		writeNotFound(w, "rule not found")
		return
	}

	if err := h.ruleEngine.DeleteRule(id); err != nil {
		slog.Warn("failed to delete rule from engine", "error", err)
	}

	writeOK(w, map[string]string{"id": id, "deleted": "true"})
}

func (h *Handlers) EnableRule(w http.ResponseWriter, r *http.Request) {
	id := getPathParam(r, "id")
	found := false
	err := h.configMgr.UpdateConfig(func(cfg *config.Config) {
		for i := range cfg.Rules {
			if cfg.Rules[i].ID == id {
				found = true
				cfg.Rules[i].Enabled = true
				return
			}
		}
	})
	if err != nil {
		writeInternalErr(w)
		return
	}
	if !found {
		writeNotFound(w, "rule not found")
		return
	}
	writeOK(w, map[string]string{"id": id, "enabled": "true"})
}

func (h *Handlers) DisableRule(w http.ResponseWriter, r *http.Request) {
	id := getPathParam(r, "id")
	found := false
	err := h.configMgr.UpdateConfig(func(cfg *config.Config) {
		for i := range cfg.Rules {
			if cfg.Rules[i].ID == id {
				found = true
				cfg.Rules[i].Enabled = false
				return
			}
		}
	})
	if err != nil {
		writeInternalErr(w)
		return
	}
	if !found {
		writeNotFound(w, "rule not found")
		return
	}
	writeOK(w, map[string]string{"id": id, "enabled": "false"})
}

type testRuleRequest struct {
	Rule   createRuleRequest `json:"rule"`
	Field  string            `json:"field"`
	Value  string            `json:"value"`
	Path   string            `json:"path"`
	Method string            `json:"method"`
	IP     string            `json:"ip"`
	Host   string            `json:"host"`
}

func (h *Handlers) TestRule(w http.ResponseWriter, r *http.Request) {
	var req testRuleRequest
	if err := readJSONStrict(r, &req); err != nil {
		writeBadRequest(w, "invalid JSON: "+err.Error())
		return
	}

	rule := rules.Rule{
		ID:        req.Rule.ID,
		Name:      req.Rule.Name,
		Severity:  req.Rule.Severity,
		Action:    req.Rule.Action,
		Phase:     req.Rule.Phase,
		Priority:  req.Rule.Priority,
		Field:     req.Rule.Field,
		Operator:  req.Rule.Operator,
		Value:     req.Rule.Value,
		Transform: req.Rule.Transform,
		Tags:      req.Rule.Tags,
	}

	fv := &rules.FieldValue{
		Path:   req.Path,
		Method: req.Method,
		RealIP: req.IP,
		Host:   req.Host,
		Headers: map[string]string{
			"User-Agent": req.Value,
		},
	}

	decision, err := h.ruleEngine.TestRule(rule, fv)
	if err != nil {
		writeBadRequest(w, "rule test failed: "+err.Error())
		return
	}

	writeOK(w, decision)
}

type bulkRuleRequest struct {
	Rules []createRuleRequest `json:"rules"`
}

func (h *Handlers) BulkRules(w http.ResponseWriter, r *http.Request) {
	var req bulkRuleRequest
	if err := readJSONStrict(r, &req); err != nil {
		writeBadRequest(w, "invalid JSON: "+err.Error())
		return
	}
	if len(req.Rules) == 0 {
		writeBadRequest(w, "rules array is required")
		return
	}
	if len(req.Rules) > 500 {
		writeBadRequest(w, "maximum 500 rules per bulk operation")
		return
	}

	created := 0
	updated := 0
	errs := make([]string, 0)

	err := h.configMgr.UpdateConfig(func(cfg *config.Config) {
		existing := make(map[string]bool)
		for _, r := range cfg.Rules {
			existing[r.ID] = true
		}

		for _, rr := range req.Rules {
			if rr.ID == "" || rr.Field == "" || rr.Operator == "" {
				errs = append(errs, fmt.Sprintf("rule missing required fields: id=%s", rr.ID))
				continue
			}
			ruleCfg := config.RuleConfig{
				ID:          rr.ID,
				Name:        rr.Name,
				Description: rr.Description,
				Severity:    rr.Severity,
				Action:      rr.Action,
				Phase:       rr.Phase,
				Priority:    rr.Priority,
				Field:       rr.Field,
				Operator:    rr.Operator,
				Value:       rr.Value,
				Transform:   rr.Transform,
				Tags:        rr.Tags,
				Params:      rr.Params,
			}
			if existing[rr.ID] {
				updated++
			} else {
				created++
			}
			cfg.Rules = append(cfg.Rules, ruleCfg)
		}
	})
	if err != nil {
		writeBadRequest(w, err.Error())
		return
	}

	writeOK(w, map[string]interface{}{
		"created": created,
		"updated": updated,
		"errors":  errs,
	})
}

func (h *Handlers) ReorderRule(w http.ResponseWriter, r *http.Request) {
	id := getPathParam(r, "id")
	var req struct {
		Priority int `json:"priority"`
	}
	if err := readJSONStrict(r, &req); err != nil {
		writeBadRequest(w, "invalid JSON: "+err.Error())
		return
	}
	if req.Priority < 0 {
		writeBadRequest(w, "priority must be >= 0")
		return
	}
	if err := h.ruleEngine.ReorderRule(id, req.Priority); err != nil {
		writeNotFound(w, err.Error())
		return
	}
	writeOK(w, map[string]interface{}{"id": id, "priority": req.Priority})
}

// --- Logs ---

type logEntry struct {
	Timestamp time.Time `json:"timestamp"`
	RequestID string    `json:"request_id"`
	Site      string    `json:"site"`
	Action    string    `json:"action"`
	Severity  string    `json:"severity"`
	RuleID    string    `json:"rule_id,omitempty"`
	RuleName  string    `json:"rule_name,omitempty"`
	IP        string    `json:"ip"`
	Method    string    `json:"method"`
	Path      string    `json:"path"`
	Status    int       `json:"status"`
	LatencyMs int64     `json:"latency_ms"`
	UserAgent string    `json:"user_agent,omitempty"`
	Country   string    `json:"country,omitempty"`
}

func (h *Handlers) QueryLogs(w http.ResponseWriter, r *http.Request) {
	h.logStoreMu.RLock()
	defer h.logStoreMu.RUnlock()

	limit := getQueryInt(r, "limit", 100)
	offset := getQueryInt(r, "offset", 0)
	if limit > 1000 {
		limit = 1000
	}

	siteFilter := r.URL.Query().Get("site")
	actionFilter := r.URL.Query().Get("action")
	severityFilter := r.URL.Query().Get("severity")

	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")
	var fromTime, toTime time.Time
	if fromStr != "" {
		if t, err := time.Parse(time.RFC3339, fromStr); err == nil {
			fromTime = t
		}
	}
	if toStr != "" {
		if t, err := time.Parse(time.RFC3339, toStr); err == nil {
			toTime = t
		}
	}

	filtered := make([]logEntry, 0, len(h.logStore))
	for _, entry := range h.logStore {
		if siteFilter != "" && entry.Site != siteFilter {
			continue
		}
		if actionFilter != "" && entry.Action != actionFilter {
			continue
		}
		if severityFilter != "" && entry.Severity != severityFilter {
			continue
		}
		if !fromTime.IsZero() && entry.Timestamp.Before(fromTime) {
			continue
		}
		if !toTime.IsZero() && entry.Timestamp.After(toTime) {
			continue
		}
		filtered = append(filtered, entry)
	}

	total := len(filtered)
	if offset >= total {
		writeOKWithMeta(w, []logEntry{}, total, offset/limit+1)
		return
	}
	end := offset + limit
	if end > total {
		end = total
	}

	writeOKWithMeta(w, filtered[offset:end], total, offset/limit+1)
}

func (h *Handlers) TailLogs(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeInternalErr(w)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	ctx := r.Context()
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case entry, ok := <-h.logCh:
			if !ok {
				return
			}
			data, _ := json.Marshal(entry)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		case <-ticker.C:
			fmt.Fprintf(w, ":ping\n\n")
			flusher.Flush()
		}
	}
}

func (h *Handlers) StreamLogs(w http.ResponseWriter, r *http.Request) {
	h.TailLogs(w, r)
}

func (h *Handlers) ExportLogs(w http.ResponseWriter, r *http.Request) {
	h.logStoreMu.RLock()
	defer h.logStoreMu.RUnlock()

	format := r.URL.Query().Get("format")
	if format == "" {
		format = "json"
	}

	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")
	var fromTime, toTime time.Time
	if fromStr != "" {
		if t, err := time.Parse(time.RFC3339, fromStr); err == nil {
			fromTime = t
		}
	}
	if toStr != "" {
		if t, err := time.Parse(time.RFC3339, toStr); err == nil {
			toTime = t
		}
	}

	filtered := make([]logEntry, 0)
	for _, entry := range h.logStore {
		if !fromTime.IsZero() && entry.Timestamp.Before(fromTime) {
			continue
		}
		if !toTime.IsZero() && entry.Timestamp.After(toTime) {
			continue
		}
		filtered = append(filtered, entry)
	}

	switch format {
	case "json":
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", "attachment; filename=logs.json")
		json.NewEncoder(w).Encode(filtered)

	case "csv":
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", "attachment; filename=logs.csv")
		cw := csv.NewWriter(w)
		cw.Write([]string{"timestamp", "request_id", "site", "action", "severity", "rule_id", "ip", "method", "path", "status", "latency_ms", "country"})
		for _, e := range filtered {
			cw.Write([]string{
				e.Timestamp.Format(time.RFC3339),
				e.RequestID,
				e.Site,
				e.Action,
				e.Severity,
				e.RuleID,
				e.IP,
				e.Method,
				e.Path,
				strconv.Itoa(e.Status),
				strconv.FormatInt(e.LatencyMs, 10),
				e.Country,
			})
		}
		cw.Flush()

	case "cef":
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Content-Disposition", "attachment; filename=logs.cef")
		for _, e := range filtered {
			severity := 0
			switch e.Severity {
			case "critical":
				severity = 10
			case "high":
				severity = 8
			case "medium":
				severity = 6
			case "low":
				severity = 4
			}
			cef := fmt.Sprintf("CEF:0|FortressWAF|%s|%s|%d|%s|%d|src=%s dhost=%s request=%s msg=%s",
				h.version, e.RuleID, severity, e.RuleName, severity, e.IP, e.Site, e.Method+" "+e.Path, e.Action)
			fmt.Fprintln(w, cef)
		}

	default:
		writeBadRequest(w, "unsupported format: "+format)
	}
}

// --- Analytics ---

func (h *Handlers) TrafficStats(w http.ResponseWriter, r *http.Request) {
	h.logStoreMu.RLock()
	defer h.logStoreMu.RUnlock()

	now := time.Now()
	window := getQueryInt(r, "window", 60)
	since := now.Add(-time.Duration(window) * time.Minute)

	var totalReqs, blockedReqs int
	var totalLatency int64
	var bandwidth int64

	statusCounts := make(map[int]int)
	methodCounts := make(map[string]int)
	buckets := make(map[string]int)

	for _, entry := range h.logStore {
		if entry.Timestamp.Before(since) {
			continue
		}
		totalReqs++
		totalLatency += entry.LatencyMs
		bandwidth += 1024
		statusCounts[entry.Status]++
		methodCounts[entry.Method]++
		bucket := entry.Timestamp.Truncate(time.Duration(window/12) * time.Minute).Format(time.RFC3339)
		buckets[bucket]++
		if entry.Action == "block" {
			blockedReqs++
		}
	}

	avgLatency := float64(0)
	if totalReqs > 0 {
		avgLatency = float64(totalLatency) / float64(totalReqs)
	}
	reqPerSec := float64(totalReqs) / float64(window*60)

	writeOK(w, map[string]interface{}{
		"window_minutes":   window,
		"total_requests":   totalReqs,
		"blocked":          blockedReqs,
		"requests_per_sec": reqPerSec,
		"avg_latency_ms":   avgLatency,
		"bandwidth_bytes":  bandwidth,
		"by_status":        statusCounts,
		"by_method":        methodCounts,
		"timeline":         buckets,
	})
}

func (h *Handlers) AttackStats(w http.ResponseWriter, r *http.Request) {
	h.logStoreMu.RLock()
	defer h.logStoreMu.RUnlock()

	window := getQueryInt(r, "window", 60)
	since := time.Now().Add(-time.Duration(window) * time.Minute)

	byType := make(map[string]int)
	byCountry := make(map[string]int)
	byEndpoint := make(map[string]int)

	for _, entry := range h.logStore {
		if entry.Timestamp.Before(since) {
			continue
		}
		if entry.Action != "block" && entry.Action != "challenge" {
			continue
		}
		byType[entry.Severity]++
		if entry.Country != "" {
			byCountry[entry.Country]++
		}
		if entry.Path != "" {
			byEndpoint[entry.Path]++
		}
	}

	writeOK(w, map[string]interface{}{
		"window_minutes": window,
		"by_type":        byType,
		"by_country":     byCountry,
		"by_endpoint":    byEndpoint,
	})
}

func (h *Handlers) TopEndpoints(w http.ResponseWriter, r *http.Request) {
	h.logStoreMu.RLock()
	defer h.logStoreMu.RUnlock()

	limit := getQueryInt(r, "limit", 20)
	window := getQueryInt(r, "window", 60)
	since := time.Now().Add(-time.Duration(window) * time.Minute)

	endpointStats := make(map[string]*struct {
		Total   int
		Blocked int
	})
	for _, entry := range h.logStore {
		if entry.Timestamp.Before(since) {
			continue
		}
		stat, ok := endpointStats[entry.Path]
		if !ok {
			stat = &struct{ Total, Blocked int }{}
			endpointStats[entry.Path] = stat
		}
		stat.Total++
		if entry.Action == "block" {
			stat.Blocked++
		}
	}

	type endpointResult struct {
		Path    string `json:"path"`
		Total   int    `json:"total"`
		Blocked int    `json:"blocked"`
	}

	results := make([]endpointResult, 0, len(endpointStats))
	for path, stat := range endpointStats {
		results = append(results, endpointResult{Path: path, Total: stat.Total, Blocked: stat.Blocked})
	}

	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].Blocked > results[i].Blocked {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	if limit < len(results) {
		results = results[:limit]
	}

	writeOKWithMeta(w, results, len(results), 1)
}

func (h *Handlers) TopIPs(w http.ResponseWriter, r *http.Request) {
	h.logStoreMu.RLock()
	defer h.logStoreMu.RUnlock()

	limit := getQueryInt(r, "limit", 20)
	window := getQueryInt(r, "window", 60)
	since := time.Now().Add(-time.Duration(window) * time.Minute)

	ipStats := make(map[string]*struct {
		Total     int
		Blocked   int
		LastSeen  time.Time
		Countries map[string]bool
	})
	for _, entry := range h.logStore {
		if entry.Timestamp.Before(since) {
			continue
		}
		stat, ok := ipStats[entry.IP]
		if !ok {
			stat = &struct {
				Total     int
				Blocked   int
				LastSeen  time.Time
				Countries map[string]bool
			}{Countries: make(map[string]bool)}
			ipStats[entry.IP] = stat
		}
		stat.Total++
		if entry.Action == "block" {
			stat.Blocked++
		}
		if entry.Timestamp.After(stat.LastSeen) {
			stat.LastSeen = entry.Timestamp
		}
		if entry.Country != "" {
			stat.Countries[entry.Country] = true
		}
	}

	type ipResult struct {
		IP        string   `json:"ip"`
		Total     int      `json:"total"`
		Blocked   int      `json:"blocked"`
		LastSeen  string   `json:"last_seen"`
		Countries []string `json:"countries"`
	}

	results := make([]ipResult, 0, len(ipStats))
	for ip, stat := range ipStats {
		countries := make([]string, 0, len(stat.Countries))
		for c := range stat.Countries {
			countries = append(countries, c)
		}
		results = append(results, ipResult{
			IP:        ip,
			Total:     stat.Total,
			Blocked:   stat.Blocked,
			LastSeen:  stat.LastSeen.Format(time.RFC3339),
			Countries: countries,
		})
	}

	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].Blocked > results[i].Blocked {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	if limit < len(results) {
		results = results[:limit]
	}

	writeOKWithMeta(w, results, len(results), 1)
}

func (h *Handlers) GeoStats(w http.ResponseWriter, r *http.Request) {
	h.logStoreMu.RLock()
	defer h.logStoreMu.RUnlock()

	window := getQueryInt(r, "window", 1440)
	since := time.Now().Add(-time.Duration(window) * time.Minute)

	byCountry := make(map[string]int)
	blockedByCountry := make(map[string]int)
	byAction := make(map[string]int)

	for _, entry := range h.logStore {
		if entry.Timestamp.Before(since) {
			continue
		}
		if entry.Country == "" {
			entry.Country = "unknown"
		}
		byCountry[entry.Country]++
		byAction[entry.Action]++
		if entry.Action == "block" {
			blockedByCountry[entry.Country]++
		}
	}

	type countryStat struct {
		Country  string  `json:"country"`
		Requests int     `json:"requests"`
		Blocked  int     `json:"blocked"`
		BlockPct float64 `json:"block_pct"`
	}
	results := make([]countryStat, 0, len(byCountry))
	for country, total := range byCountry {
		blocked := blockedByCountry[country]
		pct := float64(0)
		if total > 0 {
			pct = float64(blocked) / float64(total) * 100
		}
		results = append(results, countryStat{
			Country:  country,
			Requests: total,
			Blocked:  blocked,
			BlockPct: pct,
		})
	}

	writeOK(w, map[string]interface{}{
		"window_minutes": window,
		"countries":      results,
		"by_action":      byAction,
	})
}

// --- Virtual Patches ---

type patchRecord struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	Description string              `json:"description"`
	CVE         string              `json:"cve,omitempty"`
	Severity    string              `json:"severity"`
	Rules       []config.RuleConfig `json:"rules"`
	Status      string              `json:"status"`
	CreatedAt   time.Time           `json:"created_at"`
	DeployedAt  time.Time           `json:"deployed_at,omitempty"`
}

func (h *Handlers) ListPatches(w http.ResponseWriter, r *http.Request) {
	h.patchesMu.RLock()
	defer h.patchesMu.RUnlock()

	status := r.URL.Query().Get("status")
	filtered := make([]patchRecord, 0, len(h.patches))
	for _, p := range h.patches {
		if status != "" && p.Status != status {
			continue
		}
		filtered = append(filtered, p)
	}

	writeOKWithMeta(w, filtered, len(filtered), 1)
}

type createPatchRequest struct {
	Name        string              `json:"name"`
	Description string              `json:"description"`
	CVE         string              `json:"cve,omitempty"`
	Severity    string              `json:"severity"`
	Rules       []createRuleRequest `json:"rules"`
}

func (h *Handlers) CreatePatch(w http.ResponseWriter, r *http.Request) {
	var req createPatchRequest
	if err := readJSONStrict(r, &req); err != nil {
		writeBadRequest(w, "invalid JSON: "+err.Error())
		return
	}
	if req.Name == "" {
		writeBadRequest(w, "name is required")
		return
	}
	if len(req.Rules) == 0 {
		writeBadRequest(w, "at least one rule is required")
		return
	}

	patch := patchRecord{
		ID:          "PATCH-" + uuid.New().String()[:8],
		Name:        req.Name,
		Description: req.Description,
		CVE:         req.CVE,
		Severity:    req.Severity,
		Status:      "draft",
		CreatedAt:   time.Now(),
	}

	for _, rr := range req.Rules {
		patch.Rules = append(patch.Rules, config.RuleConfig{
			ID:        rr.ID,
			Name:      rr.Name,
			Severity:  rr.Severity,
			Action:    rr.Action,
			Field:     rr.Field,
			Operator:  rr.Operator,
			Value:     rr.Value,
			Transform: rr.Transform,
			Tags:      rr.Tags,
		})
	}

	h.patchesMu.Lock()
	h.patches = append(h.patches, patch)
	h.patchesMu.Unlock()

	writeCreated(w, patch)
}

func (h *Handlers) GetPatch(w http.ResponseWriter, r *http.Request) {
	id := getPathParam(r, "id")
	h.patchesMu.RLock()
	defer h.patchesMu.RUnlock()

	for _, p := range h.patches {
		if p.ID == id {
			writeOK(w, p)
			return
		}
	}
	writeNotFound(w, "patch not found")
}

func (h *Handlers) ApplyPatch(w http.ResponseWriter, r *http.Request) {
	id := getPathParam(r, "id")
	h.patchesMu.Lock()
	defer h.patchesMu.Unlock()

	for i := range h.patches {
		if h.patches[i].ID == id {
			if h.patches[i].Status == "deployed" {
				writeConflict(w, "patch already deployed")
				return
			}

			err := h.configMgr.UpdateConfig(func(cfg *config.Config) {
				for _, ruleCfg := range h.patches[i].Rules {
					cfg.Rules = append(cfg.Rules, ruleCfg)
				}
			})
			if err != nil {
				writeBadRequest(w, "failed to deploy patch: "+err.Error())
				return
			}

			h.patches[i].Status = "deployed"
			h.patches[i].DeployedAt = time.Now()
			writeOK(w, h.patches[i])
			return
		}
	}
	writeNotFound(w, "patch not found")
}

func (h *Handlers) RevokePatch(w http.ResponseWriter, r *http.Request) {
	id := getPathParam(r, "id")
	h.patchesMu.Lock()
	defer h.patchesMu.Unlock()

	for i := range h.patches {
		if h.patches[i].ID == id {
			if h.patches[i].Status != "deployed" {
				writeBadRequest(w, "patch is not deployed")
				return
			}

			err := h.configMgr.UpdateConfig(func(cfg *config.Config) {
				patchRuleIDs := make(map[string]bool)
				for _, r := range h.patches[i].Rules {
					patchRuleIDs[r.ID] = true
				}
				filtered := make([]config.RuleConfig, 0, len(cfg.Rules))
				for _, r := range cfg.Rules {
					if !patchRuleIDs[r.ID] {
						filtered = append(filtered, r)
					}
				}
				cfg.Rules = filtered
			})
			if err != nil {
				writeBadRequest(w, "failed to revoke patch: "+err.Error())
				return
			}

			h.patches[i].Status = "revoked"
			writeOK(w, h.patches[i])
			return
		}
	}
	writeNotFound(w, "patch not found")
}

func (h *Handlers) DeletePatch(w http.ResponseWriter, r *http.Request) {
	id := getPathParam(r, "id")
	h.patchesMu.Lock()
	defer h.patchesMu.Unlock()

	for i, p := range h.patches {
		if p.ID == id {
			if p.Status == "deployed" {
				writeBadRequest(w, "cannot delete deployed patch, revoke first")
				return
			}
			h.patches = append(h.patches[:i], h.patches[i+1:]...)
			writeOK(w, map[string]string{"id": id, "deleted": "true"})
			return
		}
	}
	writeNotFound(w, "patch not found")
}

func (h *Handlers) DeployPatch(w http.ResponseWriter, r *http.Request) {
	h.ApplyPatch(w, r)
}

func (h *Handlers) PatchCoverage(w http.ResponseWriter, r *http.Request) {
	cfg := h.configMgr.Get()
	h.patchesMu.RLock()
	defer h.patchesMu.RUnlock()

	type coverageItem struct {
		PatchID    string `json:"patch_id"`
		PatchName  string `json:"patch_name"`
		CVE        string `json:"cve"`
		Severity   string `json:"severity"`
		Status     string `json:"status"`
		RulesCount int    `json:"rules_count"`
	}

	deployedPatches := make([]coverageItem, 0)
	draftPatches := make([]coverageItem, 0)
	deployedRuleIDs := make(map[string]bool)

	for _, p := range h.patches {
		item := coverageItem{
			PatchID:    p.ID,
			PatchName:  p.Name,
			CVE:        p.CVE,
			Severity:   p.Severity,
			Status:     p.Status,
			RulesCount: len(p.Rules),
		}
		if p.Status == "deployed" {
			deployedPatches = append(deployedPatches, item)
			for _, r := range p.Rules {
				deployedRuleIDs[r.ID] = true
			}
		} else {
			draftPatches = append(draftPatches, item)
		}
	}

	totalRules := len(cfg.Rules)
	patchedRules := len(deployedRuleIDs)

	writeOK(w, map[string]interface{}{
		"total_rules":     totalRules,
		"patched_rules":   patchedRules,
		"unpatched_rules": totalRules - patchedRules,
		"coverage_pct":    float64(0),
		"deployed":        deployedPatches,
		"draft":           draftPatches,
	})
}

// --- Config ---

func maskSecrets(cfg *config.Config) map[string]interface{} {
	data := map[string]interface{}{
		"sites":   cfg.Sites,
		"rules":   cfg.Rules,
		"logging": cfg.Logging,
		"tls":     cfg.TLS,
		"admin": map[string]interface{}{
			"port":     cfg.Admin.Port,
			"enabled":  cfg.Admin.Enabled,
			"mtls":     cfg.Admin.MTLS,
			"api_keys": len(cfg.Admin.APIKeys),
		},
		"redis": map[string]interface{}{
			"enabled":   cfg.Redis.Enabled,
			"addr":      cfg.Redis.Addr,
			"db":        cfg.Redis.DB,
			"pool_size": cfg.Redis.PoolSize,
			"ttl":       cfg.Redis.TTL,
		},
		"ml": cfg.ML,
		"db": map[string]interface{}{
			"driver":   cfg.DB.Driver,
			"max_open": cfg.DB.MaxOpen,
			"max_idle": cfg.DB.MaxIdle,
		},
	}
	return data
}

func (h *Handlers) GetConfig(w http.ResponseWriter, r *http.Request) {
	cfg := h.configMgr.Get()
	writeOK(w, maskSecrets(cfg))
}

func (h *Handlers) UpdateConfigHandler(w http.ResponseWriter, r *http.Request) {
	var updates map[string]interface{}
	if err := readJSONStrict(r, &updates); err != nil {
		writeBadRequest(w, "invalid JSON: "+err.Error())
		return
	}

	err := h.configMgr.UpdateConfig(func(cfg *config.Config) {
		for key, val := range updates {
			switch key {
			case "logging":
				if m, ok := val.(map[string]interface{}); ok {
					if v, ok := m["level"].(string); ok {
						cfg.Logging.Level = v
					}
					if v, ok := m["format"].(string); ok {
						cfg.Logging.Format = v
					}
					if v, ok := m["verbose"].(bool); ok {
						cfg.Logging.Verbose = v
					}
				}
			case "ml":
				if m, ok := val.(map[string]interface{}); ok {
					if v, ok := m["enabled"].(bool); ok {
						cfg.ML.Enabled = v
					}
					if v, ok := m["endpoint"].(string); ok {
						cfg.ML.Endpoint = v
					}
					if v, ok := m["timeout"].(float64); ok {
						cfg.ML.TimeoutSec = int(v)
					}
				}
			case "redis":
				if m, ok := val.(map[string]interface{}); ok {
					if v, ok := m["enabled"].(bool); ok {
						cfg.Redis.Enabled = v
					}
					if v, ok := m["addr"].(string); ok {
						cfg.Redis.Addr = v
					}
				}
			}
		}
	})
	if err != nil {
		writeBadRequest(w, "validation failed: "+err.Error())
		return
	}

	writeOK(w, maskSecrets(h.configMgr.Get()))
}

func (h *Handlers) ValidateConfig(w http.ResponseWriter, r *http.Request) {
	var cfg config.Config
	if err := readJSONStrict(r, &cfg); err != nil {
		writeBadRequest(w, "invalid JSON: "+err.Error())
		return
	}

	if err := cfg.Validate(); err != nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"valid":  false,
			"errors": []string{err.Error()},
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"valid":  true,
		"errors": []string{},
	})
}

func (h *Handlers) ExportConfig(w http.ResponseWriter, r *http.Request) {
	cfg := h.configMgr.Get()
	data, err := yaml.Marshal(cfg)
	if err != nil {
		writeInternalErr(w)
		return
	}

	w.Header().Set("Content-Type", "application/x-yaml")
	w.Header().Set("Content-Disposition", "attachment; filename=fortresswaf.yaml")
	w.Write(data)
}

func (h *Handlers) ImportConfig(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeBadRequest(w, "failed to read body")
		return
	}
	defer r.Body.Close()

	var cfg config.Config
	if err := yaml.Unmarshal(body, &cfg); err != nil {
		writeBadRequest(w, "invalid YAML: "+err.Error())
		return
	}
	if err := cfg.Validate(); err != nil {
		writeBadRequest(w, "config validation failed: "+err.Error())
		return
	}

	err = h.configMgr.UpdateConfig(func(current *config.Config) {
		current.Sites = cfg.Sites
		current.Rules = cfg.Rules
		current.ML = cfg.ML
		current.Redis = cfg.Redis
		current.DB = cfg.DB
		current.Logging = cfg.Logging
		current.TLS = cfg.TLS
		current.Admin = cfg.Admin
	})
	if err != nil {
		writeBadRequest(w, err.Error())
		return
	}

	writeOK(w, map[string]string{"status": "imported"})
}

func (h *Handlers) DiffConfig(w http.ResponseWriter, r *http.Request) {
	var proposed config.Config
	if err := readJSONStrict(r, &proposed); err != nil {
		writeBadRequest(w, "invalid JSON: "+err.Error())
		return
	}

	current := h.configMgr.Get()

	type diffEntry struct {
		Field string      `json:"field"`
		Old   interface{} `json:"old"`
		New   interface{} `json:"new"`
	}

	diffs := []diffEntry{}

	if len(current.Sites) != len(proposed.Sites) {
		diffs = append(diffs, diffEntry{Field: "sites.count", Old: len(current.Sites), New: len(proposed.Sites)})
	}
	if len(current.Rules) != len(proposed.Rules) {
		diffs = append(diffs, diffEntry{Field: "rules.count", Old: len(current.Rules), New: len(proposed.Rules)})
	}
	if current.ML.Enabled != proposed.ML.Enabled {
		diffs = append(diffs, diffEntry{Field: "ml.enabled", Old: current.ML.Enabled, New: proposed.ML.Enabled})
	}
	if current.ML.Endpoint != proposed.ML.Endpoint {
		diffs = append(diffs, diffEntry{Field: "ml.endpoint", Old: current.ML.Endpoint, New: proposed.ML.Endpoint})
	}
	if current.Redis.Enabled != proposed.Redis.Enabled {
		diffs = append(diffs, diffEntry{Field: "redis.enabled", Old: current.Redis.Enabled, New: proposed.Redis.Enabled})
	}
	if current.Logging.Level != proposed.Logging.Level {
		diffs = append(diffs, diffEntry{Field: "logging.level", Old: current.Logging.Level, New: proposed.Logging.Level})
	}

	writeOK(w, map[string]interface{}{
		"has_changes": len(diffs) > 0,
		"diffs":       diffs,
	})
}

// --- Tenants ---

type tenantRecord struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Enabled     bool                   `json:"enabled"`
	Settings    map[string]interface{} `json:"settings,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

func (h *Handlers) ListTenants(w http.ResponseWriter, r *http.Request) {
	h.tenantsMu.RLock()
	defer h.tenantsMu.RUnlock()
	writeOKWithMeta(w, h.tenants, len(h.tenants), 1)
}

type createTenantRequest struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Settings    map[string]interface{} `json:"settings,omitempty"`
}

func (h *Handlers) CreateTenant(w http.ResponseWriter, r *http.Request) {
	var req createTenantRequest
	if err := readJSONStrict(r, &req); err != nil {
		writeBadRequest(w, "invalid JSON: "+err.Error())
		return
	}
	if req.Name == "" {
		writeBadRequest(w, "name is required")
		return
	}

	tenant := tenantRecord{
		ID:          "TENANT-" + uuid.New().String()[:8],
		Name:        req.Name,
		Description: req.Description,
		Enabled:     true,
		Settings:    req.Settings,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	h.tenantsMu.Lock()
	for _, t := range h.tenants {
		if t.Name == req.Name {
			h.tenantsMu.Unlock()
			writeConflict(w, "tenant already exists")
			return
		}
	}
	h.tenants = append(h.tenants, tenant)
	h.tenantsMu.Unlock()

	writeCreated(w, tenant)
}

func (h *Handlers) GetTenant(w http.ResponseWriter, r *http.Request) {
	id := getPathParam(r, "id")
	h.tenantsMu.RLock()
	defer h.tenantsMu.RUnlock()

	for _, t := range h.tenants {
		if t.ID == id {
			writeOK(w, t)
			return
		}
	}
	writeNotFound(w, "tenant not found")
}

func (h *Handlers) UpdateTenant(w http.ResponseWriter, r *http.Request) {
	id := getPathParam(r, "id")
	var req createTenantRequest
	if err := readJSONStrict(r, &req); err != nil {
		writeBadRequest(w, "invalid JSON: "+err.Error())
		return
	}

	h.tenantsMu.Lock()
	defer h.tenantsMu.Unlock()

	for i := range h.tenants {
		if h.tenants[i].ID == id {
			if req.Name != "" {
				h.tenants[i].Name = req.Name
			}
			if req.Description != "" {
				h.tenants[i].Description = req.Description
			}
			if req.Settings != nil {
				h.tenants[i].Settings = req.Settings
			}
			h.tenants[i].UpdatedAt = time.Now()
			writeOK(w, h.tenants[i])
			return
		}
	}
	writeNotFound(w, "tenant not found")
}

func (h *Handlers) DeleteTenant(w http.ResponseWriter, r *http.Request) {
	id := getPathParam(r, "id")
	h.tenantsMu.Lock()
	defer h.tenantsMu.Unlock()

	for i, t := range h.tenants {
		if t.ID == id {
			h.tenants = append(h.tenants[:i], h.tenants[i+1:]...)
			writeOK(w, map[string]string{"id": id, "deleted": "true"})
			return
		}
	}
	writeNotFound(w, "tenant not found")
}

// --- Alerts ---

type alertRecord struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Severity  string    `json:"severity"`
	Message   string    `json:"message"`
	Site      string    `json:"site,omitempty"`
	RuleID    string    `json:"rule_id,omitempty"`
	Acked     bool      `json:"acked"`
	CreatedAt time.Time `json:"created_at"`
}

type alertConfigRecord struct {
	Threshold       int      `json:"threshold"`
	IntervalSeconds int      `json:"interval_seconds"`
	Enabled         bool     `json:"enabled"`
	Channels        []string `json:"channels"`
	Rules           []string `json:"rules"`
}

type alertChannelRecord struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Type      string            `json:"type"`
	Enabled   bool              `json:"enabled"`
	Config    map[string]string `json:"config"`
	CreatedAt time.Time         `json:"created_at"`
}

func (h *Handlers) ListAlerts(w http.ResponseWriter, r *http.Request) {
	h.alertsMu.RLock()
	defer h.alertsMu.RUnlock()

	severity := r.URL.Query().Get("severity")
	acked := r.URL.Query().Get("acked")
	limit := getQueryInt(r, "limit", 100)

	filtered := make([]alertRecord, 0, len(h.alerts))
	for _, a := range h.alerts {
		if severity != "" && a.Severity != severity {
			continue
		}
		if acked == "true" && !a.Acked {
			continue
		}
		if acked == "false" && a.Acked {
			continue
		}
		filtered = append(filtered, a)
	}

	if limit < len(filtered) {
		filtered = filtered[:limit]
	}

	writeOKWithMeta(w, filtered, len(h.alerts), 1)
}

func (h *Handlers) CreateAlert(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Type     string `json:"type"`
		Severity string `json:"severity"`
		Message  string `json:"message"`
		Site     string `json:"site,omitempty"`
		RuleID   string `json:"rule_id,omitempty"`
	}
	if err := readJSONStrict(r, &req); err != nil {
		writeBadRequest(w, "invalid JSON: "+err.Error())
		return
	}
	if req.Type == "" || req.Message == "" {
		writeBadRequest(w, "type and message are required")
		return
	}

	alert := alertRecord{
		ID:        "ALERT-" + uuid.New().String()[:8],
		Type:      req.Type,
		Severity:  req.Severity,
		Message:   req.Message,
		Site:      req.Site,
		RuleID:    req.RuleID,
		CreatedAt: time.Now(),
	}

	h.alertsMu.Lock()
	h.alerts = append([]alertRecord{alert}, h.alerts...)
	if len(h.alerts) > 10000 {
		h.alerts = h.alerts[:10000]
	}
	h.alertsMu.Unlock()

	writeCreated(w, alert)
}

func (h *Handlers) UpdateAlert(w http.ResponseWriter, r *http.Request) {
	id := getPathParam(r, "id")
	var req struct {
		Acked *bool `json:"acked"`
	}
	if err := readJSONStrict(r, &req); err != nil {
		writeBadRequest(w, "invalid JSON: "+err.Error())
		return
	}

	h.alertsMu.Lock()
	defer h.alertsMu.Unlock()

	for i := range h.alerts {
		if h.alerts[i].ID == id {
			if req.Acked != nil {
				h.alerts[i].Acked = *req.Acked
			}
			writeOK(w, h.alerts[i])
			return
		}
	}
	writeNotFound(w, "alert not found")
}

func (h *Handlers) UpdateAlertConfig(w http.ResponseWriter, r *http.Request) {
	var req alertConfigRecord
	if err := readJSONStrict(r, &req); err != nil {
		writeBadRequest(w, "invalid JSON: "+err.Error())
		return
	}

	h.alertsMu.Lock()
	if req.Threshold > 0 {
		h.alertConfig.Threshold = req.Threshold
	}
	if req.IntervalSeconds > 0 {
		h.alertConfig.IntervalSeconds = req.IntervalSeconds
	}
	if req.Channels != nil {
		h.alertConfig.Channels = req.Channels
	}
	if req.Rules != nil {
		h.alertConfig.Rules = req.Rules
	}
	h.alertConfig.Enabled = req.Enabled
	h.alertsMu.Unlock()

	writeOK(w, h.alertConfig)
}

func (h *Handlers) ListAlertChannels(w http.ResponseWriter, r *http.Request) {
	h.alertChMu.RLock()
	defer h.alertChMu.RUnlock()
	writeOKWithMeta(w, h.alertChannels, len(h.alertChannels), 1)
}

func (h *Handlers) CreateAlertChannel(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name    string            `json:"name"`
		Type    string            `json:"type"`
		Enabled *bool             `json:"enabled"`
		Config  map[string]string `json:"config"`
	}
	if err := readJSONStrict(r, &req); err != nil {
		writeBadRequest(w, "invalid JSON: "+err.Error())
		return
	}
	if req.Name == "" || req.Type == "" {
		writeBadRequest(w, "name and type are required")
		return
	}

	channel := alertChannelRecord{
		ID:        "CH-" + uuid.New().String()[:8],
		Name:      req.Name,
		Type:      req.Type,
		Enabled:   true,
		Config:    req.Config,
		CreatedAt: time.Now(),
	}
	if req.Enabled != nil {
		channel.Enabled = *req.Enabled
	}

	h.alertChMu.Lock()
	h.alertChannels = append(h.alertChannels, channel)
	h.alertChMu.Unlock()

	writeCreated(w, channel)
}

func (h *Handlers) DeleteAlertChannel(w http.ResponseWriter, r *http.Request) {
	id := getPathParam(r, "id")
	h.alertChMu.Lock()
	defer h.alertChMu.Unlock()

	for i, ch := range h.alertChannels {
		if ch.ID == id {
			h.alertChannels = append(h.alertChannels[:i], h.alertChannels[i+1:]...)
			writeOK(w, map[string]string{"id": id, "deleted": "true"})
			return
		}
	}
	writeNotFound(w, "alert channel not found")
}

// --- System ---

func (h *Handlers) Health(w http.ResponseWriter, r *http.Request) {
	writeOK(w, map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

func (h *Handlers) SystemStatus(w http.ResponseWriter, r *http.Request) {
	uptime := time.Since(h.startTime)
	cfg := h.configMgr.Get()

	h.logStoreMu.RLock()
	logCount := len(h.logStore)
	h.logStoreMu.RUnlock()

	h.alertsMu.RLock()
	alertCount := len(h.alerts)
	h.alertsMu.RUnlock()

	writeOK(w, map[string]interface{}{
		"status":     "running",
		"version":    h.version,
		"uptime":     uptime.String(),
		"uptime_sec": int(uptime.Seconds()),
		"sites":      len(cfg.Sites),
		"rules":      len(cfg.Rules),
		"logs":       logCount,
		"alerts":     alertCount,
		"goroutines": 0,
		"started_at": h.startTime.Format(time.RFC3339),
	})
}

func (h *Handlers) Version(w http.ResponseWriter, r *http.Request) {
	writeOK(w, map[string]string{
		"version": h.version,
	})
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type sessionRecord struct {
	Token     string    `json:"token"`
	UserID    string    `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

func (h *Handlers) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := readJSONStrict(r, &req); err != nil {
		writeBadRequest(w, "invalid JSON: "+err.Error())
		return
	}
	if req.Username == "" || req.Password == "" {
		writeBadRequest(w, "username and password are required")
		return
	}

	token := "jwt-" + uuid.New().String()
	expires := time.Now().Add(24 * time.Hour)

	h.sessionsMu.Lock()
	h.sessions[token] = sessionRecord{
		Token:     token,
		UserID:    req.Username,
		CreatedAt: time.Now(),
		ExpiresAt: expires,
	}
	h.sessionsMu.Unlock()

	writeOK(w, map[string]interface{}{
		"token":      token,
		"expires_at": expires.Format(time.RFC3339),
		"token_type": "bearer",
	})
}

func (h *Handlers) Logout(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	token := ""
	fmt.Sscanf(authHeader, "Bearer %s", &token)
	if token == "" {
		writeUnauthorized(w)
		return
	}

	h.sessionsMu.Lock()
	delete(h.sessions, token)
	h.sessionsMu.Unlock()

	writeOK(w, map[string]string{"status": "logged_out"})
}

func (h *Handlers) AuthMe(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	token := ""
	fmt.Sscanf(authHeader, "Bearer %s", &token)
	if token == "" {
		writeUnauthorized(w)
		return
	}

	h.sessionsMu.RLock()
	session, ok := h.sessions[token]
	h.sessionsMu.RUnlock()

	if !ok {
		writeUnauthorized(w)
		return
	}
	if time.Now().After(session.ExpiresAt) {
		h.sessionsMu.Lock()
		delete(h.sessions, token)
		h.sessionsMu.Unlock()
		writeUnauthorized(w)
		return
	}

	writeOK(w, map[string]interface{}{
		"user_id":    session.UserID,
		"created_at": session.CreatedAt.Format(time.RFC3339),
		"expires_at": session.ExpiresAt.Format(time.RFC3339),
	})
}

type apiKeyRecord struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Key       string    `json:"key,omitempty"`
	UserID    string    `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at,omitempty"`
}

func (h *Handlers) CreateAPIKey(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name      string `json:"name"`
		UserID    string `json:"user_id"`
		ExpiresIn int    `json:"expires_in_hours"`
	}
	if err := readJSONStrict(r, &req); err != nil {
		writeBadRequest(w, "invalid JSON: "+err.Error())
		return
	}
	if req.Name == "" {
		writeBadRequest(w, "name is required")
		return
	}

	rawKey := "fwaf_" + uuid.New().String()
	keyID := "KEY-" + uuid.New().String()[:8]

	record := apiKeyRecord{
		ID:        keyID,
		Name:      req.Name,
		Key:       rawKey,
		UserID:    req.UserID,
		CreatedAt: time.Now(),
	}
	if req.ExpiresIn > 0 {
		record.ExpiresAt = time.Now().Add(time.Duration(req.ExpiresIn) * time.Hour)
	}

	h.apiKeysMu.Lock()
	h.apiKeys[rawKey] = record
	h.apiKeysMu.Unlock()

	writeCreated(w, map[string]interface{}{
		"id":         record.ID,
		"name":       record.Name,
		"key":        rawKey,
		"expires_at": record.ExpiresAt,
	})
}

func (h *Handlers) RevokeAPIKey(w http.ResponseWriter, r *http.Request) {
	id := getPathParam(r, "id")
	h.apiKeysMu.Lock()
	defer h.apiKeysMu.Unlock()

	for key, record := range h.apiKeys {
		if record.ID == id {
			delete(h.apiKeys, key)
			writeOK(w, map[string]string{"id": id, "revoked": "true"})
			return
		}
	}
	writeNotFound(w, "API key not found")
}

// --- Admin ---

func (h *Handlers) AdminStatus(w http.ResponseWriter, r *http.Request) {
	uptime := time.Since(h.startTime)
	cfg := h.configMgr.Get()

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "running",
		"version": h.version,
		"uptime":  uptime.String(),
		"sites":   len(cfg.Sites),
		"rules":   len(cfg.Rules),
		"config":  maskSecrets(cfg),
	})
}

func (h *Handlers) AdminConfig(w http.ResponseWriter, r *http.Request) {
	cfg := h.configMgr.Get()
	writeOK(w, cfg)
}

func (h *Handlers) AdminReload(w http.ResponseWriter, r *http.Request) {
	if err := h.configMgr.Reload(); err != nil {
		writeBadRequest(w, "reload failed: "+err.Error())
		return
	}

	allRules := h.configMgr.Get().Rules
	engineRules := make([]rules.Rule, len(allRules))
	for i, rc := range allRules {
		engineRules[i] = rules.Rule{
			ID:          rc.ID,
			Name:        rc.Name,
			Description: rc.Description,
			Enabled:     rc.Enabled,
			Severity:    rc.Severity,
			Action:      rc.Action,
			Phase:       rc.Phase,
			Priority:    rc.Priority,
			Field:       rc.Field,
			Operator:    rc.Operator,
			Value:       rc.Value,
			Transform:   rc.Transform,
			Tags:        rc.Tags,
			Params:      rc.Params,
		}
	}
	h.ruleEngine.AtomicSwap(engineRules)

	writeOK(w, map[string]string{"status": "reloaded"})
}

var (
	_ = json.Marshal
	_ = io.EOF
	_ = (*context.CancelFunc)(nil)
)
