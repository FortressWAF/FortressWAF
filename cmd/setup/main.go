package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/zulfff/FortressWAF/internal/config"
	"gopkg.in/yaml.v3"
)

type Wizard struct {
	reader     *bufio.Reader
	answers    *WizardAnswers
	cfg        *config.Config
	outputPath string
}

type WizardAnswers struct {
	ListenAddr      string
	AdminAddr        string
	EnableHTTP3      bool
	HTTP3Port        int
	EnableMTLS       bool
	BackendURL       string
	HealthCheck      bool
	HealthPath       string
	Domains          []string
	TLSMode          string
	CertFile         string
	KeyFile          string
	RuleTemplates    []string
	OWASPLevel       string
	RedisEnabled     bool
	RedisURL         string
	PostgresEnabled  bool
	PostgresURL      string
	VaultEnabled     bool
	SIEMEnabled      bool
	Preset           string
	Workers          int
	MaxConns         int
}

func NewWizard() *Wizard {
	return &Wizard{
		reader:  bufio.NewReader(os.Stdin),
		answers: &WizardAnswers{},
		cfg:     config.DefaultConfig(),
	}
}

func (w *Wizard) Run() error {
	w.printBanner()

	fmt.Println("\nWelcome to FortressWAF Setup Wizard!")
	fmt.Println("This wizard will help you configure FortressWAF for your environment.\n")

	if err := w.selectPreset(); err != nil {
		return err
	}

	if err := w.askNetwork(); err != nil {
		return err
	}

	if err := w.askBackend(); err != nil {
		return err
	}

	if err := w.askDomain(); err != nil {
		return err
	}

	if err := w.askRules(); err != nil {
		return err
	}

	if err := w.askIntegrations(); err != nil {
		return err
	}

	return w.generateAndSave()
}

func (w *Wizard) printBanner() {
	fmt.Println(`
██████╗ ███████╗██╗   ██╗███╗   ███╗███████╗███████╗██████╗ 
██╔══██╗██╔════╝██║   ██║████╗ ████║██╔════╝██╔════╝██╔══██╗
██║  ██║█████╗  ██║   ██║██╔████╔██║█████╗  █████╗  ██║  ██║
██║  ██║██╔══╝  ╚██╗ ██╔╝██║╚██╔╝██║██╔══╝  ██╔══╝  ██║  ██║
██████╔╝███████╗ ╚████╔╝ ██║ ╚═╝ ██║███████╗███████╗██████╔╝
╚═════╝ ╚══════╝  ╚═══╝  ╚═╝     ╚═╝╚══════╝╚══════╝╚═════╝ 

███╗   ███╗ █████╗ ██████╗  ██████╗  ██████╗ ███╗   ██╗███████╗██████╗ 
████╗ ████║██╔══██╗██╔══██╗██╔═══██╗██╔═══██╗████╗  ██║██╔════╝██╔══██╗
██╔████╔██║███████║██████╔╝██║   ██║██║   ██║██╔██╗ ██║█████╗  ██║  ██║
██║╚██╔╝██║██╔══██║██╔══██╗██║   ██║██║   ██║██║╚██╗██║██╔══╝  ██║  ██║
██║ ╚═╝ ██║██║  ██║██║  ██║╚██████╔╝╚██████╔╝██║ ╚████║███████╗██████╔╝
╚═╝     ╚═╝╚═╝  ╚═╝╚═╝  ╚═╝ ╚═════╝  ╚═════╝ ╚═╝  ╚═══╝╚══════╝╚═════╝ 
`)
}

func (w *Wizard) selectPreset() error {
	fmt.Println("\n╔═══════════════════════════════════════════════════════════╗")
	fmt.Println("║                    Setup Mode Selection                    ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════╝\n")

	fmt.Println("Select a preset configuration:")
	fmt.Println("  [1] Development  - Single node, SQLite, minimal resources")
	fmt.Println("  [2] Staging     - Single node, PostgreSQL, moderate resources")
	fmt.Println("  [3] Production  - Multi-node, PostgreSQL, Redis, HA")
	fmt.Println("  [4] Enterprise  - Multi-node, PostgreSQL, Redis, Vault, SIEM")
	fmt.Println("  [5] Custom      - Manual configuration")

	fmt.Print("\n> ")

	choice, err := w.readLine()
	if err != nil {
		return err
	}

	switch strings.TrimSpace(choice) {
	case "1":
		w.applyPreset("dev")
	case "2":
		w.applyPreset("staging")
	case "3":
		w.applyPreset("prod")
	case "4":
		w.applyPreset("enterprise")
	default:
		w.applyPreset("custom")
	}

	return nil
}

func (w *Wizard) applyPreset(preset string) {
	w.answers.Preset = preset

	switch preset {
	case "dev":
		w.answers.Workers = 1
		w.answers.MaxConns = 100
		w.answers.ListenAddr = "0.0.0.0:80"
		w.answers.AdminAddr = "0.0.0.0:8443"
		w.answers.RedisEnabled = false
		w.answers.PostgresEnabled = false
		w.answers.TLSMode = "none"
		w.answers.OWASPLevel = "basic"
	case "staging":
		w.answers.Workers = 2
		w.answers.MaxConns = 1000
		w.answers.ListenAddr = "0.0.0.0:80"
		w.answers.AdminAddr = "0.0.0.0:8443"
		w.answers.RedisEnabled = true
		w.answers.RedisURL = "redis://localhost:6379"
		w.answers.PostgresEnabled = true
		w.answers.PostgresURL = "postgres://localhost:5432/fortresswaf"
		w.answers.TLSMode = "manual"
		w.answers.OWASPLevel = "extended"
	case "prod":
		w.answers.Workers = 4
		w.answers.MaxConns = 10000
		w.answers.ListenAddr = "0.0.0.0:80"
		w.answers.AdminAddr = "0.0.0.0:8443"
		w.answers.EnableHTTP3 = true
		w.answers.HTTP3Port = 443
		w.answers.RedisEnabled = true
		w.answers.RedisURL = "redis://redis:6379"
		w.answers.PostgresEnabled = true
		w.answers.PostgresURL = "postgres://postgres:5432/fortresswaf"
		w.answers.TLSMode = "letsencrypt"
		w.answers.OWASPLevel = "maximum"
	case "enterprise":
		w.answers.Workers = 8
		w.answers.MaxConns = 50000
		w.answers.ListenAddr = "0.0.0.0:80"
		w.answers.AdminAddr = "0.0.0.0:8443"
		w.answers.EnableHTTP3 = true
		w.answers.HTTP3Port = 443
		w.answers.EnableMTLS = true
		w.answers.RedisEnabled = true
		w.answers.RedisURL = "redis://redis:6379"
		w.answers.PostgresEnabled = true
		w.answers.PostgresURL = "postgres://postgres:5432/fortresswaf"
		w.answers.VaultEnabled = true
		w.answers.SIEMEnabled = true
		w.answers.TLSMode = "letsencrypt"
		w.answers.OWASPLevel = "maximum"
	}

	fmt.Printf("\n✓ Applied %s preset\n", strings.ToUpper(preset))
}

func (w *Wizard) askNetwork() error {
	fmt.Println("\n╔═══════════════════════════════════════════════════════════╗")
	fmt.Println("║               STEP 1/6: Network Configuration            ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════╝\n")

	w.answers.ListenAddr = w.askWithDefault("Proxy Listen Address", w.answers.ListenAddr)
	w.answers.AdminAddr = w.askWithDefault("Admin API Listen Address", w.answers.AdminAddr)

	fmt.Print("Enable HTTP/3? (y/N): ")
	http3Ans, _ := w.readLine()
	w.answers.EnableHTTP3 = strings.ToLower(strings.TrimSpace(http3Ans)) == "y"

	if w.answers.EnableHTTP3 {
		w.answers.HTTP3Port = w.askIntWithDefault("HTTP/3 UDP Port", 443)
	}

	fmt.Print("Enable mTLS? (y/N): ")
	mtlsAns, _ := w.readLine()
	w.answers.EnableMTLS = strings.ToLower(strings.TrimSpace(mtlsAns)) == "y"

	return nil
}

func (w *Wizard) askBackend() error {
	fmt.Println("\n╔═══════════════════════════════════════════════════════════╗")
	fmt.Println("║               STEP 2/6: Backend Configuration            ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════╝\n")

	w.answers.BackendURL = w.askWithDefault("Backend URL", "http://localhost:3000")

	fmt.Print("Enable Health Checks? (Y/n): ")
	hcAns, _ := w.readLine()
	w.answers.HealthCheck = strings.ToLower(strings.TrimSpace(hcAns)) != "n"

	if w.answers.HealthCheck {
		w.answers.HealthPath = w.askWithDefault("Health Check Path", "/health")
	}

	return nil
}

func (w *Wizard) askDomain() error {
	fmt.Println("\n╔═══════════════════════════════════════════════════════════╗")
	fmt.Println("║               STEP 3/6: Domain & TLS                     ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════╝\n")

	fmt.Print("Enter domains (comma-separated): ")
	domains, _ := w.readLine()
	w.answers.Domains = strings.Split(domains, ",")
	for i := range w.answers.Domains {
		w.answers.Domains[i] = strings.TrimSpace(w.answers.Domains[i])
	}

	fmt.Println("\nTLS Mode:")
	fmt.Println("  [1] None (HTTP only)")
	fmt.Println("  [2] Manual (provide cert/key files)")
	fmt.Println("  [3] Let's Encrypt (auto-cert)")

	fmt.Print("\n> ")
	tlsChoice, _ := w.readLine()
	switch strings.TrimSpace(tlsChoice) {
	case "2":
		w.answers.TLSMode = "manual"
		w.answers.CertFile = w.askWithDefault("Certificate File", "/etc/fortresswaf/cert.pem")
		w.answers.KeyFile = w.askWithDefault("Key File", "/etc/fortresswaf/key.pem")
	case "3":
		w.answers.TLSMode = "letsencrypt"
	default:
		w.answers.TLSMode = "none"
	}

	return nil
}

func (w *Wizard) askRules() error {
	fmt.Println("\n╔═══════════════════════════════════════════════════════════╗")
	fmt.Println("║               STEP 4/6: Rule Selection                   ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════╝\n")

	templates := []string{
		"OWASP Top 10",
		"SQL Injection",
		"XSS",
		"CSRF",
		"Rate Limiting",
		"Bot Blocking",
		"Geo-Blocking",
		"CORS",
	}

	fmt.Println("Available rule templates (comma-separated numbers, all for all):")
	for i, t := range templates {
		fmt.Printf("  [%d] %s\n", i+1, t)
	}

	fmt.Print("\n> ")
	choice, _ := w.readLine()
	w.answers.RuleTemplates = strings.Split(choice, ",")

	fmt.Println("\nOWASP Level:")
	fmt.Println("  [1] Basic")
	fmt.Println("  [2] Extended")
	fmt.Println("  [3] Maximum")

	fmt.Print("\n> ")
	levelChoice, _ := w.readLine()
	switch strings.TrimSpace(levelChoice) {
	case "3":
		w.answers.OWASPLevel = "maximum"
	case "2":
		w.answers.OWASPLevel = "extended"
	default:
		w.answers.OWASPLevel = "basic"
	}

	return nil
}

func (w *Wizard) askIntegrations() error {
	fmt.Println("\n╔═══════════════════════════════════════════════════════════╗")
	fmt.Println("║               STEP 5/6: Integrations                     ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════╝\n")

	fmt.Print("Enable Redis? (Y/n): ")
	redisAns, _ := w.readLine()
	w.answers.RedisEnabled = strings.ToLower(strings.TrimSpace(redisAns)) != "n"

	if w.answers.RedisEnabled {
		w.answers.RedisURL = w.askWithDefault("Redis URL", "redis://localhost:6379")
	}

	fmt.Print("Enable PostgreSQL? (Y/n): ")
	pgAns, _ := w.readLine()
	w.answers.PostgresEnabled = strings.ToLower(strings.TrimSpace(pgAns)) != "n"

	if w.answers.PostgresEnabled {
		w.answers.PostgresURL = w.askWithDefault("PostgreSQL URL", "postgres://localhost:5432/fortresswaf")
	}

	fmt.Print("Enable HashiCorp Vault? (y/N): ")
	vaultAns, _ := w.readLine()
	w.answers.VaultEnabled = strings.ToLower(strings.TrimSpace(vaultAns)) == "y"

	fmt.Print("Enable SIEM Integration? (y/N): ")
	siemAns, _ := w.readLine()
	w.answers.SIEMEnabled = strings.ToLower(strings.TrimSpace(siemAns)) == "y"

	return nil
}

func (w *Wizard) generateAndSave() error {
	fmt.Println("\n╔═══════════════════════════════════════════════════════════╗")
	fmt.Println("║               STEP 6/6: Generate Config                   ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════╝\n")

	w.cfg.Sites = []config.SiteConfig{
		{
			Name:       "default",
			Domains:    w.answers.Domains,
			Upstream:   w.answers.BackendURL,
			Port:       80,
			WAFEnabled: true,
		},
	}

	if w.answers.TLSMode == "manual" {
		w.cfg.Sites[0].TLS = true
		w.cfg.Sites[0].CertFile = w.answers.CertFile
		w.cfg.Sites[0].KeyFile = w.answers.KeyFile
	}

	w.cfg.Redis.Enabled = w.answers.RedisEnabled
	w.cfg.Redis.Addr = w.answers.RedisURL

	if w.answers.PostgresEnabled {
		w.cfg.DB.Driver = "postgresql"
		w.cfg.DB.DSN = w.answers.PostgresURL
	}

	if w.answers.VaultEnabled {
		w.cfg.Secrets.Provider = "vault"
	}

	w.cfg.ML.Enabled = true
	w.cfg.ML.Endpoint = "http://127.0.0.1:9090"

	w.applyRuleTemplates()

	data, err := yaml.Marshal(w.cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	outputPath := w.outputPath
	if outputPath == "" {
		outputPath = "config.yaml"
	}

	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	fmt.Printf("✓ Configuration saved to %s\n\n", outputPath)

	fmt.Println("Next steps:")
	fmt.Println("  1. Review the generated config at:", outputPath)
	fmt.Println("  2. Run: ./fortresswaf --config", outputPath)
	fmt.Println("  3. Access the dashboard at: http://localhost:8443")

	return nil
}

func (w *Wizard) applyRuleTemplates() {
	for _, t := range w.answers.RuleTemplates {
		t = strings.TrimSpace(t)
		switch t {
		case "1", "OWASP Top 10", "owasp":
			w.cfg.Rules = append(w.cfg.Rules, config.RuleConfig{
				ID:       "OWASP-001",
				Name:     "OWASP Top 10 Protection",
				Enabled:  true,
				Severity: "high",
				Action:   "block",
				Phase:    "request",
				Priority: 1,
			})
		case "2", "SQL Injection", "sqli":
			w.cfg.Rules = append(w.cfg.Rules, config.RuleConfig{
				ID:       "SQLI-001",
				Name:     "SQL Injection Protection",
				Enabled:  true,
				Severity: "critical",
				Action:   "block",
				Phase:    "request",
				Priority: 2,
			})
		}
	}
}

func (w *Wizard) askWithDefault(prompt, defaultVal string) string {
	fmt.Printf("%s [%s]: ", prompt, defaultVal)
	val, _ := w.readLine()
	val = strings.TrimSpace(val)
	if val == "" {
		return defaultVal
	}
	return val
}

func (w *Wizard) askIntWithDefault(prompt string, defaultVal int) int {
	fmt.Printf("%s [%d]: ", prompt, defaultVal)
	val, _ := w.readLine()
	val = strings.TrimSpace(val)
	if val == "" {
		return defaultVal
	}
	i, err := strconv.Atoi(val)
	if err != nil {
		return defaultVal
	}
	return i
}

func (w *Wizard) readLine() (string, error) {
	line, err := w.reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return line, nil
}

func (w *Wizard) readPassword() string {
	fmt.Print("Password: ")
	password, _ := w.reader.ReadString('\n')
	return strings.TrimSpace(password)
}

func getwd() string {
	dir, _ := os.Getwd()
	return dir
}

func execCommand(name string, args ...string) *exec.Cmd {
	return exec.Command(name, args...)
}

func sortStrings(s []string) {
	sort.Strings(s)
}

var _ = filepath.Join
