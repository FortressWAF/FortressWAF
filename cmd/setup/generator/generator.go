package generator

import (
	"fmt"
	"os"
	"text/template"

	"github.com/zulfff/FortressWAF/internal/config"
)

type ConfigGenerator struct {
	Preset    string
	Answers   WizardAnswers
	Overrides map[string]interface{}
}

type WizardAnswers struct {
	ListenAddr     string
	AdminAddr      string
	EnableHTTP3    bool
	HTTP3Port      int
	EnableMTLS     bool
	BackendURL     string
	HealthCheck    bool
	HealthPath     string
	Domains        []string
	TLSMode        string
	CertFile       string
	KeyFile        string
	RuleTemplates  []string
	OWASPLevel     string
	RedisEnabled   bool
	RedisURL       string
	PostgresEnabled bool
	PostgresURL    string
	VaultEnabled   bool
	SIEMEnabled    bool
	Workers        int
	MaxConns       int
}

func NewGenerator(preset string, answers WizardAnswers) *ConfigGenerator {
	return &ConfigGenerator{
		Preset:    preset,
		Answers:   answers,
		Overrides: make(map[string]interface{}),
	}
}

func (g *ConfigGenerator) Generate() (*config.Config, error) {
	cfg := config.DefaultConfig()

	cfg.Sites = []config.SiteConfig{
		{
			Name:       "default",
			Domains:    g.Answers.Domains,
			Upstream:   g.Answers.BackendURL,
			Port:       80,
			WAFEnabled: true,
		},
	}

	switch g.Answers.TLSMode {
	case "manual":
		cfg.Sites[0].TLS = true
		cfg.Sites[0].CertFile = g.Answers.CertFile
		cfg.Sites[0].KeyFile = g.Answers.KeyFile
	case "letsencrypt":
		cfg.TLS.AutoCert.Enabled = true
	}

	if g.Answers.RedisEnabled {
		cfg.Redis.Enabled = true
		cfg.Redis.Addr = g.Answers.RedisURL
	}

	if g.Answers.PostgresEnabled {
		cfg.DB.Driver = "postgresql"
		cfg.DB.DSN = g.Answers.PostgresURL
	}

	if g.Answers.VaultEnabled {
		cfg.Secrets.Provider = "vault"
	}

	cfg.ML.Enabled = true

	g.applyRuleTemplates(cfg)

	return cfg, nil
}

func (g *ConfigGenerator) applyRuleTemplates(cfg *config.Config) {
	for _, t := range g.Answers.RuleTemplates {
		switch t {
		case "OWASP Top 10", "owasp", "1":
			cfg.Rules = append(cfg.Rules, config.RuleConfig{
				ID:          "OWASP-TOP10",
				Name:        "OWASP Top 10 Protection",
				Description: "Protection against OWASP Top 10 vulnerabilities",
				Enabled:     true,
				Severity:    "high",
				Action:      "block",
				Phase:       "request",
				Priority:    1,
			})
		case "SQL Injection", "sqli", "2":
			cfg.Rules = append(cfg.Rules, config.RuleConfig{
				ID:          "SQLI-001",
				Name:        "SQL Injection Protection",
				Description: "Blocks SQL injection attempts",
				Enabled:     true,
				Severity:    "critical",
				Action:      "block",
				Phase:       "request",
				Priority:    2,
				Field:       "body",
				Operator:    "contains",
				Value:       "union select",
				Transform:   []string{"lowercase", "remove_comments"},
			})
		case "XSS", "3":
			cfg.Rules = append(cfg.Rules, config.RuleConfig{
				ID:          "XSS-001",
				Name:        "XSS Protection",
				Description: "Blocks cross-site scripting attempts",
				Enabled:     true,
				Severity:    "high",
				Action:      "block",
				Phase:       "request",
				Priority:    3,
				Field:       "body",
				Operator:    "regex",
				Value:       "<script[^>]*>.*?</script>",
			})
		case "CSRF", "4":
			cfg.Rules = append(cfg.Rules, config.RuleConfig{
				ID:          "CSRF-001",
				Name:        "CSRF Protection",
				Description: "Blocks cross-site request forgery",
				Enabled:     true,
				Severity:    "medium",
				Action:      "block",
				Phase:       "request",
				Priority:    4,
			})
		case "Rate Limiting", "ratelimit", "5":
			cfg.RateLimit.Enabled = true
		case "Bot Blocking", "bot", "6":
			cfg.Rules = append(cfg.Rules, config.RuleConfig{
				ID:       "BOT-001",
				Name:     "Bot Blocking",
				Enabled:  true,
				Severity: "high",
				Action:   "challenge",
				Phase:    "request",
				Priority: 5,
			})
		case "Geo-Blocking", "geo", "7":
			cfg.Rules = append(cfg.Rules, config.RuleConfig{
				ID:          "GEO-001",
				Name:        "Geo Blocking",
				Description: "Block traffic from specific countries",
				Enabled:     false,
				Severity:    "medium",
				Action:      "block",
				Phase:       "request",
				Priority:    6,
			})
		case "CORS", "8":
			cfg.Rules = append(cfg.Rules, config.RuleConfig{
				ID:          "CORS-001",
				Name:        "CORS Protection",
				Description: "Enforce CORS policies",
				Enabled:     true,
				Severity:    "low",
				Action:      "monitor",
				Phase:       "request",
				Priority:    7,
			})
		}
	}
}

func (g *ConfigGenerator) GenerateDockerCompose() ([]byte, error) {
	templateStr := `version: '3.8'

services:
  fortresswaf:
    image: fortresswaf/fortresswaf:{{.Version}}
    container_name: fortresswaf
    ports:
      - "{{.ListenAddr}}"
      - "{{.AdminAddr}}"
    volumes:
      - ./config.yaml:/etc/fortresswaf/config.yaml
      - ./rules:/etc/fortresswaf/rules
    environment:
      - TZ=UTC
    restart: unless-stopped
    networks:
      - fortresswaf

{{if .RedisEnabled}}
  redis:
    image: redis:7-alpine
    container_name: fortresswaf-redis
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data
    restart: unless-stopped
    networks:
      - fortresswaf
{{end}}

{{if .PostgresEnabled}}
  postgres:
    image: postgres:16-alpine
    container_name: fortresswaf-postgres
    environment:
      POSTGRES_DB: fortresswaf
      POSTGRES_USER: fortresswaf
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD:-fortresswaf}
    volumes:
      - postgres_data:/var/lib/postgresql/data
    ports:
      - "5432:5432"
    restart: unless-stopped
    networks:
      - fortresswaf
{{end}}

  ml-engine:
    build:
      context: ./ml-engine
      dockerfile: Dockerfile
    container_name: fortresswaf-ml
    ports:
      - "9090:9090"
    environment:
      - ML_MODEL_PATH=/app/models
      - ML_INFERENCE_MODE=cpu
    restart: unless-stopped
    networks:
      - fortresswaf

volumes:
  redis_data:
  postgres_data:

networks:
  fortresswaf:
    driver: bridge
`

	tmpl, err := template.New("docker-compose").Parse(templateStr)
	if err != nil {
		return nil, err
	}

	data := map[string]interface{}{
		"Version":       "1.0.0",
		"ListenAddr":    g.Answers.ListenAddr,
		"AdminAddr":     g.Answers.AdminAddr,
		"RedisEnabled":   g.Answers.RedisEnabled,
		"PostgresEnabled": g.Answers.PostgresEnabled,
	}

	var buf []byte
	err = tmpl.Execute(&buf, data)
	if err != nil {
		return nil, err
	}

	return buf, nil
}

func (g *ConfigGenerator) SaveDockerCompose(path string) error {
	data, err := g.GenerateDockerCompose()
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func (g *ConfigGenerator) GenerateKubernetes() ([]byte, error) {
	return nil, fmt.Errorf("not implemented")
}

func (g *ConfigGenerator) SaveKubernetes(path string) error {
	data, err := g.GenerateKubernetes()
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

var _ = template.New
