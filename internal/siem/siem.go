package siem

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"
)

type Manager struct {
	mu        sync.RWMutex
	config    SIEMConfig
	exporters map[string]Exporter
	buffer    []SIEMEvent
	flushCh   chan struct{}
	done      chan struct{}
}

type SIEMConfig struct {
	Enabled        bool
	ExportInterval time.Duration
	BatchSize      int
	Exporters      []ExporterConfig
}

type ExporterConfig struct {
	Type      string
	Enabled   bool
	URL       string
	Token     string
	Index     string
	Username  string
	Password  string
	VerifySSL bool
}

type SIEMEvent struct {
	Timestamp   time.Time         `json:"timestamp"`
	EventType   string            `json:"event_type"`
	Host        string            `json:"host"`
	Source      string            `json:"source"`
	EventID     int               `json:"event_id"`
	Name        string            `json:"name"`
	Severity    int               `json:"severity"`
	SrcIP       string            `json:"src_ip"`
	DstIP       string            `json:"dst_ip"`
	HttpMethod  string            `json:"http_method,omitempty"`
	HttpURI     string            `json:"http_uri,omitempty"`
	UserAgent   string            `json:"http_user_agent,omitempty"`
	AttackType  string            `json:"attack_type,omitempty"`
	RuleID      string            `json:"rule_id,omitempty"`
	ThreatScore float64           `json:"threat_score"`
	Blocked     bool              `json:"blocked"`
	Country     string            `json:"country,omitempty"`
	Latency     float64           `json:"latency_ms,omitempty"`
	RawEvent    string            `json:"raw_event,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

type Exporter interface {
	Send(events []SIEMEvent) error
	Close() error
}

func NewManager(cfg SIEMConfig) (*Manager, error) {
	m := &Manager{
		config:    cfg,
		exporters: make(map[string]Exporter),
		buffer:    make([]SIEMEvent, 0, cfg.BatchSize),
		flushCh:   make(chan struct{}, 1),
		done:      make(chan struct{}),
	}

	for _, ec := range cfg.Exporters {
		if !ec.Enabled {
			continue
		}

		var ex Exporter
		var err error

		switch ec.Type {
		case "splunk":
			ex, err = newSplunkExporter(ec)
		case "elasticsearch":
			ex, err = newElasticsearchExporter(ec)
		default:
			slog.Warn("unknown SIEM exporter type", "type", ec.Type)
			continue
		}

		if err != nil {
			slog.Warn("failed to create SIEM exporter", "type", ec.Type, "error", err)
			continue
		}

		m.exporters[ec.Type] = ex
	}

	if len(m.exporters) == 0 {
		slog.Warn("no SIEM exporters configured")
	}

	go m.flushLoop()

	return m, nil
}

func (m *Manager) Send(event SIEMEvent) {
	m.mu.Lock()
	m.buffer = append(m.buffer, event)
	shouldFlush := len(m.buffer) >= m.config.BatchSize
	m.mu.Unlock()

	if shouldFlush {
		select {
		case m.flushCh <- struct{}{}:
		default:
			slog.Warn("siem flush channel full, signal dropped")
		}
	}
}

func (m *Manager) SendBatch(events []SIEMEvent) {
	for _, e := range events {
		m.Send(e)
	}
}

func (m *Manager) flushLoop() {
	ticker := time.NewTicker(m.config.ExportInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.done:
			return
		case <-ticker.C:
			m.flush()
		case <-m.flushCh:
			m.flush()
		}
	}
}

func (m *Manager) flush() {
	m.mu.Lock()
	if len(m.buffer) == 0 {
		m.mu.Unlock()
		return
	}

	events := m.buffer
	m.buffer = make([]SIEMEvent, 0, m.config.BatchSize)
	m.mu.Unlock()

	for _, ex := range m.exporters {
		if err := ex.Send(events); err != nil {
			slog.Error("SIEM export failed", "exporter", fmt.Sprintf("%T", ex), "error", err)
		}
	}
}

func (m *Manager) Close() error {
	close(m.done)
	m.flush()

	var errs []string
	for name, ex := range m.exporters {
		if err := ex.Close(); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", name, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("SIEM close errors: %s", strings.Join(errs, ", "))
	}
	return nil
}

func (m *Manager) Stats() map[string]int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return map[string]int{
		"buffer_size": len(m.buffer),
		"exporters":   len(m.exporters),
	}
}

type SplunkExporter struct {
	url       string
	token     string
	index     string
	client    *http.Client
	verifySSL bool
}

func newSplunkExporter(cfg ExporterConfig) (*SplunkExporter, error) {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: !cfg.VerifySSL,
	}
	return &SplunkExporter{
		url:       cfg.URL,
		token:     cfg.Token,
		index:     cfg.Index,
		verifySSL: cfg.VerifySSL,
		client: &http.Client{
			Timeout:   30 * time.Second,
			Transport: transport,
		},
	}, nil
}

func (s *SplunkExporter) Send(events []SIEMEvent) error {
	if len(events) == 0 {
		return nil
	}

	var buf bytes.Buffer
	for _, event := range events {
		data, err := json.Marshal(event)
		if err != nil {
			continue
		}

		hecEvent := map[string]interface{}{
			"event": string(data),
			"host":  event.Host,
			"index": s.index,
			"time":  float64(event.Timestamp.Unix()),
		}

		if err := json.NewEncoder(&buf).Encode(hecEvent); err != nil {
			continue
		}
	}

	req, err := http.NewRequest("POST", s.url, &buf)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Splunk "+s.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("splunk returned %d", resp.StatusCode)
	}

	return nil
}

func (s *SplunkExporter) Close() error {
	return nil
}

type ElasticsearchExporter struct {
	urls     []string
	index    string
	username string
	password string
	client   *http.Client
}

func newElasticsearchExporter(cfg ExporterConfig) (*ElasticsearchExporter, error) {
	urls := strings.Split(cfg.URL, ",")

	return &ElasticsearchExporter{
		urls:     urls,
		index:    cfg.Index,
		username: cfg.Username,
		password: cfg.Password,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

func (e *ElasticsearchExporter) Send(events []SIEMEvent) error {
	if len(events) == 0 {
		return nil
	}

	var buf bytes.Buffer
	indexName := fmt.Sprintf("%s-%s", e.index, time.Now().Format("2006.01.02"))

	for _, event := range events {
		meta := map[string]interface{}{
			"index": indexName,
		}

		if err := json.NewEncoder(&buf).Encode(meta); err != nil {
			continue
		}

		if err := json.NewEncoder(&buf).Encode(event); err != nil {
			continue
		}
	}

	req, err := http.NewRequest("POST", e.urls[0]+"/_bulk", &buf)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-ndjson")
	if e.username != "" {
		req.SetBasicAuth(e.username, e.password)
	}

	resp, err := e.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("elasticsearch returned %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (e *ElasticsearchExporter) Close() error {
	return nil
}

func FormatCEF(event SIEMEvent) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("CEF:%d|%s|%s|%s|%d|%s|%s",
		0,
		event.Host,
		"FortressWAF",
		event.Name,
		event.Severity,
		event.Name,
		formatCEFSeverity(event.Severity),
	))

	sb.WriteString(fmt.Sprintf(" src=%s", event.SrcIP))
	sb.WriteString(fmt.Sprintf(" dst=%s", event.DstIP))
	sb.WriteString(fmt.Sprintf(" dpt=%s", event.Host))

	if event.HttpMethod != "" {
		sb.WriteString(fmt.Sprintf(" requestMethod=%s", event.HttpMethod))
	}
	if event.HttpURI != "" {
		sb.WriteString(fmt.Sprintf(" request=%s", event.HttpURI))
	}
	if event.AttackType != "" {
		sb.WriteString(fmt.Sprintf(" attackType=%s", event.AttackType))
	}
	if event.RuleID != "" {
		sb.WriteString(fmt.Sprintf(" ruleId=%s", event.RuleID))
	}

	return sb.String()
}

func formatCEFSeverity(level int) string {
	switch {
	case level >= 8:
		return "Very High"
	case level >= 6:
		return "High"
	case level >= 4:
		return "Medium"
	case level >= 2:
		return "Low"
	default:
		return "Unknown"
	}
}
